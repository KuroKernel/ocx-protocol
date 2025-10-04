package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// RecoveryManager manages recovery operations
type RecoveryManager struct {
	// Configuration
	config RecoveryConfig

	// Recovery state
	recoveries      map[string]*Recovery
	recoveriesMutex sync.RWMutex
}

// RecoveryConfig defines configuration for recovery operations
type RecoveryConfig struct {
	// Recovery settings
	RecoveryDir   string `json:"recovery_dir"`
	TempDir       string `json:"temp_dir"`
	MaxConcurrent int    `json:"max_concurrent"`

	// Database recovery settings
	DatabaseConfig DatabaseRecoveryConfig `json:"database_config"`

	// File recovery settings
	FileConfig FileRecoveryConfig `json:"file_config"`

	// Verification settings
	EnableVerification  bool `json:"enable_verification"`
	VerifyAfterRecovery bool `json:"verify_after_recovery"`

	// Safety settings
	RequireConfirmation        bool `json:"require_confirmation"`
	CreateBackupBeforeRecovery bool `json:"create_backup_before_recovery"`
}

// DatabaseRecoveryConfig defines database recovery configuration
type DatabaseRecoveryConfig struct {
	// Database connection
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	Username string `json:"username"`
	Password string `json:"password"`
	SSLMode  string `json:"ssl_mode"`

	// Recovery options
	DropDatabase      bool `json:"drop_database"`
	CreateDatabase    bool `json:"create_database"`
	RestoreSchema     bool `json:"restore_schema"`
	RestoreData       bool `json:"restore_data"`
	RestorePrivileges bool `json:"restore_privileges"`

	// Advanced options
	SingleTransaction bool `json:"single_transaction"`
	ExitOnError       bool `json:"exit_on_error"`
	Verbose           bool `json:"verbose"`
}

// FileRecoveryConfig defines file recovery configuration
type FileRecoveryConfig struct {
	// Destination paths
	DestinationPaths map[string]string `json:"destination_paths"` // source -> destination mapping

	// Recovery options
	OverwriteExisting   bool `json:"overwrite_existing"`
	PreserveTimestamps  bool `json:"preserve_timestamps"`
	PreservePermissions bool `json:"preserve_permissions"`
	PreserveOwnership   bool `json:"preserve_ownership"`

	// Safety options
	CreateBackupOfExisting bool   `json:"create_backup_of_existing"`
	BackupExistingTo       string `json:"backup_existing_to"`
}

// Recovery represents a recovery operation
type Recovery struct {
	ID        string                 `json:"id"`
	BackupID  string                 `json:"backup_id"`
	Type      string                 `json:"type"`   // "database", "file", "full"
	Status    string                 `json:"status"` // "running", "completed", "failed", "cancelled"
	StartTime time.Time              `json:"start_time"`
	EndTime   time.Time              `json:"end_time"`
	Duration  time.Duration          `json:"duration"`
	Progress  float64                `json:"progress"` // 0.0 to 1.0
	Metadata  map[string]interface{} `json:"metadata"`
	Error     string                 `json:"error,omitempty"`
	Warnings  []string               `json:"warnings,omitempty"`
}

// RecoveryResult represents the result of a recovery operation
type RecoveryResult struct {
	Recovery   *Recovery          `json:"recovery"`
	Success    bool               `json:"success"`
	Error      string             `json:"error,omitempty"`
	Warnings   []string           `json:"warnings,omitempty"`
	Statistics RecoveryStatistics `json:"statistics"`
}

// RecoveryStatistics represents recovery statistics
type RecoveryStatistics struct {
	FilesRecovered int           `json:"files_recovered"`
	FilesSkipped   int           `json:"files_skipped"`
	FilesFailed    int           `json:"files_failed"`
	BytesRecovered int64         `json:"bytes_recovered"`
	BytesSkipped   int64         `json:"bytes_skipped"`
	BytesFailed    int64         `json:"bytes_failed"`
	RecoveryTime   time.Duration `json:"recovery_time"`
}

// NewRecoveryManager creates a new recovery manager
func NewRecoveryManager(config RecoveryConfig) (*RecoveryManager, error) {
	rm := &RecoveryManager{
		config:     config,
		recoveries: make(map[string]*Recovery),
	}

	// Create recovery directory if it doesn't exist
	if err := os.MkdirAll(config.RecoveryDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create recovery directory: %w", err)
	}

	// Create temp directory if it doesn't exist
	if err := os.MkdirAll(config.TempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	return rm, nil
}

// RecoverFromBackup recovers from a backup
func (rm *RecoveryManager) RecoverFromBackup(backupID string, recoveryType string, options map[string]interface{}) (*RecoveryResult, error) {
	recoveryID := fmt.Sprintf("recovery-%d", time.Now().Unix())

	recovery := &Recovery{
		ID:        recoveryID,
		BackupID:  backupID,
		Type:      recoveryType,
		Status:    "running",
		StartTime: time.Now(),
		Progress:  0.0,
		Metadata:  make(map[string]interface{}),
		Warnings:  make([]string, 0),
	}

	rm.recoveriesMutex.Lock()
	rm.recoveries[recoveryID] = recovery
	rm.recoveriesMutex.Unlock()

	// Find backup file
	backupPath := rm.findBackupFile(backupID)
	if backupPath == "" {
		recovery.Status = "failed"
		recovery.Error = "backup file not found"
		recovery.EndTime = time.Now()
		recovery.Duration = recovery.EndTime.Sub(recovery.StartTime)
		return &RecoveryResult{Recovery: recovery, Success: false, Error: recovery.Error}, fmt.Errorf("backup file not found")
	}

	// Extract backup to temp directory
	tempDir := filepath.Join(rm.config.TempDir, recoveryID)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		recovery.Status = "failed"
		recovery.Error = err.Error()
		recovery.EndTime = time.Now()
		recovery.Duration = recovery.EndTime.Sub(recovery.StartTime)
		return &RecoveryResult{Recovery: recovery, Success: false, Error: err.Error()}, err
	}

	// Extract archive
	if err := rm.extractArchive(backupPath, tempDir); err != nil {
		recovery.Status = "failed"
		recovery.Error = err.Error()
		recovery.EndTime = time.Now()
		recovery.Duration = recovery.EndTime.Sub(recovery.StartTime)
		return &RecoveryResult{Recovery: recovery, Success: false, Error: err.Error()}, err
	}

	recovery.Progress = 0.3

	// Read manifest
	manifestPath := filepath.Join(tempDir, "manifest.json")
	manifest, err := rm.readManifest(manifestPath)
	if err != nil {
		recovery.Warnings = append(recovery.Warnings, fmt.Sprintf("Failed to read manifest: %v", err))
	}

	recovery.Progress = 0.4

	var results []*RecoveryResult
	var allWarnings []string

	// Recover database
	if recoveryType == "database" || recoveryType == "full" {
		dbResult, err := rm.recoverDatabase(tempDir, options)
		if err != nil {
			allWarnings = append(allWarnings, fmt.Sprintf("Database recovery failed: %v", err))
		} else {
			results = append(results, dbResult)
		}
	}

	recovery.Progress = 0.7

	// Recover files
	if recoveryType == "file" || recoveryType == "full" {
		fileResult, err := rm.recoverFiles(tempDir, options)
		if err != nil {
			allWarnings = append(allWarnings, fmt.Sprintf("File recovery failed: %v", err))
		} else {
			results = append(results, fileResult)
		}
	}

	recovery.Progress = 0.9

	// Clean up temp directory
	if err := os.RemoveAll(tempDir); err != nil {
		allWarnings = append(allWarnings, fmt.Sprintf("Failed to clean up temp directory: %v", err))
	}

	// Aggregate statistics
	statistics := RecoveryStatistics{}
	for _, result := range results {
		statistics.FilesRecovered += result.Statistics.FilesRecovered
		statistics.FilesSkipped += result.Statistics.FilesSkipped
		statistics.FilesFailed += result.Statistics.FilesFailed
		statistics.BytesRecovered += result.Statistics.BytesRecovered
		statistics.BytesSkipped += result.Statistics.BytesSkipped
		statistics.BytesFailed += result.Statistics.BytesFailed
	}

	// Update recovery status
	recovery.Status = "completed"
	recovery.EndTime = time.Now()
	recovery.Duration = recovery.EndTime.Sub(recovery.StartTime)
	recovery.Progress = 1.0
	recovery.Warnings = allWarnings

	// Verify recovery if enabled
	if rm.config.EnableVerification && rm.config.VerifyAfterRecovery {
		if err := rm.verifyRecovery(recovery, manifest); err != nil {
			recovery.Warnings = append(recovery.Warnings, fmt.Sprintf("Recovery verification failed: %v", err))
		}
	}

	return &RecoveryResult{
		Recovery:   recovery,
		Success:    true,
		Warnings:   allWarnings,
		Statistics: statistics,
	}, nil
}

// findBackupFile finds the backup file for a given backup ID
func (rm *RecoveryManager) findBackupFile(backupID string) string {
	// Look for backup file in common locations
	possiblePaths := []string{
		filepath.Join(rm.config.RecoveryDir, backupID+".tar.gz"),
		filepath.Join(rm.config.RecoveryDir, backupID+".zip"),
		filepath.Join(rm.config.RecoveryDir, backupID),
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// extractArchive extracts a compressed archive
func (rm *RecoveryManager) extractArchive(archivePath, destDir string) error {
	// Open archive file
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Determine compression type
	var reader io.Reader = file
	if strings.HasSuffix(archivePath, ".gz") {
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			return err
		}
		defer gzReader.Close()
		reader = gzReader
	}

	// Create tar reader
	tarReader := tar.NewReader(reader)

	// Extract files
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Calculate destination path
		destPath := filepath.Join(destDir, header.Name)

		// Create directory if needed
		if header.Typeflag == tar.TypeDir {
			if err := os.MkdirAll(destPath, os.FileMode(header.Mode)); err != nil {
				return err
			}
			continue
		}

		// Create parent directory
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		// Extract file
		if header.Typeflag == tar.TypeReg {
			destFile, err := os.Create(destPath)
			if err != nil {
				return err
			}

			_, err = io.Copy(destFile, tarReader)
			destFile.Close()
			if err != nil {
				return err
			}

			// Set file permissions
			if err := os.Chmod(destPath, os.FileMode(header.Mode)); err != nil {
				return err
			}
		}
	}

	return nil
}

// readManifest reads a backup manifest
func (rm *RecoveryManager) readManifest(manifestPath string) (*BackupManifest, error) {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}

	var manifest BackupManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}

// recoverDatabase recovers database from backup
func (rm *RecoveryManager) recoverDatabase(backupDir string, options map[string]interface{}) (*RecoveryResult, error) {
	recoveryID := fmt.Sprintf("db-recovery-%d", time.Now().Unix())

	recovery := &Recovery{
		ID:        recoveryID,
		Type:      "database",
		Status:    "running",
		StartTime: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	// Find database backup file (try both .sql and .dump extensions)
	var dbBackupPath string
	if _, err := os.Stat(filepath.Join(backupDir, "database.dump")); err == nil {
		dbBackupPath = filepath.Join(backupDir, "database.dump")
	} else if _, err := os.Stat(filepath.Join(backupDir, "database.sql")); err == nil {
		dbBackupPath = filepath.Join(backupDir, "database.sql")
	} else {
		recovery.Status = "failed"
		recovery.Error = "database backup file not found"
		recovery.EndTime = time.Now()
		recovery.Duration = recovery.EndTime.Sub(recovery.StartTime)
		return &RecoveryResult{Recovery: recovery, Success: false, Error: recovery.Error}, fmt.Errorf("database backup file not found")
	}

	// Build connection string from config
	connString := rm.buildConnectionString()

	// Use real pg_restore implementation
	if err := RestorePostgresWithConfig(context.Background(), connString, dbBackupPath, PostgresBackupConfig{
		Logger:  log.New(io.Discard, "", 0), // Use structured logging in production
		Timeout: 30 * time.Minute,
	}); err != nil {
		recovery.Status = "failed"
		recovery.Error = err.Error()
		recovery.EndTime = time.Now()
		recovery.Duration = recovery.EndTime.Sub(recovery.StartTime)
		return &RecoveryResult{Recovery: recovery, Success: false, Error: err.Error()}, err
	}

	// Get file size for statistics
	stat, err := os.Stat(dbBackupPath)
	if err != nil {
		// Log warning but don't fail the recovery
		stat = &fakeFileInfo{size: 0}
	}

	recovery.Status = "completed"
	recovery.EndTime = time.Now()
	recovery.Duration = recovery.EndTime.Sub(recovery.StartTime)

	return &RecoveryResult{
		Recovery: recovery,
		Success:  true,
		Statistics: RecoveryStatistics{
			FilesRecovered: 1,
			BytesRecovered: stat.Size(),
		},
	}, nil
}

// recoverFiles recovers files from backup
func (rm *RecoveryManager) recoverFiles(backupDir string, options map[string]interface{}) (*RecoveryResult, error) {
	recoveryID := fmt.Sprintf("file-recovery-%d", time.Now().Unix())

	recovery := &Recovery{
		ID:        recoveryID,
		Type:      "file",
		Status:    "running",
		StartTime: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	filesDir := filepath.Join(backupDir, "files")
	if _, err := os.Stat(filesDir); err != nil {
		recovery.Status = "failed"
		recovery.Error = "files backup directory not found"
		recovery.EndTime = time.Now()
		recovery.Duration = recovery.EndTime.Sub(recovery.StartTime)
		return &RecoveryResult{Recovery: recovery, Success: false, Error: recovery.Error}, fmt.Errorf("files backup directory not found")
	}

	statistics := RecoveryStatistics{}

	// Recover files
	if err := rm.recoverDirectory(filesDir, &statistics); err != nil {
		recovery.Status = "failed"
		recovery.Error = err.Error()
		recovery.EndTime = time.Now()
		recovery.Duration = recovery.EndTime.Sub(recovery.StartTime)
		return &RecoveryResult{Recovery: recovery, Success: false, Error: err.Error()}, err
	}

	recovery.Status = "completed"
	recovery.EndTime = time.Now()
	recovery.Duration = recovery.EndTime.Sub(recovery.StartTime)

	return &RecoveryResult{
		Recovery:   recovery,
		Success:    true,
		Statistics: statistics,
	}, nil
}

// recoverDirectory recovers a directory recursively
func (rm *RecoveryManager) recoverDirectory(sourceDir string, stats *RecoveryStatistics) error {
	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			stats.FilesFailed++
			return nil // Continue with other files
		}

		// Calculate relative path
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			stats.FilesFailed++
			return nil
		}

		// Determine destination path
		destPath := rm.getDestinationPath(relPath)
		if destPath == "" {
			stats.FilesSkipped++
			return nil
		}

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		// Check if file already exists
		if _, err := os.Stat(destPath); err == nil {
			if !rm.config.FileConfig.OverwriteExisting {
				stats.FilesSkipped++
				return nil
			}

			// Create backup of existing file if configured
			if rm.config.FileConfig.CreateBackupOfExisting {
				backupPath := filepath.Join(rm.config.FileConfig.BackupExistingTo, relPath)
				if err := os.MkdirAll(filepath.Dir(backupPath), 0755); err != nil {
					stats.FilesFailed++
					return nil
				}
				if err := rm.copyFile(destPath, backupPath); err != nil {
					stats.FilesFailed++
					return nil
				}
			}
		}

		// Recover file
		if err := rm.recoverFile(path, destPath, info); err != nil {
			stats.FilesFailed++
			stats.BytesFailed += info.Size()
			return nil
		}

		stats.FilesRecovered++
		stats.BytesRecovered += info.Size()
		return nil
	})
}

// getDestinationPath gets the destination path for a source path
func (rm *RecoveryManager) getDestinationPath(sourcePath string) string {
	// Check if there's a specific mapping
	if destPath, exists := rm.config.FileConfig.DestinationPaths[sourcePath]; exists {
		return destPath
	}

	// Use default mapping (restore to original location)
	return sourcePath
}

// recoverFile recovers a single file
func (rm *RecoveryManager) recoverFile(src, dst string, info os.FileInfo) error {
	// Create destination directory
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	// Copy file
	if err := rm.copyFile(src, dst); err != nil {
		return err
	}

	// Preserve permissions if configured
	if rm.config.FileConfig.PreservePermissions {
		if err := os.Chmod(dst, info.Mode()); err != nil {
			return err
		}
	}

	// Preserve timestamps if configured
	if rm.config.FileConfig.PreserveTimestamps {
		if err := os.Chtimes(dst, info.ModTime(), info.ModTime()); err != nil {
			return err
		}
	}

	return nil
}

// copyFile copies a file from src to dst
func (rm *RecoveryManager) copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// verifyRecovery verifies a recovery operation
func (rm *RecoveryManager) verifyRecovery(recovery *Recovery, manifest *BackupManifest) error {
	// Basic verification - check if recovered files exist
	// Implementation: do more comprehensive verification

	if manifest == nil {
		return fmt.Errorf("no manifest available for verification")
	}

	// Check if all components were recovered
	for _, component := range manifest.Components {
		if component.Success {
			if _, err := os.Stat(component.Path); err != nil {
				return fmt.Errorf("recovered file not found: %s", component.Path)
			}
		}
	}

	return nil
}

// GetRecovery returns a recovery by ID
func (rm *RecoveryManager) GetRecovery(recoveryID string) (*Recovery, error) {
	rm.recoveriesMutex.RLock()
	defer rm.recoveriesMutex.RUnlock()

	recovery, exists := rm.recoveries[recoveryID]
	if !exists {
		return nil, fmt.Errorf("recovery %s not found", recoveryID)
	}

	return recovery, nil
}

// GetRecoveries returns all recoveries with optional filtering
func (rm *RecoveryManager) GetRecoveries(filter RecoveryFilter) []*Recovery {
	rm.recoveriesMutex.RLock()
	defer rm.recoveriesMutex.RUnlock()

	var filteredRecoveries []*Recovery
	for _, recovery := range rm.recoveries {
		if rm.matchesFilter(recovery, filter) {
			filteredRecoveries = append(filteredRecoveries, recovery)
		}
	}

	return filteredRecoveries
}

// matchesFilter checks if a recovery matches the filter criteria
func (rm *RecoveryManager) matchesFilter(recovery *Recovery, filter RecoveryFilter) bool {
	if len(filter.Types) > 0 {
		found := false
		for _, recoveryType := range filter.Types {
			if recovery.Type == recoveryType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if len(filter.Statuses) > 0 {
		found := false
		for _, status := range filter.Statuses {
			if recovery.Status == status {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if filter.StartTime != nil && recovery.StartTime.Before(*filter.StartTime) {
		return false
	}

	if filter.EndTime != nil && recovery.StartTime.After(*filter.EndTime) {
		return false
	}

	return true
}

// RecoveryFilter defines filtering criteria for recoveries
type RecoveryFilter struct {
	Types     []string   `json:"types,omitempty"`
	Statuses  []string   `json:"statuses,omitempty"`
	StartTime *time.Time `json:"start_time,omitempty"`
	EndTime   *time.Time `json:"end_time,omitempty"`
}

// GetRecoveryStatistics returns recovery statistics
func (rm *RecoveryManager) GetRecoveryStatistics() RecoveryStatistics {
	rm.recoveriesMutex.RLock()
	defer rm.recoveriesMutex.RUnlock()

	stats := RecoveryStatistics{}

	for _, recovery := range rm.recoveries {
		if recovery.Status == "completed" {
			stats.FilesRecovered++
		} else if recovery.Status == "failed" {
			stats.FilesFailed++
		}
	}

	return stats
}

// buildConnectionString builds a PostgreSQL connection string from the recovery config
func (rm *RecoveryManager) buildConnectionString() string {
	cfg := rm.config.DatabaseConfig

	// Build connection string components
	parts := []string{}

	if cfg.Host != "" {
		parts = append(parts, fmt.Sprintf("host=%s", cfg.Host))
	}
	if cfg.Port > 0 {
		parts = append(parts, fmt.Sprintf("port=%d", cfg.Port))
	}
	if cfg.Database != "" {
		parts = append(parts, fmt.Sprintf("dbname=%s", cfg.Database))
	}
	if cfg.Username != "" {
		parts = append(parts, fmt.Sprintf("user=%s", cfg.Username))
	}
	if cfg.Password != "" {
		parts = append(parts, fmt.Sprintf("password=%s", cfg.Password))
	}
	if cfg.SSLMode != "" {
		parts = append(parts, fmt.Sprintf("sslmode=%s", cfg.SSLMode))
	}

	return strings.Join(parts, " ")
}

// fakeFileInfo implements os.FileInfo for cases where we can't stat a file
type fakeFileInfo struct {
	size int64
}

func (f *fakeFileInfo) Name() string       { return "" }
func (f *fakeFileInfo) Size() int64        { return f.size }
func (f *fakeFileInfo) Mode() os.FileMode  { return 0 }
func (f *fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (f *fakeFileInfo) IsDir() bool        { return false }
func (f *fakeFileInfo) Sys() interface{}   { return nil }
