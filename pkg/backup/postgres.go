// Package backup provides production-grade PostgreSQL backup and recovery functionality
// for OCX. Requires pg_dump and pg_restore binaries in PATH or specified via environment
// variables BACKUP_PG_DUMP and BACKUP_PG_RESTORE.
//
// Environment variables:
//
//	PG* - Standard PostgreSQL connection variables
//	BACKUP_PG_DUMP - Path to pg_dump binary (optional, defaults to PATH lookup)
//	BACKUP_PG_RESTORE - Path to pg_restore binary (optional, defaults to PATH lookup)
package backup

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	// Connection string pattern for redacting sensitive data
	connStringRegex = regexp.MustCompile(`(?i)(password|pwd)=[^&\s]*`)

	// Default timeout for backup operations
	DefaultBackupTimeout = 30 * time.Minute
)

// PostgresBackupConfig holds configuration for PostgreSQL backup operations
type PostgresBackupConfig struct {
	Logger  *log.Logger
	Timeout time.Duration
}

// DumpPostgres creates a PostgreSQL database dump using pg_dump
func DumpPostgres(ctx context.Context, connString, outFile string, compress bool) error {
	return DumpPostgresWithConfig(ctx, connString, outFile, compress, PostgresBackupConfig{
		Logger:  log.New(io.Discard, "", 0),
		Timeout: DefaultBackupTimeout,
	})
}

// DumpPostgresWithConfig creates a PostgreSQL database dump with custom configuration
func DumpPostgresWithConfig(ctx context.Context, connString, outFile string, compress bool, cfg PostgresBackupConfig) error {
	logger := cfg.Logger
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}

	// Find pg_dump binary
	pgDumpPath, err := findBinary("pg_dump", "BACKUP_PG_DUMP")
	if err != nil {
		return fmt.Errorf("failed to find pg_dump binary: %w", err)
	}

	// Create timeout context
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = DefaultBackupTimeout
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Prepare output directory
	outDir := filepath.Dir(outFile)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create temporary file in same directory for atomic operation
	tempFile := outFile + ".tmp"
	defer func() {
		if err := os.Remove(tempFile); err != nil && !os.IsNotExist(err) {
			logger.Printf("Failed to clean up temp file %s: %v", tempFile, err)
		}
	}()

	// Build pg_dump command
	args := []string{
		"--format=custom",
		"--no-owner",
		"--no-privileges",
		"--verbose",
		connString,
	}

	cmd := exec.CommandContext(ctx, pgDumpPath, args...)

	// Redact connection string in logs
	redactedCmd := strings.Join(append([]string{pgDumpPath}, redactConnString(args)...), " ")
	logger.Printf("Starting database dump: %s -> %s", redactedCmd, outFile)

	// Set up output handling
	var finalFile *os.File

	if compress {
		// Create compressed output
		finalFile, err = os.Create(tempFile + ".gz")
		if err != nil {
			return fmt.Errorf("failed to create compressed temp file: %w", err)
		}
		defer finalFile.Close()

		// Use gzip compression
		gzipCmd := exec.CommandContext(ctx, "gzip", "-c")
		gzipCmd.Stdin, err = cmd.StdoutPipe()
		if err != nil {
			return fmt.Errorf("failed to create stdout pipe: %w", err)
		}
		gzipCmd.Stdout = finalFile

		if err := gzipCmd.Start(); err != nil {
			return fmt.Errorf("failed to start gzip: %w", err)
		}
		defer func() {
			if err := gzipCmd.Wait(); err != nil && ctx.Err() == nil {
				logger.Printf("Gzip process failed: %v", err)
			}
		}()
	} else {
		// Direct output to file
		finalFile, err = os.Create(tempFile)
		if err != nil {
			return fmt.Errorf("failed to create temp file: %w", err)
		}
		defer finalFile.Close()
		cmd.Stdout = finalFile
	}

	// Capture stderr for error reporting
	stderrBuf := &limitedBuffer{limit: 8192} // Bound stderr capture
	cmd.Stderr = stderrBuf

	// Execute pg_dump
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start pg_dump: %w", err)
	}

	err = cmd.Wait()
	if err != nil {
		errMsg := stderrBuf.String()
		if errMsg != "" {
			logger.Printf("pg_dump failed: %s", errMsg)
		}
		return fmt.Errorf("pg_dump failed: %w", err)
	}

	// Ensure data is written to disk
	if err := finalFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temp file: %w", err)
	}
	finalFile.Close()

	// Calculate SHA256 checksum
	tempFileName := tempFile
	if compress {
		tempFileName = tempFile + ".gz"
	}

	checksum, err := calculateSHA256(tempFileName)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	// Write checksum sidecar file
	checksumFile := outFile + ".sha256"
	if err := os.WriteFile(checksumFile+".tmp", []byte(checksum+"  "+filepath.Base(outFile)+"\n"), 0644); err != nil {
		return fmt.Errorf("failed to write checksum file: %w", err)
	}

	// Atomic moves
	finalOutFile := outFile
	if compress && !strings.HasSuffix(outFile, ".gz") {
		finalOutFile = outFile + ".gz"
	}

	if err := os.Rename(tempFileName, finalOutFile); err != nil {
		return fmt.Errorf("failed to move dump file to final location: %w", err)
	}

	if err := os.Rename(checksumFile+".tmp", checksumFile); err != nil {
		return fmt.Errorf("failed to move checksum file to final location: %w", err)
	}

	logger.Printf("Database dump completed successfully: %s (checksum: %s)", finalOutFile, checksum)

	return nil
}

// RestorePostgres restores a PostgreSQL database from a dump file
func RestorePostgres(ctx context.Context, connString, dumpFile string) error {
	return RestorePostgresWithConfig(ctx, connString, dumpFile, PostgresBackupConfig{
		Logger:  log.New(io.Discard, "", 0),
		Timeout: DefaultBackupTimeout,
	})
}

// RestorePostgresWithConfig restores a PostgreSQL database with custom configuration
func RestorePostgresWithConfig(ctx context.Context, connString, dumpFile string, cfg PostgresBackupConfig) error {
	logger := cfg.Logger
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}

	// Find pg_restore binary
	pgRestorePath, err := findBinary("pg_restore", "BACKUP_PG_RESTORE")
	if err != nil {
		return fmt.Errorf("failed to find pg_restore binary: %w", err)
	}

	// Create timeout context
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = DefaultBackupTimeout
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Verify dump file exists
	if _, err := os.Stat(dumpFile); err != nil {
		return fmt.Errorf("dump file not found: %w", err)
	}

	// Verify SHA256 checksum if sidecar file exists
	checksumFile := dumpFile + ".sha256"
	if _, err := os.Stat(checksumFile); err == nil {
		if err := verifyChecksum(dumpFile, checksumFile); err != nil {
			return fmt.Errorf("checksum verification failed: %w", err)
		}
		logger.Printf("Checksum verification passed: %s", dumpFile)
	} else {
		logger.Printf("No checksum file found, skipping verification: %s", dumpFile)
	}

	// Build pg_restore command
	args := []string{
		"--clean",
		"--no-owner",
		"--no-privileges",
		"--verbose",
		"--dbname=" + connString,
		dumpFile,
	}

	cmd := exec.CommandContext(ctx, pgRestorePath, args...)

	// Redact connection string in logs
	redactedCmd := strings.Join(append([]string{pgRestorePath}, redactConnString(args)...), " ")
	logger.Printf("Starting database restore: %s <- %s", redactedCmd, dumpFile)

	// Capture stdout and stderr
	stdoutBuf := &limitedBuffer{limit: 8192}
	stderrBuf := &limitedBuffer{limit: 8192}
	cmd.Stdout = stdoutBuf
	cmd.Stderr = stderrBuf

	// Execute pg_restore
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start pg_restore: %w", err)
	}

	err = cmd.Wait()
	if err != nil {
		errMsg := stderrBuf.String()
		if errMsg != "" {
			logger.Printf("pg_restore failed: %s", errMsg)
		}
		return fmt.Errorf("pg_restore failed: %w", err)
	}

	logger.Printf("Database restore completed successfully: %s", dumpFile)

	// Perform post-restore health check
	if err := performHealthCheck(ctx, connString, logger); err != nil {
		logger.Printf("Post-restore health check failed: %v", err)
		// Don't fail the restore for health check failures
	}

	return nil
}

// findBinary locates a binary in PATH or via environment variable
func findBinary(name, envVar string) (string, error) {
	if path := os.Getenv(envVar); path != "" {
		if _, err := os.Stat(path); err != nil {
			return "", fmt.Errorf("binary specified in %s not found: %w", envVar, err)
		}
		return path, nil
	}

	path, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("binary %s not found in PATH (set %s to specify location): %w", name, envVar, err)
	}
	return path, nil
}

// redactConnString removes sensitive information from connection strings for logging
func redactConnString(args []string) []string {
	result := make([]string, len(args))
	for i, arg := range args {
		result[i] = connStringRegex.ReplaceAllString(arg, "${1}=***")
	}
	return result
}

// calculateSHA256 computes SHA256 hash of a file
func calculateSHA256(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// verifyChecksum verifies the SHA256 checksum of a file against its sidecar file
func verifyChecksum(filePath, checksumPath string) error {
	// Read expected checksum
	checksumData, err := os.ReadFile(checksumPath)
	if err != nil {
		return fmt.Errorf("failed to read checksum file: %w", err)
	}

	// Parse checksum (format: "hash  filename")
	checksumParts := strings.Fields(string(checksumData))
	if len(checksumParts) < 1 {
		return fmt.Errorf("invalid checksum file format")
	}
	expectedChecksum := checksumParts[0]

	// Calculate actual checksum
	actualChecksum, err := calculateSHA256(filePath)
	if err != nil {
		return fmt.Errorf("failed to calculate file checksum: %w", err)
	}

	// Compare checksums
	if actualChecksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
	}

	return nil
}

// performHealthCheck performs a basic health check on the database
func performHealthCheck(ctx context.Context, connString string, logger *log.Logger) error {
	// Real health check implementation using pgx
	logger.Printf("Performing post-restore health check")

	// Create a temporary connection for health check
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return fmt.Errorf("failed to parse connection string: %w", err)
	}

	// Set short timeout for health check
	config.MaxConns = 1
	config.MinConns = 0
	config.MaxConnLifetime = 30 * time.Second
	config.MaxConnIdleTime = 10 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to create connection pool: %w", err)
	}
	defer pool.Close()

	// Execute simple query to verify connectivity
	var result int
	err = pool.QueryRow(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		return fmt.Errorf("health check query failed: %w", err)
	}

	if result != 1 {
		return fmt.Errorf("unexpected health check result: %d", result)
	}

	logger.Printf("Health check passed")
	return nil
}

// limitedBuffer implements a buffer with a maximum size to prevent memory exhaustion
type limitedBuffer struct {
	data  []byte
	limit int
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	remaining := b.limit - len(b.data)
	if remaining <= 0 {
		return len(p), nil // Discard excess data
	}

	if len(p) > remaining {
		p = p[:remaining]
	}

	b.data = append(b.data, p...)
	return len(p), nil
}

func (b *limitedBuffer) String() string {
	return string(b.data)
}
