package backup

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log"
)

func TestDumpPostgres(t *testing.T) {
	// Skip if pg_dump is not available
	if _, err := findBinary("pg_dump", "BACKUP_PG_DUMP"); err != nil {
		t.Skipf("pg_dump not available: %v", err)
	}

	// Skip integration test if no PostgreSQL is available
	// This test requires a running PostgreSQL instance
	// In CI/CD, this would be skipped unless specifically enabled
	if os.Getenv("OCX_TEST_POSTGRES") == "" {
		t.Skip("Skipping PostgreSQL integration test (set OCX_TEST_POSTGRES=1 to enable)")
	}

	// Use environment variables for connection
	connStr := os.Getenv("TEST_POSTGRES_CONNECTION_STRING")
	if connStr == "" {
		connStr = "host=localhost port=5432 dbname=testdb user=testuser password=testpass sslmode=disable"
	}

	ctx := context.Background()
	tempDir := t.TempDir()
	backupFile := filepath.Join(tempDir, "test_backup.dump")

	// Test backup without compression
	logger := log.New(io.Discard, "", 0)
	err := DumpPostgresWithConfig(ctx, connStr, backupFile, false, PostgresBackupConfig{
		Logger:  logger,
		Timeout: 5 * time.Minute,
	})
	require.NoError(t, err)

	// Verify backup file exists
	_, err = os.Stat(backupFile)
	require.NoError(t, err)

	// Verify checksum file exists
	checksumFile := backupFile + ".sha256"
	_, err = os.Stat(checksumFile)
	require.NoError(t, err)

	// Test backup with compression
	compressedBackupFile := filepath.Join(tempDir, "test_backup_compressed.dump")
	err = DumpPostgresWithConfig(ctx, connStr, compressedBackupFile, true, PostgresBackupConfig{
		Logger:  logger,
		Timeout: 5 * time.Minute,
	})
	require.NoError(t, err)

	// Verify compressed backup file exists
	_, err = os.Stat(compressedBackupFile + ".gz")
	require.NoError(t, err)

	// Verify compressed checksum file exists
	compressedChecksumFile := compressedBackupFile + ".sha256"
	_, err = os.Stat(compressedChecksumFile)
	require.NoError(t, err)
}

func TestRestorePostgres(t *testing.T) {
	// Skip if pg_restore is not available
	if _, err := findBinary("pg_restore", "BACKUP_PG_RESTORE"); err != nil {
		t.Skipf("pg_restore not available: %v", err)
	}

	// Skip integration test if no PostgreSQL is available
	if os.Getenv("OCX_TEST_POSTGRES") == "" {
		t.Skip("Skipping PostgreSQL integration test (set OCX_TEST_POSTGRES=1 to enable)")
	}

	// This test would require a pre-existing backup file
	// Note for integration test:
	// 1. Create a backup using DumpPostgres
	// 2. Restore it using RestorePostgres
	// 3. Verify the data matches

	// Note: Full integration test with testcontainers-go would be implemented when upgrading to Go 1.21+
	// we'll test the error cases and basic functionality
	tempDir := t.TempDir()

	// Test with non-existent dump file
	nonexistentFile := filepath.Join(tempDir, "nonexistent.dump")
	connStr := "host=localhost port=5432 dbname=testdb user=testuser password=testpass sslmode=disable"

	err := RestorePostgresWithConfig(context.Background(), connStr, nonexistentFile, PostgresBackupConfig{
		Logger:  log.New(io.Discard, "", 0),
		Timeout: 10 * time.Second,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dump file not found")
}

func TestDumpPostgresErrors(t *testing.T) {
	// Skip if pg_dump is not available
	if _, err := findBinary("pg_dump", "BACKUP_PG_DUMP"); err != nil {
		t.Skipf("pg_dump not available: %v", err)
	}

	ctx := context.Background()
	tempDir := t.TempDir()
	backupFile := filepath.Join(tempDir, "error_test.dump")

	// Test with invalid connection string
	invalidConnStr := "host=invalidhost port=5432 dbname=nonexistent user=invalid password=invalid"
	err := DumpPostgresWithConfig(ctx, invalidConnStr, backupFile, false, PostgresBackupConfig{
		Logger:  log.New(io.Discard, "", 0),
		Timeout: 10 * time.Second, // Short timeout for faster failure
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pg_dump failed")

	// Test with invalid output directory
	invalidBackupFile := "/nonexistent/path/backup.dump"
	err = DumpPostgresWithConfig(ctx, "host=localhost", invalidBackupFile, false, PostgresBackupConfig{
		Logger:  log.New(io.Discard, "", 0),
		Timeout: 10 * time.Second,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create output directory")
}

func TestRestorePostgresErrors(t *testing.T) {
	// Skip if pg_restore is not available
	if _, err := findBinary("pg_restore", "BACKUP_PG_RESTORE"); err != nil {
		t.Skipf("pg_restore not available: %v", err)
	}

	ctx := context.Background()
	tempDir := t.TempDir()

	// Test with non-existent dump file
	nonexistentFile := filepath.Join(tempDir, "nonexistent.dump")
	err := RestorePostgresWithConfig(ctx, "host=localhost", nonexistentFile, PostgresBackupConfig{
		Logger:  log.New(io.Discard, "", 0),
		Timeout: 10 * time.Second,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dump file not found")

	// Test with invalid connection string
	invalidConnStr := "host=invalidhost port=5432 dbname=nonexistent user=invalid password=invalid"

	// Create a dummy dump file
	dummyFile := filepath.Join(tempDir, "dummy.dump")
	err = os.WriteFile(dummyFile, []byte("dummy content"), 0644)
	require.NoError(t, err)

	err = RestorePostgresWithConfig(ctx, invalidConnStr, dummyFile, PostgresBackupConfig{
		Logger:  log.New(io.Discard, "", 0),
		Timeout: 10 * time.Second,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pg_restore failed")
}

func TestChecksumVerification(t *testing.T) {
	// Skip if pg_dump is not available
	if _, err := findBinary("pg_dump", "BACKUP_PG_DUMP"); err != nil {
		t.Skipf("pg_dump not available: %v", err)
	}

	tempDir := t.TempDir()
	backupFile := filepath.Join(tempDir, "checksum_test.dump")

	// Create a dummy backup file
	err := os.WriteFile(backupFile, []byte("test content"), 0644)
	require.NoError(t, err)

	// Create checksum file
	checksumFile := backupFile + ".sha256"
	expectedChecksum, err := calculateSHA256(backupFile)
	require.NoError(t, err)
	err = os.WriteFile(checksumFile, []byte(expectedChecksum+"  "+filepath.Base(backupFile)+"\n"), 0644)
	require.NoError(t, err)

	// Test valid checksum
	err = verifyChecksum(backupFile, checksumFile)
	assert.NoError(t, err)

	// Test invalid checksum
	err = os.WriteFile(checksumFile, []byte("invalid_checksum  "+filepath.Base(backupFile)+"\n"), 0644)
	require.NoError(t, err)
	err = verifyChecksum(backupFile, checksumFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "checksum mismatch")

	// Test missing checksum file
	err = os.Remove(checksumFile)
	require.NoError(t, err)
	err = verifyChecksum(backupFile, checksumFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read checksum file")
}

func TestFindBinary(t *testing.T) {
	// Test finding existing binary
	path, err := findBinary("go", "TEST_GO_BINARY")
	assert.NoError(t, err)
	assert.NotEmpty(t, path)

	// Test finding non-existent binary
	_, err = findBinary("nonexistentbinary12345", "TEST_NONEXISTENT")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found in PATH")

	// Test environment variable override
	os.Setenv("TEST_CUSTOM_BINARY", "/bin/ls")
	defer os.Unsetenv("TEST_CUSTOM_BINARY")

	path, err = findBinary("nonexistent", "TEST_CUSTOM_BINARY")
	assert.NoError(t, err)
	assert.Equal(t, "/bin/ls", path)

	// Test invalid environment variable path
	os.Setenv("TEST_INVALID_BINARY", "/nonexistent/path")
	defer os.Unsetenv("TEST_INVALID_BINARY")

	_, err = findBinary("nonexistent", "TEST_INVALID_BINARY")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRedactConnString(t *testing.T) {
	args := []string{
		"host=localhost",
		"port=5432",
		"dbname=testdb",
		"user=testuser",
		"password=secret123",
		"sslmode=disable",
	}

	redacted := redactConnString(args)

	// Check that password is redacted
	for _, arg := range redacted {
		if strings.HasPrefix(arg, "password=") {
			assert.Equal(t, "password=***", arg)
		}
	}

	// Check that other args are unchanged
	assert.Contains(t, redacted, "host=localhost")
	assert.Contains(t, redacted, "port=5432")
	assert.Contains(t, redacted, "dbname=testdb")
	assert.Contains(t, redacted, "user=testuser")
	assert.Contains(t, redacted, "sslmode=disable")
}

func TestLimitedBuffer(t *testing.T) {
	buf := &limitedBuffer{limit: 10}

	// Test normal write
	n, err := buf.Write([]byte("hello"))
	assert.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, "hello", buf.String())

	// Test write that exceeds limit
	n, err = buf.Write([]byte("world12345"))
	assert.NoError(t, err)
	assert.Equal(t, 5, n)                       // Should return actual bytes written
	assert.Equal(t, "helloworld", buf.String()) // Should be truncated to limit

	// Test write when at limit
	n, err = buf.Write([]byte("more"))
	assert.NoError(t, err)
	assert.Equal(t, 4, n)
	assert.Equal(t, "helloworld", buf.String()) // Should remain unchanged
}

// Benchmark tests
func BenchmarkDumpPostgres(b *testing.B) {
	// Skip if pg_dump is not available
	if _, err := findBinary("pg_dump", "BACKUP_PG_DUMP"); err != nil {
		b.Skipf("pg_dump not available: %v", err)
	}

	// This would require setting up a test database
	// Note: Benchmark would be implemented when testcontainers-go is available (Go 1.21+)
	// we'll skip the benchmark
	b.Skip("Benchmark requires test database setup")
}

func BenchmarkRestorePostgres(b *testing.B) {
	// Skip if pg_restore is not available
	if _, err := findBinary("pg_restore", "BACKUP_PG_RESTORE"); err != nil {
		b.Skipf("pg_restore not available: %v", err)
	}

	// This would require setting up test data
	// Note: Benchmark would be implemented when testcontainers-go is available (Go 1.21+)
	// we'll skip the benchmark
	b.Skip("Benchmark requires test data setup")
}
