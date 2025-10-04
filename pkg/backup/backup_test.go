package backup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBackupManager(t *testing.T) {
	// Create test configuration
	config := BackupConfig{
		BackupDir:        "/tmp/test-backups",
		MaxBackups:       10,
		RetentionDays:    7,
		CompressionLevel: 6,
		DatabaseConfig: DatabaseBackupConfig{
			Host:          "localhost",
			Port:          5432,
			Database:      "test_db",
			Username:      "test_user",
			Password:      "test_pass",
			SSLMode:       "disable",
			Format:        "sql",
			Compression:   true,
			IncludeSchema: true,
			IncludeData:   true,
		},
		FileConfig: FileBackupConfig{
			SourcePaths:         []string{"/tmp/test-source"},
			ExcludePatterns:     []string{"*.tmp", "*.log"},
			MaxFileSize:         1024 * 1024 * 100, // 100MB
			PreservePermissions: true,
			PreserveTimestamps:  true,
		},
		EnableScheduledBackups: false, // Disable for testing
		EnableEncryption:       false,
		EnableVerification:     true,
		VerifyAfterBackup:      true,
		EnableNotifications:    false,
	}

	bm, err := NewBackupManager(config)
	if err != nil {
		t.Fatalf("Failed to create backup manager: %v", err)
	}
	defer bm.Stop()

	// Create test source directory
	testSourceDir := "/tmp/test-source"
	if err := os.MkdirAll(testSourceDir, 0755); err != nil {
		t.Fatalf("Failed to create test source directory: %v", err)
	}
	defer os.RemoveAll(testSourceDir)

	// Create test files
	testFiles := []string{
		"test1.txt",
		"test2.txt",
		"subdir/test3.txt",
	}

	for _, file := range testFiles {
		filePath := filepath.Join(testSourceDir, file)
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			t.Fatalf("Failed to create directory for %s: %v", file, err)
		}
		if err := os.WriteFile(filePath, []byte("test content for "+file), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", file, err)
		}
	}

	// Test creating a full backup
	result, err := bm.CreateFullBackup()
	if err != nil {
		t.Fatalf("Failed to create full backup: %v", err)
	}

	if !result.Success {
		t.Errorf("Backup should be successful, got error: %s", result.Error)
	}

	if result.Backup.Status != "completed" {
		t.Errorf("Expected backup status 'completed', got '%s'", result.Backup.Status)
	}

	if result.Backup.Size <= 0 {
		t.Error("Backup size should be greater than 0")
	}

	if result.Backup.Checksum == "" {
		t.Error("Backup checksum should not be empty")
	}

	// Test getting backup
	retrievedBackup, err := bm.GetBackup(result.Backup.ID)
	if err != nil {
		t.Fatalf("Failed to get backup: %v", err)
	}

	if retrievedBackup.ID != result.Backup.ID {
		t.Error("Retrieved backup ID does not match")
	}

	// Test getting backups with filter
	filter := BackupFilter{
		Statuses: []string{"completed"},
	}
	backups := bm.GetBackups(filter)
	if len(backups) != 1 {
		t.Errorf("Expected 1 completed backup, got %d", len(backups))
	}

	// Test backup statistics
	stats := bm.GetBackupStatistics()
	if stats.FilesProcessed == 0 {
		t.Error("Backup statistics should show processed files")
	}

	// Clean up
	os.RemoveAll("/tmp/test-backups")
}

func TestRecoveryManager(t *testing.T) {
	// Create test configuration
	config := RecoveryConfig{
		RecoveryDir:   "/tmp/test-recovery",
		TempDir:       "/tmp/test-recovery-temp",
		MaxConcurrent: 2,
		DatabaseConfig: DatabaseRecoveryConfig{
			Host:           "localhost",
			Port:           5432,
			Database:       "test_db",
			Username:       "test_user",
			Password:       "test_pass",
			SSLMode:        "disable",
			DropDatabase:   false,
			CreateDatabase: true,
			RestoreSchema:  true,
			RestoreData:    true,
		},
		FileConfig: FileRecoveryConfig{
			DestinationPaths: map[string]string{
				"files/": "/tmp/test-recovery-dest/",
			},
			OverwriteExisting:   false,
			PreserveTimestamps:  true,
			PreservePermissions: true,
		},
		EnableVerification:         true,
		VerifyAfterRecovery:        true,
		RequireConfirmation:        false,
		CreateBackupBeforeRecovery: false,
	}

	rm, err := NewRecoveryManager(config)
	if err != nil {
		t.Fatalf("Failed to create recovery manager: %v", err)
	}

	// Create a test backup file
	backupDir := "/tmp/test-backup"
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatalf("Failed to create backup directory: %v", err)
	}
	defer os.RemoveAll(backupDir)

	// Create test backup content
	manifest := BackupManifest{
		ID:        "test-backup",
		Type:      "full",
		CreatedAt: time.Now(),
		Components: []BackupComponent{
			{
				Type:     "database",
				Path:     "database.sql",
				Size:     1024,
				Checksum: "test-checksum",
				Success:  true,
			},
			{
				Type:     "file",
				Path:     "files/",
				Size:     2048,
				Checksum: "test-checksum-2",
				Success:  true,
			},
		},
		Statistics: BackupStatistics{
			FilesProcessed: 2,
			BytesProcessed: 3072,
		},
	}

	// Save manifest
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal manifest: %v", err)
	}

	manifestPath := filepath.Join(backupDir, "manifest.json")
	if err := os.WriteFile(manifestPath, manifestData, 0644); err != nil {
		t.Fatalf("Failed to write manifest: %v", err)
	}

	// Create database backup file
	dbBackupPath := filepath.Join(backupDir, "database.sql")
	dbContent := "-- Test database backup\nSELECT 'test' as status;"
	if err := os.WriteFile(dbBackupPath, []byte(dbContent), 0644); err != nil {
		t.Fatalf("Failed to create database backup: %v", err)
	}

	// Create files backup directory
	filesDir := filepath.Join(backupDir, "files")
	if err := os.MkdirAll(filesDir, 0755); err != nil {
		t.Fatalf("Failed to create files directory: %v", err)
	}

	testFile := filepath.Join(filesDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create compressed archive
	archivePath := filepath.Join(config.RecoveryDir, "test-backup.tar.gz")
	if err := os.MkdirAll(config.RecoveryDir, 0755); err != nil {
		t.Fatalf("Failed to create recovery directory: %v", err)
	}

	// Create a simple backup manager to create the archive
	backupConfig := BackupConfig{
		BackupDir:        config.RecoveryDir,
		CompressionLevel: 6,
	}
	bm, err := NewBackupManager(backupConfig)
	if err != nil {
		t.Fatalf("Failed to create backup manager: %v", err)
	}
	defer bm.Stop()

	// Create archive
	if err := bm.createArchive(backupDir, archivePath); err != nil {
		t.Fatalf("Failed to create archive: %v", err)
	}

	// Test recovery
	result, err := rm.RecoverFromBackup("test-backup", "full", map[string]interface{}{
		"overwrite": true,
	})
	if err != nil {
		t.Fatalf("Failed to recover from backup: %v", err)
	}

	if !result.Success {
		t.Errorf("Recovery should be successful, got error: %s", result.Error)
	}

	if result.Recovery.Status != "completed" {
		t.Errorf("Expected recovery status 'completed', got '%s'", result.Recovery.Status)
	}

	// Test getting recovery
	retrievedRecovery, err := rm.GetRecovery(result.Recovery.ID)
	if err != nil {
		t.Fatalf("Failed to get recovery: %v", err)
	}

	if retrievedRecovery.ID != result.Recovery.ID {
		t.Error("Retrieved recovery ID does not match")
	}

	// Test recovery statistics
	stats := rm.GetRecoveryStatistics()
	if stats.FilesRecovered == 0 {
		t.Error("Recovery statistics should show recovered files")
	}

	// Clean up
	os.RemoveAll("/tmp/test-recovery")
	os.RemoveAll("/tmp/test-recovery-temp")
	os.RemoveAll("/tmp/test-recovery-dest")
	os.RemoveAll("/tmp/test-backup")
}

func TestDisasterRecoveryManager(t *testing.T) {
	// Create test configuration
	config := DisasterRecoveryConfig{
		EnableAutoRecovery:       false, // Disable for testing
		RecoveryTimeout:          time.Minute * 30,
		MaxRecoveryAttempts:      3,
		RecoveryCooldown:         time.Minute * 5,
		EnableHealthMonitoring:   false, // Disable for testing
		HealthCheckInterval:      time.Minute * 1,
		FailureThreshold:         3,
		EnableNotifications:      false,
		EnableBackupVerification: false, // Disable for testing
		VerificationInterval:     time.Hour * 24,
		EnableRecoveryTesting:    false, // Disable for testing
		TestingInterval:          time.Hour * 24 * 7,
		TestingEnvironment:       "test",
	}

	// Create mock backup and recovery managers
	backupConfig := BackupConfig{
		BackupDir: "/tmp/test-backups",
	}
	backupManager, err := NewBackupManager(backupConfig)
	if err != nil {
		t.Fatalf("Failed to create backup manager: %v", err)
	}
	defer backupManager.Stop()

	recoveryConfig := RecoveryConfig{
		RecoveryDir: "/tmp/test-recovery",
		TempDir:     "/tmp/test-recovery-temp",
	}
	recoveryManager, err := NewRecoveryManager(recoveryConfig)
	if err != nil {
		t.Fatalf("Failed to create recovery manager: %v", err)
	}

	drm, err := NewDisasterRecoveryManager(config, backupManager, recoveryManager)
	if err != nil {
		t.Fatalf("Failed to create disaster recovery manager: %v", err)
	}
	defer drm.Stop()

	// Test getting recovery plans
	plans := drm.GetRecoveryPlans()
	if len(plans) == 0 {
		t.Error("Should have default recovery plans")
	}

	// Test getting a specific recovery plan
	plan, err := drm.GetRecoveryPlan("database-failure")
	if err != nil {
		t.Fatalf("Failed to get recovery plan: %v", err)
	}

	if plan.Name != "Database Failure Recovery" {
		t.Errorf("Expected plan name 'Database Failure Recovery', got '%s'", plan.Name)
	}

	if plan.Priority != 1 {
		t.Errorf("Expected plan priority 1, got %d", plan.Priority)
	}

	if !plan.Enabled {
		t.Error("Default recovery plan should be enabled")
	}

	// Test creating a new recovery plan
	newPlan := &RecoveryPlan{
		Name:        "Test Recovery Plan",
		Description: "A test recovery plan",
		Priority:    3,
		Enabled:     true,
		Triggers: []RecoveryTrigger{
			{
				Type:      "manual",
				Condition: "test_condition",
				Duration:  time.Minute * 1,
			},
		},
		Steps: []RecoveryStep{
			{
				ID:          "test-step",
				Name:        "Test Step",
				Description: "A test recovery step",
				Type:        "custom",
				Order:       1,
				Enabled:     true,
				Command:     "echo 'test'",
				Timeout:     time.Minute * 1,
				Retries:     1,
			},
		},
		StepTimeout:  time.Minute * 5,
		TotalTimeout: time.Minute * 10,
	}

	err = drm.CreateRecoveryPlan(newPlan)
	if err != nil {
		t.Fatalf("Failed to create recovery plan: %v", err)
	}

	// Test getting the new plan
	retrievedPlan, err := drm.GetRecoveryPlan(newPlan.ID)
	if err != nil {
		t.Fatalf("Failed to get new recovery plan: %v", err)
	}

	if retrievedPlan.Name != newPlan.Name {
		t.Error("Retrieved plan name does not match")
	}

	// Test updating the plan
	retrievedPlan.Description = "Updated test recovery plan"
	err = drm.UpdateRecoveryPlan(newPlan.ID, retrievedPlan)
	if err != nil {
		t.Fatalf("Failed to update recovery plan: %v", err)
	}

	// Test getting disaster recovery status
	status := drm.GetDisasterRecoveryStatus()
	if len(status.Plans) == 0 {
		t.Error("Disaster recovery status should have plans")
	}

	if status.Config.EnableAutoRecovery != config.EnableAutoRecovery {
		t.Error("Disaster recovery status config does not match")
	}

	// Test deleting the plan
	err = drm.DeleteRecoveryPlan(newPlan.ID)
	if err != nil {
		t.Fatalf("Failed to delete recovery plan: %v", err)
	}

	// Verify plan is deleted
	_, err = drm.GetRecoveryPlan(newPlan.ID)
	if err == nil {
		t.Error("Recovery plan should be deleted")
	}

	// Clean up
	os.RemoveAll("/tmp/test-backups")
	os.RemoveAll("/tmp/test-recovery")
	os.RemoveAll("/tmp/test-recovery-temp")
}

func TestBackupConfig(t *testing.T) {
	config := BackupConfig{
		BackupDir:              "/tmp/backups",
		MaxBackups:             10,
		RetentionDays:          30,
		CompressionLevel:       6,
		EnableScheduledBackups: true,
		BackupInterval:         time.Hour * 24,
		BackupTime:             "02:00",
		EnableEncryption:       true,
		EnableVerification:     true,
		VerifyAfterBackup:      true,
		EnableNotifications:    true,
	}

	// Test configuration validation
	if config.BackupDir == "" {
		t.Error("Backup directory should not be empty")
	}

	if config.MaxBackups <= 0 {
		t.Error("Max backups should be positive")
	}

	if config.RetentionDays <= 0 {
		t.Error("Retention days should be positive")
	}

	if config.CompressionLevel < 1 || config.CompressionLevel > 9 {
		t.Error("Compression level should be between 1 and 9")
	}

	if config.BackupInterval <= 0 {
		t.Error("Backup interval should be positive")
	}

	// Test backup time format
	if _, err := time.Parse("15:04", config.BackupTime); err != nil {
		t.Errorf("Invalid backup time format: %v", err)
	}
}

func TestRecoveryConfig(t *testing.T) {
	config := RecoveryConfig{
		RecoveryDir:                "/tmp/recovery",
		TempDir:                    "/tmp/recovery-temp",
		MaxConcurrent:              3,
		EnableVerification:         true,
		VerifyAfterRecovery:        true,
		RequireConfirmation:        true,
		CreateBackupBeforeRecovery: true,
		FileConfig: FileRecoveryConfig{
			BackupExistingTo: "/tmp/backup-existing",
		},
	}

	// Test configuration validation
	if config.RecoveryDir == "" {
		t.Error("Recovery directory should not be empty")
	}

	if config.TempDir == "" {
		t.Error("Temp directory should not be empty")
	}

	if config.MaxConcurrent <= 0 {
		t.Error("Max concurrent should be positive")
	}
}

func TestDisasterRecoveryConfig(t *testing.T) {
	config := DisasterRecoveryConfig{
		EnableAutoRecovery:       true,
		RecoveryTimeout:          time.Hour * 1,
		MaxRecoveryAttempts:      3,
		RecoveryCooldown:         time.Minute * 10,
		EnableHealthMonitoring:   true,
		HealthCheckInterval:      time.Minute * 5,
		FailureThreshold:         3,
		EnableNotifications:      true,
		EnableBackupVerification: true,
		VerificationInterval:     time.Hour * 12,
		EnableRecoveryTesting:    true,
		TestingInterval:          time.Hour * 24 * 7,
		TestingEnvironment:       "staging",
	}

	// Test configuration validation
	if config.RecoveryTimeout <= 0 {
		t.Error("Recovery timeout should be positive")
	}

	if config.MaxRecoveryAttempts <= 0 {
		t.Error("Max recovery attempts should be positive")
	}

	if config.RecoveryCooldown <= 0 {
		t.Error("Recovery cooldown should be positive")
	}

	if config.HealthCheckInterval <= 0 {
		t.Error("Health check interval should be positive")
	}

	if config.FailureThreshold <= 0 {
		t.Error("Failure threshold should be positive")
	}

	if config.VerificationInterval <= 0 {
		t.Error("Verification interval should be positive")
	}

	if config.TestingInterval <= 0 {
		t.Error("Testing interval should be positive")
	}

	if config.TestingEnvironment == "" {
		t.Error("Testing environment should not be empty")
	}
}
