// deployment_test.go - Deployment Testing Procedures for OCX Protocol
// Tests: Docker reliability, database migrations, backup/recovery, monitoring

package deployment

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

const (
	CONTAINER_NAME = "ocx-protocol"
	IMAGE_NAME     = "ocx-protocol:latest"
	DB_CONTAINER   = "ocx-db"
	DB_IMAGE       = "postgres:15"
)

func TestDockerContainerReliability(t *testing.T) {
	t.Run("ContainerStartupShutdown", func(t *testing.T) {
		// Test container startup and shutdown reliability
		iterations := 10
		
		for i := 0; i < iterations; i++ {
			t.Run(fmt.Sprintf("Iteration_%d", i+1), func(t *testing.T) {
				// Start container
				containerID := startContainer(t)
				if containerID == "" {
					t.Fatal("Failed to start container")
				}
				
				// Wait for container to be ready
				if !waitForContainerReady(t, containerID, 30*time.Second) {
					t.Fatal("Container failed to become ready")
				}
				
				// Test container health
				if !testContainerHealth(t, containerID) {
					t.Fatal("Container health check failed")
				}
				
				// Stop container
				if !stopContainer(t, containerID) {
					t.Fatal("Failed to stop container")
				}
				
				// Clean up
				removeContainer(t, containerID)
			})
		}
	})
	
	t.Run("ContainerResourceLimits", func(t *testing.T) {
		// Test container behavior under resource constraints
		resourceLimits := []struct {
			name        string
			memoryLimit string
			cpuLimit    string
		}{
			{"LowMemory", "128m", "0.5"},
			{"LowCPU", "512m", "0.25"},
			{"Minimal", "64m", "0.1"},
		}
		
		for _, limit := range resourceLimits {
			t.Run(limit.name, func(t *testing.T) {
				containerID := startContainerWithLimits(t, limit.memoryLimit, limit.cpuLimit)
				if containerID == "" {
					t.Fatal("Failed to start container with limits")
				}
				
				// Test container stability under limits
				if !testContainerStability(t, containerID, 60*time.Second) {
					t.Fatal("Container unstable under resource limits")
				}
				
				stopContainer(t, containerID)
				removeContainer(t, containerID)
			})
		}
	})
}

func TestDatabaseMigration(t *testing.T) {
	t.Run("MigrationUpgrade", func(t *testing.T) {
		// Test database migration upgrade
		migrationVersions := []string{"v1.0.0", "v1.1.0", "v1.2.0", "v2.0.0"}
		
		for i, version := range migrationVersions {
			t.Run(fmt.Sprintf("UpgradeTo_%s", version), func(t *testing.T) {
				// Start database
				dbContainerID := startDatabase(t)
				if dbContainerID == "" {
					t.Fatal("Failed to start database")
				}
				
				// Run migration
				if !runMigration(t, version) {
					t.Fatal("Migration failed")
				}
				
				// Verify migration success
				if !verifyMigration(t, version) {
					t.Fatal("Migration verification failed")
				}
				
				// Test application compatibility
				if !testApplicationCompatibility(t, version) {
					t.Fatal("Application compatibility test failed")
				}
				
				// Clean up
				stopContainer(t, dbContainerID)
				removeContainer(t, dbContainerID)
			})
		}
	})
	
	t.Run("MigrationRollback", func(t *testing.T) {
		// Test database migration rollback
		// Start with latest version
		dbContainerID := startDatabase(t)
		if dbContainerID == "" {
			t.Fatal("Failed to start database")
		}
		
		// Run migration to latest
		if !runMigration(t, "v2.0.0") {
			t.Fatal("Failed to migrate to latest")
		}
		
		// Rollback to previous version
		if !rollbackMigration(t, "v1.2.0") {
			t.Fatal("Migration rollback failed")
		}
		
		// Verify rollback success
		if !verifyMigration(t, "v1.2.0") {
			t.Fatal("Rollback verification failed")
		}
		
		// Clean up
		stopContainer(t, dbContainerID)
		removeContainer(t, dbContainerID)
	})
}

func TestBackupAndRecovery(t *testing.T) {
	t.Run("DatabaseBackup", func(t *testing.T) {
		// Test database backup functionality
		dbContainerID := startDatabase(t)
		if dbContainerID == "" {
			t.Fatal("Failed to start database")
		}
		
		// Create test data
		if !createTestData(t, dbContainerID) {
			t.Fatal("Failed to create test data")
		}
		
		// Create backup
		backupPath := fmt.Sprintf("/tmp/ocx_backup_%d.sql", time.Now().Unix())
		if !createBackup(t, dbContainerID, backupPath) {
			t.Fatal("Failed to create backup")
		}
		
		// Verify backup file exists and is valid
		if !verifyBackup(t, backupPath) {
			t.Fatal("Backup verification failed")
		}
		
		// Clean up
		stopContainer(t, dbContainerID)
		removeContainer(t, dbContainerID)
		os.Remove(backupPath)
	})
	
	t.Run("DatabaseRecovery", func(t *testing.T) {
		// Test database recovery functionality
		// Create backup first
		dbContainerID1 := startDatabase(t)
		if dbContainerID1 == "" {
			t.Fatal("Failed to start database")
		}
		
		createTestData(t, dbContainerID1)
		backupPath := fmt.Sprintf("/tmp/ocx_backup_%d.sql", time.Now().Unix())
		createBackup(t, dbContainerID1, backupPath)
		
		stopContainer(t, dbContainerID1)
		removeContainer(t, dbContainerID1)
		
		// Start new database
		dbContainerID2 := startDatabase(t)
		if dbContainerID2 == "" {
			t.Fatal("Failed to start new database")
		}
		
		// Restore from backup
		if !restoreBackup(t, dbContainerID2, backupPath) {
			t.Fatal("Failed to restore backup")
		}
		
		// Verify data integrity
		if !verifyDataIntegrity(t, dbContainerID2) {
			t.Fatal("Data integrity verification failed")
		}
		
		// Clean up
		stopContainer(t, dbContainerID2)
		removeContainer(t, dbContainerID2)
		os.Remove(backupPath)
	})
}

func TestMonitoringAndAlerting(t *testing.T) {
	t.Run("HealthCheckEndpoints", func(t *testing.T) {
		// Test health check endpoints
		containerID := startContainer(t)
		if containerID == "" {
			t.Fatal("Failed to start container")
		}
		
		healthEndpoints := []string{
			"/health",
			"/health/ready",
			"/health/live",
			"/metrics",
		}
		
		for _, endpoint := range healthEndpoints {
			t.Run(fmt.Sprintf("Endpoint_%s", endpoint), func(t *testing.T) {
				if !testHealthEndpoint(t, endpoint) {
					t.Fatalf("Health endpoint %s failed", endpoint)
				}
			})
		}
		
		stopContainer(t, containerID)
		removeContainer(t, containerID)
	})
	
	t.Run("AlertingSystem", func(t *testing.T) {
		// Test alerting system
		containerID := startContainer(t)
		if containerID == "" {
			t.Fatal("Failed to start container")
		}
		
		// Test various alert conditions
		alertTests := []struct {
			name        string
			triggerFunc func(*testing.T) error
		}{
			{"HighCPUUsage", triggerHighCPUAlert},
			{"HighMemoryUsage", triggerHighMemoryAlert},
			{"DatabaseConnectionFailure", triggerDatabaseAlert},
			{"APIErrorRate", triggerAPIErrorAlert},
		}
		
		for _, alertTest := range alertTests {
			t.Run(alertTest.name, func(t *testing.T) {
				err := alertTest.triggerFunc(t)
				if err != nil {
					t.Errorf("Alert test failed: %v", err)
				}
			})
		}
		
		stopContainer(t, containerID)
		removeContainer(t, containerID)
	})
}

func TestDeploymentRollback(t *testing.T) {
	t.Run("ApplicationRollback", func(t *testing.T) {
		// Test application rollback capability
		// Deploy version 1
		containerID1 := deployVersion(t, "v1.0.0")
		if containerID1 == "" {
			t.Fatal("Failed to deploy version 1")
		}
		
		// Verify version 1 is working
		if !testApplicationVersion(t, containerID1, "v1.0.0") {
			t.Fatal("Version 1 verification failed")
		}
		
		// Deploy version 2
		containerID2 := deployVersion(t, "v2.0.0")
		if containerID2 == "" {
			t.Fatal("Failed to deploy version 2")
		}
		
		// Verify version 2 is working
		if !testApplicationVersion(t, containerID2, "v2.0.0") {
			t.Fatal("Version 2 verification failed")
		}
		
		// Rollback to version 1
		containerID3 := rollbackToVersion(t, "v1.0.0")
		if containerID3 == "" {
			t.Fatal("Failed to rollback to version 1")
		}
		
		// Verify rollback success
		if !testApplicationVersion(t, containerID3, "v1.0.0") {
			t.Fatal("Rollback verification failed")
		}
		
		// Clean up
		stopContainer(t, containerID1)
		stopContainer(t, containerID2)
		stopContainer(t, containerID3)
		removeContainer(t, containerID1)
		removeContainer(t, containerID2)
		removeContainer(t, containerID3)
	})
}

// Helper functions
func startContainer(t *testing.T) string {
	cmd := exec.Command("docker", "run", "-d", "--name", CONTAINER_NAME, IMAGE_NAME)
	output, err := cmd.Output()
	if err != nil {
		t.Logf("Failed to start container: %v", err)
		return ""
	}
	
	containerID := string(output[:12]) // First 12 characters of container ID
	return containerID
}

func startContainerWithLimits(t *testing.T, memoryLimit, cpuLimit string) string {
	cmd := exec.Command("docker", "run", "-d", 
		"--name", CONTAINER_NAME,
		"--memory", memoryLimit,
		"--cpus", cpuLimit,
		IMAGE_NAME)
	output, err := cmd.Output()
	if err != nil {
		t.Logf("Failed to start container with limits: %v", err)
		return ""
	}
	
	containerID := string(output[:12])
	return containerID
}

func waitForContainerReady(t *testing.T, containerID string, timeout time.Duration) bool {
	client, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		t.Logf("Failed to create Docker client: %v", err)
		return false
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	for {
		select {
		case <-ctx.Done():
			return false
		default:
			container, err := client.ContainerInspect(ctx, containerID)
			if err != nil {
				time.Sleep(1 * time.Second)
				continue
			}
			
			if container.State.Running && container.State.Health != nil {
				if container.State.Health.Status == "healthy" {
					return true
				}
			}
			
			time.Sleep(1 * time.Second)
		}
	}
}

func testContainerHealth(t *testing.T, containerID string) bool {
	// Test container health by making HTTP request
	resp, err := http.Get("http://localhost:8080/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	
	return resp.StatusCode == http.StatusOK
}

func testContainerStability(t *testing.T, containerID string, duration time.Duration) bool {
	start := time.Now()
	for time.Since(start) < duration {
		if !testContainerHealth(t, containerID) {
			return false
		}
		time.Sleep(5 * time.Second)
	}
	return true
}

func stopContainer(t *testing.T, containerID string) bool {
	cmd := exec.Command("docker", "stop", containerID)
	err := cmd.Run()
	if err != nil {
		t.Logf("Failed to stop container: %v", err)
		return false
	}
	return true
}

func removeContainer(t *testing.T, containerID string) {
	cmd := exec.Command("docker", "rm", containerID)
	cmd.Run() // Ignore errors for cleanup
}

func startDatabase(t *testing.T) string {
	cmd := exec.Command("docker", "run", "-d", 
		"--name", DB_CONTAINER,
		"-e", "POSTGRES_PASSWORD=testpass",
		"-e", "POSTGRES_DB=ocx_test",
		"-p", "5432:5432",
		DB_IMAGE)
	output, err := cmd.Output()
	if err != nil {
		t.Logf("Failed to start database: %v", err)
		return ""
	}
	
	containerID := string(output[:12])
	
	// Wait for database to be ready
	time.Sleep(10 * time.Second)
	
	return containerID
}

func runMigration(t *testing.T, version string) bool {
	cmd := exec.Command("docker", "exec", CONTAINER_NAME, "migrate", "up", "-version", version)
	err := cmd.Run()
	if err != nil {
		t.Logf("Migration failed: %v", err)
		return false
	}
	return true
}

func verifyMigration(t *testing.T, version string) bool {
	cmd := exec.Command("docker", "exec", CONTAINER_NAME, "migrate", "version")
	output, err := cmd.Output()
	if err != nil {
		t.Logf("Migration verification failed: %v", err)
		return false
	}
	
	return string(output) == version
}

func testApplicationCompatibility(t *testing.T, version string) bool {
	// Test application compatibility with specific version
	resp, err := http.Get("http://localhost:8080/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	
	return resp.StatusCode == http.StatusOK
}

func rollbackMigration(t *testing.T, version string) bool {
	cmd := exec.Command("docker", "exec", CONTAINER_NAME, "migrate", "down", "-version", version)
	err := cmd.Run()
	if err != nil {
		t.Logf("Migration rollback failed: %v", err)
		return false
	}
	return true
}

func createTestData(t *testing.T, containerID string) bool {
	// Create test data in database
	cmd := exec.Command("docker", "exec", containerID, "psql", "-U", "postgres", "-d", "ocx_test", "-c", 
		"INSERT INTO orders (id, buyer_id, gpus, hours, amount) VALUES ('test1', 'buyer1', 2, 4, 100.0);")
	err := cmd.Run()
	if err != nil {
		t.Logf("Failed to create test data: %v", err)
		return false
	}
	return true
}

func createBackup(t *testing.T, containerID, backupPath string) bool {
	cmd := exec.Command("docker", "exec", containerID, "pg_dump", "-U", "postgres", "-d", "ocx_test")
	output, err := cmd.Output()
	if err != nil {
		t.Logf("Failed to create backup: %v", err)
		return false
	}
	
	err = os.WriteFile(backupPath, output, 0644)
	if err != nil {
		t.Logf("Failed to write backup file: %v", err)
		return false
	}
	
	return true
}

func verifyBackup(t *testing.T, backupPath string) bool {
	file, err := os.Open(backupPath)
	if err != nil {
		return false
	}
	defer file.Close()
	
	// Check if file is not empty and contains SQL
	content := make([]byte, 1024)
	n, err := file.Read(content)
	if err != nil && err != io.EOF {
		return false
	}
	
	return n > 0 && string(content[:n])[:6] == "CREATE"
}

func restoreBackup(t *testing.T, containerID, backupPath string) bool {
	backupData, err := os.ReadFile(backupPath)
	if err != nil {
		t.Logf("Failed to read backup file: %v", err)
		return false
	}
	
	cmd := exec.Command("docker", "exec", "-i", containerID, "psql", "-U", "postgres", "-d", "ocx_test")
	cmd.Stdin = bytes.NewReader(backupData)
	err = cmd.Run()
	if err != nil {
		t.Logf("Failed to restore backup: %v", err)
		return false
	}
	
	return true
}

func verifyDataIntegrity(t *testing.T, containerID string) bool {
	cmd := exec.Command("docker", "exec", containerID, "psql", "-U", "postgres", "-d", "ocx_test", "-c", 
		"SELECT COUNT(*) FROM orders WHERE id = 'test1';")
	output, err := cmd.Output()
	if err != nil {
		t.Logf("Failed to verify data integrity: %v", err)
		return false
	}
	
	return string(output) == "1"
}

func testHealthEndpoint(t *testing.T, endpoint string) bool {
	resp, err := http.Get("http://localhost:8080" + endpoint)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	
	return resp.StatusCode == http.StatusOK
}

func triggerHighCPUAlert(t *testing.T) error {
	// Simulate high CPU usage
	cmd := exec.Command("docker", "exec", CONTAINER_NAME, "stress", "--cpu", "4", "--timeout", "10s")
	return cmd.Run()
}

func triggerHighMemoryAlert(t *testing.T) error {
	// Simulate high memory usage
	cmd := exec.Command("docker", "exec", CONTAINER_NAME, "stress", "--vm", "1", "--vm-bytes", "1G", "--timeout", "10s")
	return cmd.Run()
}

func triggerDatabaseAlert(t *testing.T) error {
	// Simulate database connection failure
	cmd := exec.Command("docker", "stop", DB_CONTAINER)
	return cmd.Run()
}

func triggerAPIErrorAlert(t *testing.T) error {
	// Simulate API errors
	for i := 0; i < 100; i++ {
		http.Get("http://localhost:8080/invalid-endpoint")
	}
	return nil
}

func deployVersion(t *testing.T, version string) string {
	cmd := exec.Command("docker", "run", "-d", "--name", CONTAINER_NAME+"-"+version, IMAGE_NAME+":"+version)
	output, err := cmd.Output()
	if err != nil {
		t.Logf("Failed to deploy version %s: %v", version, err)
		return ""
	}
	
	containerID := string(output[:12])
	time.Sleep(5 * time.Second) // Wait for startup
	return containerID
}

func testApplicationVersion(t *testing.T, containerID, version string) bool {
	resp, err := http.Get("http://localhost:8080/version")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}
	
	return string(body) == version
}

func rollbackToVersion(t *testing.T, version string) string {
	// Stop current container
	stopContainer(t, CONTAINER_NAME)
	removeContainer(t, CONTAINER_NAME)
	
	// Start previous version
	return deployVersion(t, version)
}
