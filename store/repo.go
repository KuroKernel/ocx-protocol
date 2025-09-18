// repo.go — Repository layer for data persistence
// go 1.22+

package store

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Repository handles all database operations
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new repository instance
func NewRepository(dbPath string) (*Repository, error) {
	// Ensure database directory exists
	if err := EnsureDatabaseDir(dbPath); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Run migrations
	if err := RunMigrations(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return &Repository{db: db}, nil
}

// Close closes the database connection
func (r *Repository) Close() error {
	return r.db.Close()
}

// Ping checks database connectivity
func (r *Repository) Ping() error {
	return r.db.Ping()
}

// Identity operations
func (r *Repository) CreateIdentity(partyID, role, displayName, email, keyID, publicKey string) error {
	_, err := r.db.Exec(`
		INSERT INTO identities (party_id, role, display_name, email, key_id, public_key, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, partyID, role, displayName, email, keyID, publicKey, time.Now().Format(time.RFC3339))
	return err
}

func (r *Repository) GetIdentity(partyID string) (*Identity, error) {
	var identity Identity
	err := r.db.QueryRow(`
		SELECT party_id, role, display_name, email, key_id, public_key, created_at
		FROM identities WHERE party_id = ?
	`, partyID).Scan(&identity.PartyID, &identity.Role, &identity.DisplayName, 
		&identity.Email, &identity.KeyID, &identity.PublicKey, &identity.CreatedAt)
	
	if err != nil {
		return nil, err
	}
	return &identity, nil
}

func (r *Repository) ListIdentities() ([]*Identity, error) {
	rows, err := r.db.Query(`
		SELECT party_id, role, display_name, email, key_id, public_key, created_at
		FROM identities ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var identities []*Identity
	for rows.Next() {
		var identity Identity
		err := rows.Scan(&identity.PartyID, &identity.Role, &identity.DisplayName,
			&identity.Email, &identity.KeyID, &identity.PublicKey, &identity.CreatedAt)
		if err != nil {
			return nil, err
		}
		identities = append(identities, &identity)
	}
	return identities, nil
}

// Offer operations
func (r *Repository) CreateOffer(offerID, providerID, fleetID, unit string, unitPriceAmount, unitPriceCurrency string, unitPriceScale int, minHours, maxHours, minGPUs, maxGPUs int, validFrom, validTo time.Time) error {
	_, err := r.db.Exec(`
		INSERT INTO offers (offer_id, provider_id, fleet_id, unit, unit_price_amount, unit_price_currency, unit_price_scale, min_hours, max_hours, min_gpus, max_gpus, valid_from, valid_to)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, offerID, providerID, fleetID, unit, unitPriceAmount, unitPriceCurrency, unitPriceScale, minHours, maxHours, minGPUs, maxGPUs, validFrom.Format(time.RFC3339), validTo.Format(time.RFC3339))
	return err
}

func (r *Repository) GetOffer(offerID string) (*Offer, error) {
	var offer Offer
	err := r.db.QueryRow(`
		SELECT offer_id, provider_id, fleet_id, unit, unit_price_amount, unit_price_currency, unit_price_scale, min_hours, max_hours, min_gpus, max_gpus, valid_from, valid_to
		FROM offers WHERE offer_id = ?
	`, offerID).Scan(&offer.OfferID, &offer.ProviderID, &offer.FleetID, &offer.Unit, &offer.UnitPriceAmount, &offer.UnitPriceCurrency, &offer.UnitPriceScale, &offer.MinHours, &offer.MaxHours, &offer.MinGPUs, &offer.MaxGPUs, &offer.ValidFrom, &offer.ValidTo)
	
	if err != nil {
		return nil, err
	}
	return &offer, nil
}

func (r *Repository) ListOffers() ([]*Offer, error) {
	rows, err := r.db.Query(`
		SELECT offer_id, provider_id, fleet_id, unit, unit_price_amount, unit_price_currency, unit_price_scale, min_hours, max_hours, min_gpus, max_gpus, valid_from, valid_to
		FROM offers ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var offers []*Offer
	for rows.Next() {
		var offer Offer
		err := rows.Scan(&offer.OfferID, &offer.ProviderID, &offer.FleetID, &offer.Unit, &offer.UnitPriceAmount, &offer.UnitPriceCurrency, &offer.UnitPriceScale, &offer.MinHours, &offer.MaxHours, &offer.MinGPUs, &offer.MaxGPUs, &offer.ValidFrom, &offer.ValidTo)
		if err != nil {
			return nil, err
		}
		offers = append(offers, &offer)
	}
	return offers, nil
}

// Order operations
func (r *Repository) CreateOrder(orderID, buyerID, offerID string, requestedGPUs, hours int, budgetAmount, budgetCurrency string, budgetScale int, state string) error {
	now := time.Now().Format(time.RFC3339)
	_, err := r.db.Exec(`
		INSERT INTO orders (order_id, buyer_id, offer_id, requested_gpus, hours, budget_amount, budget_currency, budget_scale, state, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, orderID, buyerID, offerID, requestedGPUs, hours, budgetAmount, budgetCurrency, budgetScale, state, now, now)
	return err
}

func (r *Repository) UpdateOrderState(orderID, state string) error {
	_, err := r.db.Exec(`
		UPDATE orders SET state = ?, updated_at = ? WHERE order_id = ?
	`, state, time.Now().Format(time.RFC3339), orderID)
	return err
}

func (r *Repository) ListOrders() ([]*Order, error) {
	rows, err := r.db.Query(`
		SELECT order_id, buyer_id, offer_id, requested_gpus, hours, budget_amount, budget_currency, budget_scale, state, created_at, updated_at
		FROM orders ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []*Order
	for rows.Next() {
		var order Order
		err := rows.Scan(&order.OrderID, &order.BuyerID, &order.OfferID, &order.RequestedGPUs, &order.Hours, &order.BudgetAmount, &order.BudgetCurrency, &order.BudgetScale, &order.State, &order.CreatedAt, &order.UpdatedAt)
		if err != nil {
			return nil, err
		}
		orders = append(orders, &order)
	}
	return orders, nil
}

// Lease operations
func (r *Repository) CreateLease(leaseID, orderID, fleetID string, assignedGPUs int, startAt, endAt time.Time, state string) error {
	var endAtStr *string
	if !endAt.IsZero() {
		endAtStr = &[]string{endAt.Format(time.RFC3339)}[0]
	}
	
	_, err := r.db.Exec(`
		INSERT INTO leases (lease_id, order_id, fleet_id, assigned_gpus, start_at, end_at, state)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, leaseID, orderID, fleetID, assignedGPUs, startAt.Format(time.RFC3339), endAtStr, state)
	return err
}

func (r *Repository) UpdateLeaseState(leaseID, state string) error {
	_, err := r.db.Exec(`
		UPDATE leases SET state = ? WHERE lease_id = ?
	`, state, leaseID)
	return err
}

func (r *Repository) GetLease(leaseID string) (*Lease, error) {
	var lease Lease
	var endAtStr *string
	err := r.db.QueryRow(`
		SELECT lease_id, order_id, fleet_id, assigned_gpus, start_at, end_at, state
		FROM leases WHERE lease_id = ?
	`, leaseID).Scan(&lease.LeaseID, &lease.OrderID, &lease.FleetID, &lease.AssignedGPUs, &lease.StartAt, &lease.EndAt, &endAtStr, &lease.State)
	
	if err != nil {
		return nil, err
	}
	
	if endAtStr != nil {
		if endAt, err := time.Parse(time.RFC3339, *endAtStr); err == nil {
			lease.EndAt = &endAt
		}
	}
	
	return &lease, nil
}

func (r *Repository) ListLeases() ([]*Lease, error) {
	rows, err := r.db.Query(`
		SELECT lease_id, order_id, fleet_id, assigned_gpus, start_at, end_at, state
		FROM leases ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var leases []*Lease
	for rows.Next() {
		var lease Lease
		var endAtStr *string
		err := rows.Scan(&lease.LeaseID, &lease.OrderID, &lease.FleetID, &lease.AssignedGPUs, &lease.StartAt, &lease.EndAt, &endAtStr, &lease.State)
		if err != nil {
			return nil, err
		}
		
		if endAtStr != nil {
			if endAt, err := time.Parse(time.RFC3339, *endAtStr); err == nil {
				lease.EndAt = &endAt
			}
		}
		
		leases = append(leases, &lease)
	}
	return leases, nil
}

// Data structures for repository
type Identity struct {
	PartyID     string `json:"party_id"`
	Role        string `json:"role"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	KeyID       string `json:"key_id"`
	PublicKey   string `json:"public_key"`
	CreatedAt   string `json:"created_at"`
}

type Offer struct {
	OfferID         string `json:"offer_id"`
	ProviderID      string `json:"provider_id"`
	FleetID         string `json:"fleet_id"`
	Unit            string `json:"unit"`
	UnitPriceAmount string `json:"unit_price_amount"`
	UnitPriceCurrency string `json:"unit_price_currency"`
	UnitPriceScale  int    `json:"unit_price_scale"`
	MinHours        int    `json:"min_hours"`
	MaxHours        int    `json:"max_hours"`
	MinGPUs         int    `json:"min_gpus"`
	MaxGPUs         int    `json:"max_gpus"`
	ValidFrom       string `json:"valid_from"`
	ValidTo         string `json:"valid_to"`
}

type Order struct {
	OrderID       string `json:"order_id"`
	BuyerID       string `json:"buyer_id"`
	OfferID       string `json:"offer_id"`
	RequestedGPUs int    `json:"requested_gpus"`
	Hours         int    `json:"hours"`
	BudgetAmount  string `json:"budget_amount"`
	BudgetCurrency string `json:"budget_currency"`
	BudgetScale   int    `json:"budget_scale"`
	State         string `json:"state"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

type Lease struct {
	LeaseID      string     `json:"lease_id"`
	OrderID      string     `json:"order_id"`
	FleetID      string     `json:"fleet_id"`
	AssignedGPUs int        `json:"assigned_gpus"`
	StartAt      time.Time  `json:"start_at"`
	EndAt        *time.Time `json:"end_at,omitempty"`
	State        string     `json:"state"`
}

// GetOffers retrieves all offers
func (r *Repository) GetOffers() ([]Offer, error) {
	rows, err := r.db.Query("SELECT offer_id, provider_id, fleet_id, unit, unit_price_amount, unit_price_currency, unit_price_scale, min_hours, max_hours, min_gpus, max_gpus, valid_from, valid_to FROM offers")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var offers []Offer
	for rows.Next() {
		var offer Offer
		err := rows.Scan(&offer.OfferID, &offer.ProviderID, &offer.FleetID, &offer.Unit, &offer.UnitPriceAmount, &offer.UnitPriceCurrency, &offer.UnitPriceScale, &offer.MinHours, &offer.MaxHours, &offer.MinGPUs, &offer.MaxGPUs, &offer.ValidFrom, &offer.ValidTo)
		if err != nil {
			return nil, err
		}
		offers = append(offers, offer)
	}

	return offers, nil
}

// GetOrders retrieves all orders
func (r *Repository) GetOrders() ([]Order, error) {
	rows, err := r.db.Query("SELECT order_id, buyer_id, offer_id, requested_gpus, hours, budget_amount, budget_currency, budget_scale, state, created_at, updated_at FROM orders")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var order Order
		err := rows.Scan(&order.OrderID, &order.BuyerID, &order.OfferID, &order.RequestedGPUs, &order.Hours, &order.BudgetAmount, &order.BudgetCurrency, &order.BudgetScale, &order.State, &order.CreatedAt, &order.UpdatedAt)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}

	return orders, nil
}

// GetLeases retrieves all leases
func (r *Repository) GetLeases() ([]Lease, error) {
	rows, err := r.db.Query("SELECT lease_id, order_id, fleet_id, assigned_gpus, start_at, end_at, state FROM leases")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var leases []Lease
	for rows.Next() {
		var lease Lease
		var endAt sql.NullString
		err := rows.Scan(&lease.LeaseID, &lease.OrderID, &lease.FleetID, &lease.AssignedGPUs, &lease.StartAt, &endAt, &lease.State)
		if err != nil {
			return nil, err
		}
		if endAt.Valid {
			if t, err := time.Parse(time.RFC3339, endAt.String); err == nil {
				lease.EndAt = &t
			}
		}
		leases = append(leases, lease)
	}

	return leases, nil
}

// GetParties retrieves all parties
func (r *Repository) GetParties() ([]Identity, error) {
	rows, err := r.db.Query("SELECT party_id, role, display_name, email, key_id, public_key, created_at FROM identities")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var parties []Identity
	for rows.Next() {
		var party Identity
		err := rows.Scan(&party.PartyID, &party.Role, &party.DisplayName, &party.Email, &party.KeyID, &party.PublicKey, &party.CreatedAt)
		if err != nil {
			return nil, err
		}
		parties = append(parties, party)
	}

	return parties, nil
}
