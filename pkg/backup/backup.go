package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
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

// BackupManager manages backup operations
type BackupManager struct {
	// Configuration
	config BackupConfig

	// Backup state
	backups      map[string]*Backup
	backupsMutex sync.RWMutex

	// Control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// BackupConfig defines configuration for backup operations
type BackupConfig struct {
	// Storage settings
	BackupDir        string `json:"backup_dir"`
	MaxBackups       int    `json:"max_backups"`
	RetentionDays    int    `json:"retention_days"`
	CompressionLevel int    `json:"compression_level"` // 1-9, 9 = best compression

	// Database backup settings
	DatabaseConfig DatabaseBackupConfig `json:"database_config"`

	// File backup settings
	FileConfig FileBackupConfig `json:"file_config"`

	// Scheduling settings
	EnableScheduledBackups bool          `json:"enable_scheduled_backups"`
	BackupInterval         time.Duration `json:"backup_interval"`
	BackupTime             string        `json:"backup_time"` // HH:MM format

	// Encryption settings
	EnableEncryption bool   `json:"enable_encryption"`
	EncryptionKey    string `json:"encryption_key"`

	// Verification settings
	EnableVerification bool `json:"enable_verification"`
	VerifyAfterBackup  bool `json:"verify_after_backup"`

	// Notification settings
	EnableNotifications  bool     `json:"enable_notifications"`
	NotificationChannels []string `json:"notification_channels"`
}

// DatabaseBackupConfig defines database backup configuration
type DatabaseBackupConfig struct {
	// Database connection
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	Username string `json:"username"`
	Password string `json:"password"`
	SSLMode  string `json:"ssl_mode"`

	// Backup options
	Format        string   `json:"format"` // "sql", "custom", "directory", "tar"
	Compression   bool     `json:"compression"`
	IncludeSchema bool     `json:"include_schema"`
	IncludeData   bool     `json:"include_data"`
	Tables        []string `json:"tables"` // Empty means all tables
	ExcludeTables []string `json:"exclude_tables"`

	// Advanced options
	NoOwner       bool `json:"no_owner"`
	NoPrivileges  bool `json:"no_privileges"`
	NoTablespaces bool `json:"no_tablespaces"`
	Clean         bool `json:"clean"`
	Create        bool `json:"create"`
}

// FileBackupConfig defines file backup configuration
type FileBackupConfig struct {
	// Source paths
	SourcePaths []string `json:"source_paths"`

	// Exclude patterns
	ExcludePatterns []string `json:"exclude_patterns"`

	// Include patterns
	IncludePatterns []string `json:"include_patterns"`

	// File filters
	MaxFileSize       int64    `json:"max_file_size"`   // 0 = no limit
	MinFileSize       int64    `json:"min_file_size"`   // 0 = no limit
	FileExtensions    []string `json:"file_extensions"` // Empty means all extensions
	ExcludeExtensions []string `json:"exclude_extensions"`

	// Symlink handling
	FollowSymlinks   bool `json:"follow_symlinks"`
	PreserveSymlinks bool `json:"preserve_symlinks"`

	// Permissions
	PreservePermissions bool `json:"preserve_permissions"`
	PreserveOwnership   bool `json:"preserve_ownership"`
	PreserveTimestamps  bool `json:"preserve_timestamps"`
}

// Backup represents a backup operation
type Backup struct {
	ID             string                 `json:"id"`
	Type           string                 `json:"type"`   // "database", "file", "full"
	Status         string                 `json:"status"` // "running", "completed", "failed", "cancelled"
	StartTime      time.Time              `json:"start_time"`
	EndTime        time.Time              `json:"end_time"`
	Duration       time.Duration          `json:"duration"`
	Size           int64                  `json:"size"`
	Checksum       string                 `json:"checksum"`
	Path           string                 `json:"path"`
	Metadata       map[string]interface{} `json:"metadata"`
	Error          string                 `json:"error,omitempty"`
	RetentionUntil time.Time              `json:"retention_until"`
}

// BackupResult represents the result of a backup operation
type BackupResult struct {
	Backup     *Backup          `json:"backup"`
	Success    bool             `json:"success"`
	Error      string           `json:"error,omitempty"`
	Warnings   []string         `json:"warnings,omitempty"`
	Statistics BackupStatistics `json:"statistics"`
}

// BackupStatistics represents backup statistics
type BackupStatistics struct {
	FilesProcessed   int           `json:"files_processed"`
	FilesSkipped     int           `json:"files_skipped"`
	FilesFailed      int           `json:"files_failed"`
	BytesProcessed   int64         `json:"bytes_processed"`
	BytesSkipped     int64         `json:"bytes_skipped"`
	BytesFailed      int64         `json:"bytes_failed"`
	CompressionRatio float64       `json:"compression_ratio"`
	ProcessingTime   time.Duration `json:"processing_time"`
}

// NewBackupManager creates a new backup manager
func NewBackupManager(config BackupConfig) (*BackupManager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	bm := &BackupManager{
		config:  config,
		backups: make(map[string]*Backup),
		ctx:     ctx,
		cancel:  cancel,
	}

	// Create backup directory if it doesn't exist
	if err := os.MkdirAll(config.BackupDir, 0755); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Start scheduled backups if enabled
	if config.EnableScheduledBackups {
		bm.wg.Add(1)
		go bm.scheduledBackupLoop()
	}

	// Start cleanup routine
	bm.wg.Add(1)
	go bm.cleanupLoop()

	return bm, nil
}

// scheduledBackupLoop runs scheduled backups
func (bm *BackupManager) scheduledBackupLoop() {
	defer bm.wg.Done()

	// Parse backup time
	backupTime, err := time.Parse("15:04", bm.config.BackupTime)
	if err != nil {
		fmt.Printf("Invalid backup time format: %v\n", err)
		return
	}

	ticker := time.NewTicker(time.Minute) // Check every minute
	defer ticker.Stop()

	for {
		select {
		case <-bm.ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			// Check if it's time for backup
			if now.Hour() == backupTime.Hour() && now.Minute() == backupTime.Minute() {
				// Run full backup
				_, err := bm.CreateFullBackup()
				if err != nil {
					fmt.Printf("Scheduled backup failed: %v\n", err)
				}
			}
		}
	}
}

// cleanupLoop runs cleanup operations
func (bm *BackupManager) cleanupLoop() {
	defer bm.wg.Done()

	ticker := time.NewTicker(time.Hour * 24) // Run daily
	defer ticker.Stop()

	for {
		select {
		case <-bm.ctx.Done():
			return
		case <-ticker.C:
			bm.cleanupOldBackups()
		}
	}
}

// cleanupOldBackups removes old backups based on retention policy
func (bm *BackupManager) cleanupOldBackups() {
	bm.backupsMutex.Lock()
	defer bm.backupsMutex.Unlock()

	cutoffTime := time.Now().AddDate(0, 0, -bm.config.RetentionDays)

	for id, backup := range bm.backups {
		if backup.RetentionUntil.Before(cutoffTime) {
			// Remove backup file
			if err := os.Remove(backup.Path); err != nil {
				fmt.Printf("Failed to remove old backup %s: %v\n", id, err)
			}

			// Remove from map
			delete(bm.backups, id)
		}
	}
}

// CreateFullBackup creates a full system backup
func (bm *BackupManager) CreateFullBackup() (*BackupResult, error) {
	backupID := fmt.Sprintf("full-%d", time.Now().Unix())

	backup := &Backup{
		ID:             backupID,
		Type:           "full",
		Status:         "running",
		StartTime:      time.Now(),
		Metadata:       make(map[string]interface{}),
		RetentionUntil: time.Now().AddDate(0, 0, bm.config.RetentionDays),
	}

	bm.backupsMutex.Lock()
	bm.backups[backupID] = backup
	bm.backupsMutex.Unlock()

	// Create backup directory
	backupDir := filepath.Join(bm.config.BackupDir, backupID)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		backup.Status = "failed"
		backup.Error = err.Error()
		backup.EndTime = time.Now()
		backup.Duration = backup.EndTime.Sub(backup.StartTime)
		return &BackupResult{Backup: backup, Success: false, Error: err.Error()}, err
	}

	var results []*BackupResult
	var warnings []string

	// Backup database
	if bm.config.DatabaseConfig.Database != "" {
		dbResult, err := bm.createDatabaseBackup(backupDir)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("Database backup failed: %v", err))
		} else {
			results = append(results, dbResult)
		}
	}

	// Backup files
	if len(bm.config.FileConfig.SourcePaths) > 0 {
		fileResult, err := bm.createFileBackup(backupDir)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("File backup failed: %v", err))
		} else {
			results = append(results, fileResult)
		}
	}

	// Create backup manifest
	manifest := BackupManifest{
		ID:         backupID,
		Type:       "full",
		CreatedAt:  backup.StartTime,
		Components: make([]BackupComponent, 0),
		Statistics: BackupStatistics{},
	}

	// Aggregate statistics
	for _, result := range results {
		manifest.Components = append(manifest.Components, BackupComponent{
			Type:     result.Backup.Type,
			Path:     result.Backup.Path,
			Size:     result.Backup.Size,
			Checksum: result.Backup.Checksum,
			Success:  result.Success,
			Error:    result.Error,
		})

		manifest.Statistics.FilesProcessed += result.Statistics.FilesProcessed
		manifest.Statistics.FilesSkipped += result.Statistics.FilesSkipped
		manifest.Statistics.FilesFailed += result.Statistics.FilesFailed
		manifest.Statistics.BytesProcessed += result.Statistics.BytesProcessed
		manifest.Statistics.BytesSkipped += result.Statistics.BytesSkipped
		manifest.Statistics.BytesFailed += result.Statistics.BytesFailed
	}

	// Save manifest
	manifestPath := filepath.Join(backupDir, "manifest.json")
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("Failed to create manifest: %v", err))
	} else {
		if err := os.WriteFile(manifestPath, manifestData, 0644); err != nil {
			warnings = append(warnings, fmt.Sprintf("Failed to save manifest: %v", err))
		}
	}

	// Create compressed archive
	archivePath := filepath.Join(bm.config.BackupDir, backupID+".tar.gz")
	if err := bm.createArchive(backupDir, archivePath); err != nil {
		backup.Status = "failed"
		backup.Error = err.Error()
		backup.EndTime = time.Now()
		backup.Duration = backup.EndTime.Sub(backup.StartTime)
		return &BackupResult{Backup: backup, Success: false, Error: err.Error(), Warnings: warnings}, err
	}

	// Get archive size and checksum
	stat, err := os.Stat(archivePath)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("Failed to get archive stats: %v", err))
	} else {
		backup.Size = stat.Size()
	}

	checksum, err := bm.calculateChecksum(archivePath)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("Failed to calculate checksum: %v", err))
	} else {
		backup.Checksum = checksum
	}

	// Clean up temporary directory
	if err := os.RemoveAll(backupDir); err != nil {
		warnings = append(warnings, fmt.Sprintf("Failed to clean up temp directory: %v", err))
	}

	// Update backup status
	backup.Status = "completed"
	backup.EndTime = time.Now()
	backup.Duration = backup.EndTime.Sub(backup.StartTime)
	backup.Path = archivePath

	// Verify backup if enabled
	if bm.config.EnableVerification && bm.config.VerifyAfterBackup {
		if err := bm.verifyBackup(backup); err != nil {
			warnings = append(warnings, fmt.Sprintf("Backup verification failed: %v", err))
		}
	}

	return &BackupResult{
		Backup:     backup,
		Success:    true,
		Warnings:   warnings,
		Statistics: manifest.Statistics,
	}, nil
}

// createDatabaseBackup creates a database backup
func (bm *BackupManager) createDatabaseBackup(backupDir string) (*BackupResult, error) {
	backupID := fmt.Sprintf("db-%d", time.Now().Unix())

	backup := &Backup{
		ID:        backupID,
		Type:      "database",
		Status:    "running",
		StartTime: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	// Create database backup file
	dbBackupPath := filepath.Join(backupDir, "database.dump")

	// Build connection string from config
	connString := bm.buildConnectionString()

	// Use real pg_dump implementation
	compress := bm.config.DatabaseConfig.Compression
	if err := DumpPostgresWithConfig(context.Background(), connString, dbBackupPath, compress, PostgresBackupConfig{
		Logger:  log.New(io.Discard, "", 0), // Use structured logging in production
		Timeout: 30 * time.Minute,
	}); err != nil {
		backup.Status = "failed"
		backup.Error = err.Error()
		backup.EndTime = time.Now()
		backup.Duration = backup.EndTime.Sub(backup.StartTime)
		return &BackupResult{Backup: backup, Success: false, Error: err.Error()}, err
	}

	// Get file size and checksum
	stat, err := os.Stat(dbBackupPath)
	if err != nil {
		backup.Status = "failed"
		backup.Error = err.Error()
		backup.EndTime = time.Now()
		backup.Duration = backup.EndTime.Sub(backup.StartTime)
		return &BackupResult{Backup: backup, Success: false, Error: err.Error()}, err
	}

	checksum, err := bm.calculateChecksum(dbBackupPath)
	if err != nil {
		backup.Status = "failed"
		backup.Error = err.Error()
		backup.EndTime = time.Now()
		backup.Duration = backup.EndTime.Sub(backup.StartTime)
		return &BackupResult{Backup: backup, Success: false, Error: err.Error()}, err
	}

	backup.Status = "completed"
	backup.EndTime = time.Now()
	backup.Duration = backup.EndTime.Sub(backup.StartTime)
	backup.Size = stat.Size()
	backup.Checksum = checksum
	backup.Path = dbBackupPath

	return &BackupResult{
		Backup:  backup,
		Success: true,
		Statistics: BackupStatistics{
			FilesProcessed: 1,
			BytesProcessed: stat.Size(),
		},
	}, nil
}

// createFileBackup creates a file backup
func (bm *BackupManager) createFileBackup(backupDir string) (*BackupResult, error) {
	backupID := fmt.Sprintf("files-%d", time.Now().Unix())

	backup := &Backup{
		ID:        backupID,
		Type:      "file",
		Status:    "running",
		StartTime: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	filesDir := filepath.Join(backupDir, "files")
	if err := os.MkdirAll(filesDir, 0755); err != nil {
		backup.Status = "failed"
		backup.Error = err.Error()
		backup.EndTime = time.Now()
		backup.Duration = backup.EndTime.Sub(backup.StartTime)
		return &BackupResult{Backup: backup, Success: false, Error: err.Error()}, err
	}

	statistics := BackupStatistics{}

	// Process each source path
	for _, sourcePath := range bm.config.FileConfig.SourcePaths {
		if err := bm.copyDirectory(sourcePath, filesDir, &statistics); err != nil {
			statistics.FilesFailed++
			// Continue with other paths
		}
	}

	backup.Status = "completed"
	backup.EndTime = time.Now()
	backup.Duration = backup.EndTime.Sub(backup.StartTime)
	backup.Path = filesDir

	return &BackupResult{
		Backup:     backup,
		Success:    true,
		Statistics: statistics,
	}, nil
}

// copyDirectory copies a directory recursively
func (bm *BackupManager) copyDirectory(src, dst string, stats *BackupStatistics) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			stats.FilesFailed++
			return nil // Continue with other files
		}

		// Check if file should be excluded
		if bm.shouldExcludeFile(path, info) {
			stats.FilesSkipped++
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Calculate relative path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			stats.FilesFailed++
			return nil
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		// Copy file
		if err := bm.copyFile(path, dstPath, info); err != nil {
			stats.FilesFailed++
			stats.BytesFailed += info.Size()
			return nil
		}

		stats.FilesProcessed++
		stats.BytesProcessed += info.Size()
		return nil
	})
}

// shouldExcludeFile checks if a file should be excluded
func (bm *BackupManager) shouldExcludeFile(path string, info os.FileInfo) bool {
	// Check file size limits
	if bm.config.FileConfig.MaxFileSize > 0 && info.Size() > bm.config.FileConfig.MaxFileSize {
		return true
	}
	if bm.config.FileConfig.MinFileSize > 0 && info.Size() < bm.config.FileConfig.MinFileSize {
		return true
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(path))
	if len(bm.config.FileConfig.ExcludeExtensions) > 0 {
		for _, excludeExt := range bm.config.FileConfig.ExcludeExtensions {
			if ext == strings.ToLower(excludeExt) {
				return true
			}
		}
	}

	// Check include patterns
	if len(bm.config.FileConfig.IncludePatterns) > 0 {
		matched := false
		for _, pattern := range bm.config.FileConfig.IncludePatterns {
			if matched, _ := filepath.Match(pattern, path); matched {
				matched = true
				break
			}
		}
		if !matched {
			return true
		}
	}

	// Check exclude patterns
	for _, pattern := range bm.config.FileConfig.ExcludePatterns {
		if matched, _ := filepath.Match(pattern, path); matched {
			return true
		}
	}

	return false
}

// copyFile copies a single file
func (bm *BackupManager) copyFile(src, dst string, info os.FileInfo) error {
	// Create destination directory
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Create destination file
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// Copy file contents
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	// Preserve permissions if configured
	if bm.config.FileConfig.PreservePermissions {
		if err := os.Chmod(dst, info.Mode()); err != nil {
			return err
		}
	}

	// Preserve timestamps if configured
	if bm.config.FileConfig.PreserveTimestamps {
		if err := os.Chtimes(dst, info.ModTime(), info.ModTime()); err != nil {
			return err
		}
	}

	return nil
}

// createArchive creates a compressed archive
func (bm *BackupManager) createArchive(sourceDir, archivePath string) error {
	// Create archive file
	file, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create gzip writer
	gzWriter, err := gzip.NewWriterLevel(file, bm.config.CompressionLevel)
	if err != nil {
		return err
	}
	defer gzWriter.Close()

	// Create tar writer
	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	// Walk through source directory
	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate relative path
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relPath

		// Write header
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		// Write file content if it's a regular file
		if info.Mode().IsRegular() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = io.Copy(tarWriter, file)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// calculateChecksum calculates SHA256 checksum of a file
func (bm *BackupManager) calculateChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
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

// verifyBackup verifies a backup
func (bm *BackupManager) verifyBackup(backup *Backup) error {
	// Check if backup file exists
	if _, err := os.Stat(backup.Path); err != nil {
		return fmt.Errorf("backup file not found: %w", err)
	}

	// Verify checksum
	calculatedChecksum, err := bm.calculateChecksum(backup.Path)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	if calculatedChecksum != backup.Checksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", backup.Checksum, calculatedChecksum)
	}

	return nil
}

// GetBackup returns a backup by ID
func (bm *BackupManager) GetBackup(backupID string) (*Backup, error) {
	bm.backupsMutex.RLock()
	defer bm.backupsMutex.RUnlock()

	backup, exists := bm.backups[backupID]
	if !exists {
		return nil, fmt.Errorf("backup %s not found", backupID)
	}

	return backup, nil
}

// GetBackups returns all backups with optional filtering
func (bm *BackupManager) GetBackups(filter BackupFilter) []*Backup {
	bm.backupsMutex.RLock()
	defer bm.backupsMutex.RUnlock()

	var filteredBackups []*Backup
	for _, backup := range bm.backups {
		if bm.matchesFilter(backup, filter) {
			filteredBackups = append(filteredBackups, backup)
		}
	}

	return filteredBackups
}

// matchesFilter checks if a backup matches the filter criteria
func (bm *BackupManager) matchesFilter(backup *Backup, filter BackupFilter) bool {
	if len(filter.Types) > 0 {
		found := false
		for _, backupType := range filter.Types {
			if backup.Type == backupType {
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
			if backup.Status == status {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if filter.StartTime != nil && backup.StartTime.Before(*filter.StartTime) {
		return false
	}

	if filter.EndTime != nil && backup.StartTime.After(*filter.EndTime) {
		return false
	}

	return true
}

// BackupFilter defines filtering criteria for backups
type BackupFilter struct {
	Types     []string   `json:"types,omitempty"`
	Statuses  []string   `json:"statuses,omitempty"`
	StartTime *time.Time `json:"start_time,omitempty"`
	EndTime   *time.Time `json:"end_time,omitempty"`
}

// BackupManifest represents a backup manifest
type BackupManifest struct {
	ID         string            `json:"id"`
	Type       string            `json:"type"`
	CreatedAt  time.Time         `json:"created_at"`
	Components []BackupComponent `json:"components"`
	Statistics BackupStatistics  `json:"statistics"`
}

// BackupComponent represents a component of a backup
type BackupComponent struct {
	Type     string `json:"type"`
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	Checksum string `json:"checksum"`
	Success  bool   `json:"success"`
	Error    string `json:"error,omitempty"`
}

// GetBackupStatistics returns backup statistics
func (bm *BackupManager) GetBackupStatistics() BackupStatistics {
	bm.backupsMutex.RLock()
	defer bm.backupsMutex.RUnlock()

	stats := BackupStatistics{}

	for _, backup := range bm.backups {
		if backup.Status == "completed" {
			stats.FilesProcessed++
			stats.BytesProcessed += backup.Size
		} else if backup.Status == "failed" {
			stats.FilesFailed++
		}
	}

	return stats
}

// buildConnectionString builds a PostgreSQL connection string from the database config
func (bm *BackupManager) buildConnectionString() string {
	cfg := bm.config.DatabaseConfig

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

// Stop stops the backup manager
func (bm *BackupManager) Stop() {
	bm.cancel()
	bm.wg.Wait()
}
