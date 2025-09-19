// migrate.go — Database migrations
// go 1.22+

package store

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// Migration represents a database migration
type Migration struct {
	Version int
	Name    string
	SQL     string
}

// Migrations list all database migrations
var Migrations = []Migration{
	{
		Version: 1,
		Name:    "create_identities_table",
		SQL: `CREATE TABLE IF NOT EXISTS identities(
			party_id TEXT PRIMARY KEY,
			role TEXT NOT NULL,
			display_name TEXT NOT NULL,
			email TEXT,
			key_id TEXT NOT NULL,
			public_key TEXT NOT NULL,
			created_at TEXT NOT NULL
		);`,
	},
	{
		Version: 2,
		Name:    "create_offers_table",
		SQL: `CREATE TABLE IF NOT EXISTS offers(
			offer_id TEXT PRIMARY KEY,
			provider_id TEXT NOT NULL,
			fleet_id TEXT NOT NULL,
			unit TEXT NOT NULL,
			unit_price_amount TEXT NOT NULL,
			unit_price_currency TEXT NOT NULL,
			unit_price_scale INTEGER NOT NULL,
			min_hours INTEGER NOT NULL,
			max_hours INTEGER NOT NULL,
			min_gpus INTEGER NOT NULL,
			max_gpus INTEGER NOT NULL,
			valid_from TEXT NOT NULL,
			valid_to TEXT NOT NULL
		);`,
	},
	{
		Version: 3,
		Name:    "create_orders_table",
		SQL: `CREATE TABLE IF NOT EXISTS orders(
			order_id TEXT PRIMARY KEY,
			buyer_id TEXT NOT NULL,
			offer_id TEXT,
			requested_gpus INTEGER NOT NULL,
			hours INTEGER NOT NULL,
			budget_amount TEXT,
			budget_currency TEXT,
			budget_scale INTEGER,
			state TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
	},
	{
		Version: 4,
		Name:    "create_leases_table",
		SQL: `CREATE TABLE IF NOT EXISTS leases(
			lease_id TEXT PRIMARY KEY,
			order_id TEXT NOT NULL,
			fleet_id TEXT NOT NULL,
			assigned_gpus INTEGER NOT NULL,
			start_at TEXT NOT NULL,
			end_at TEXT,
			state TEXT NOT NULL
		);`,
	},
	{
		Version: 5,
		Name:    "create_meters_table",
		SQL: `CREATE TABLE IF NOT EXISTS meters(
			meter_id TEXT PRIMARY KEY,
			lease_id TEXT NOT NULL,
			metric_name TEXT NOT NULL,
			value REAL NOT NULL,
			timestamp TEXT NOT NULL,
			FOREIGN KEY (lease_id) REFERENCES leases(lease_id)
		);`,
	},
	{
		Version: 6,
		Name:    "create_receipts_table",
		SQL: `CREATE TABLE IF NOT EXISTS receipts(
			receipt_hash BLOB PRIMARY KEY,
			receipt_body BLOB NOT NULL,
			lease_id TEXT NOT NULL,
			artifact_hash BLOB NOT NULL,
			input_hash BLOB NOT NULL,
			cycles_used INTEGER NOT NULL,
			price_micro_units INTEGER NOT NULL,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			FOREIGN KEY (lease_id) REFERENCES leases(lease_id)
		);`,
	},
	{
		Version: 7,
		Name:    "create_receipts_indexes",
		SQL: `CREATE INDEX IF NOT EXISTS idx_receipts_lease ON receipts(lease_id);
		CREATE INDEX IF NOT EXISTS idx_receipts_artifact ON receipts(artifact_hash);
		CREATE INDEX IF NOT EXISTS idx_receipts_created ON receipts(created_at);`,
	},
}

// RunMigrations runs all pending migrations
func RunMigrations(db *sql.DB) error {
	// Create migrations table if it doesn't exist
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TEXT NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get applied migrations
	appliedVersions := make(map[int]bool)
	rows, err := db.Query("SELECT version FROM schema_migrations")
	if err != nil {
		return fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return fmt.Errorf("failed to scan migration version: %w", err)
		}
		appliedVersions[version] = true
	}

	// Apply pending migrations
	for _, migration := range Migrations {
		if !appliedVersions[migration.Version] {
			log.Printf("Applying migration %d: %s", migration.Version, migration.Name)
			
			_, err := db.Exec(migration.SQL)
			if err != nil {
				return fmt.Errorf("failed to apply migration %d (%s): %w", 
					migration.Version, migration.Name, err)
			}

			// Record migration as applied
			_, err = db.Exec(`
				INSERT INTO schema_migrations (version, name, applied_at) 
				VALUES (?, ?, datetime('now'))
			`, migration.Version, migration.Name)
			if err != nil {
				return fmt.Errorf("failed to record migration %d: %w", migration.Version, err)
			}
		}
	}

	return nil
}

// GetDatabasePath returns the database file path
func GetDatabasePath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	return filepath.Join(homeDir, ".ocx", "gateway.db")
}

// EnsureDatabaseDir ensures the database directory exists
func EnsureDatabaseDir(dbPath string) error {
	dir := filepath.Dir(dbPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return os.MkdirAll(dir, 0755)
	}
	return nil
}