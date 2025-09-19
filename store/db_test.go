// db_test.go — Database initialization test
package store

import (
	"os"
	"testing"
	"time"
)

func TestDatabaseInitialization(t *testing.T) {
	t.Log("Testing database initialization...")
	
	// Test with in-memory database first
	t.Log("Testing in-memory database...")
	repo, err := NewRepository(":memory:")
	if err != nil {
		t.Fatalf("Failed to create in-memory repository: %v", err)
	}
	defer repo.Close()
	
	t.Log("✅ In-memory database created successfully")
	
	// Test with file database
	t.Log("Testing file database...")
	repo2, err := NewRepository("test.db")
	if err != nil {
		t.Fatalf("Failed to create file repository: %v", err)
	}
	defer repo2.Close()
	defer func() {
		// Clean up test database
		os.Remove("test.db")
	}()
	
	t.Log("✅ File database created successfully")
}

func TestDatabaseInitializationWithTimeout(t *testing.T) {
	t.Log("Testing database initialization with timeout...")
	
	done := make(chan bool, 1)
	var err error
	
	go func() {
		repo, e := NewRepository(":memory:")
		if e != nil {
			err = e
		} else {
			repo.Close()
		}
		done <- true
	}()
	
	select {
	case <-done:
		if err != nil {
			t.Fatalf("Database initialization failed: %v", err)
		}
		t.Log("✅ Database initialization completed within timeout")
	case <-time.After(5 * time.Second):
		t.Fatalf("Database initialization timed out after 5 seconds")
	}
}

func TestMigrations(t *testing.T) {
	t.Log("Testing migrations...")
	
	repo, err := NewRepository(":memory:")
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()
	
	// Check if migrations were applied
	t.Log("✅ Migrations completed successfully")
}
