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
			record_id TEXT PRIMARY KEY,
			lease_id TEXT NOT NULL,
			timestamp TEXT NOT NULL,
			gpu_hours INTEGER NOT NULL,
			egress_gb INTEGER NOT NULL,
			cpu_hours INTEGER NOT NULL,
			notes TEXT
		);`,
	},
	{
		Version: 6,
		Name:    "create_invoices_table",
		SQL: `CREATE TABLE IF NOT EXISTS invoices(
			invoice_id TEXT PRIMARY KEY,
			lease_id TEXT NOT NULL,
			issuer_id TEXT NOT NULL,
			recipient_id TEXT NOT NULL,
			total_amount TEXT NOT NULL,
			total_currency TEXT NOT NULL,
			total_scale INTEGER NOT NULL,
			state TEXT NOT NULL,
			issued_at TEXT NOT NULL,
			due_at TEXT NOT NULL
		);`,
	},
	{
		Version: 7,
		Name:    "create_migrations_table",
		SQL: `CREATE TABLE IF NOT EXISTS migrations(
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TEXT NOT NULL
		);`,
	},
}

// RunMigrations runs all pending migrations
func RunMigrations(db *sql.DB) error {
	// Create migrations table if it doesn't exist
	if _, err := db.Exec(Migrations[6].SQL); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get current version
	var currentVersion int
	err := db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM migrations").Scan(&currentVersion)
	if err != nil {
		return fmt.Errorf("failed to get current migration version: %w", err)
	}

	// Run pending migrations
	for _, migration := range Migrations {
		if migration.Version > currentVersion {
			log.Printf("Running migration %d: %s", migration.Version, migration.Name)
			
			if _, err := db.Exec(migration.SQL); err != nil {
				return fmt.Errorf("failed to run migration %d: %w", migration.Version, err)
			}

			// Record migration
			_, err = db.Exec("INSERT INTO migrations (version, name, applied_at) VALUES (?, ?, datetime('now'))",
				migration.Version, migration.Name)
			if err != nil {
				return fmt.Errorf("failed to record migration %d: %w", migration.Version, err)
			}

			log.Printf("Migration %d completed successfully", migration.Version)
		}
	}

	return nil
}

// GetDatabasePath returns the database path from environment
func GetDatabasePath() string {
	dbPath := os.Getenv("OCX_DB")
	if dbPath == "" {
		dbPath = "sqlite:///ocx.db"
	}
	
	// Extract path from sqlite:///path
	if len(dbPath) > 10 && dbPath[:10] == "sqlite:///" {
		return dbPath[10:]
	}
	
	return "ocx.db"
}

// EnsureDatabaseDir ensures the database directory exists
func EnsureDatabaseDir(dbPath string) error {
	dir := filepath.Dir(dbPath)
	if dir != "." {
		return os.MkdirAll(dir, 0755)
	}
	return nil
}
