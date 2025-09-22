package consensus

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"time"
)

// OCXStateMachine implements the OCX protocol state machine
type OCXStateMachine struct {
	app *OCXApplication
}

// OCXApplication handles the application logic for the consensus layer
type OCXApplication struct {
	state        *OCXState
	lastHeight   int64
	validatorSet *ValidatorSet
}

// OCXState represents the current state of the OCX protocol
type OCXState struct {
	LastHeight     int64
	OrderBook      *OrderBook
	ActiveSessions map[string]*Session
	EscrowAccounts map[string]*EscrowAccount
}

// OrderBook manages buy and sell orders
type OrderBook struct {
	BuyOrders  []*Order
	SellOrders []*Order
}

// Order represents a trading order
type Order struct {
	ID        string
	TraderID  string
	Asset     string
	Quantity  int64
	Price     int64
	Side      string // "buy" or "sell"
	Status    string // "active", "filled", "cancelled"
	CreatedAt time.Time
}

// Session represents an active trading session
type Session struct {
	ID           string
	BuyerID      string
	SellerID     string
	Asset        string
	Quantity     int64
	Price        int64
	Status       string // "active", "completed", "disputed"
	StartedAt    time.Time
	CompletedAt  *time.Time
	ReceiptHash  string
}

// EscrowAccount manages escrow funds
type EscrowAccount struct {
	TraderID string
	Asset    string
	Balance  int64
	Locked   int64
}

// ValidatorSet manages the validator nodes
type ValidatorSet struct {
	Validators []*Validator
}

// Validator represents a consensus validator
type Validator struct {
	ID        string
	PublicKey ed25519.PublicKey
	Power     int64
}

// Message types for the protocol
type MsgPlaceOrder struct {
	TraderID string
	Asset    string
	Quantity int64
	Price    int64
	Side     string
}

type MsgMatchOrder struct {
	BuyOrderID  string
	SellOrderID string
	Quantity    int64
	Price       int64
}

type MsgProvisionSession struct {
	SessionID string
	BuyerID   string
	SellerID  string
	Asset     string
	Quantity  int64
	Price     int64
}

type MsgSettleSession struct {
	SessionID string
	Receipt   string
}

// NewOCXStateMachine creates a new state machine
func NewOCXStateMachine() *OCXStateMachine {
	app := &OCXApplication{
		state: &OCXState{
			LastHeight:     0,
			OrderBook:      &OrderBook{},
			ActiveSessions: make(map[string]*Session),
			EscrowAccounts: make(map[string]*EscrowAccount),
		},
		validatorSet: &ValidatorSet{},
	}

	return &OCXStateMachine{app: app}
}

// ProcessMessage processes a protocol message
func (sm *OCXStateMachine) ProcessMessage(ctx context.Context, msgType string, msgData []byte) error {
	var msg interface{}
	if err := json.Unmarshal(msgData, &msg); err != nil {
		return fmt.Errorf("invalid message format: %w", err)
	}

	switch msgType {
	case "place_order":
		var placeOrder MsgPlaceOrder
		if err := json.Unmarshal(msgData, &placeOrder); err != nil {
			return fmt.Errorf("invalid place order message: %w", err)
		}
		return sm.app.ValidateOrderPlacement(ctx, &placeOrder)
	case "match_order":
		var matchOrder MsgMatchOrder
		if err := json.Unmarshal(msgData, &matchOrder); err != nil {
			return fmt.Errorf("invalid match order message: %w", err)
		}
		return sm.app.ExecuteMatching(ctx, &matchOrder)
	case "provision_session":
		var provisionSession MsgProvisionSession
		if err := json.Unmarshal(msgData, &provisionSession); err != nil {
			return fmt.Errorf("invalid provision session message: %w", err)
		}
		return sm.app.ExecuteProvisioning(ctx, &provisionSession)
	case "settle_session":
		var settleSession MsgSettleSession
		if err := json.Unmarshal(msgData, &settleSession); err != nil {
			return fmt.Errorf("invalid settle session message: %w", err)
		}
		return sm.app.ExecuteSettlement(ctx, &settleSession)
	default:
		return fmt.Errorf("unknown message type: %s", msgType)
	}
}

// ValidateOrderPlacement validates a new order
func (app *OCXApplication) ValidateOrderPlacement(ctx context.Context, msg *MsgPlaceOrder) error {
	// Basic validation
	if msg.TraderID == "" {
		return fmt.Errorf("trader ID is required")
	}
	if msg.Asset == "" {
		return fmt.Errorf("asset is required")
	}
	if msg.Quantity <= 0 {
		return fmt.Errorf("quantity must be positive")
	}
	if msg.Price <= 0 {
		return fmt.Errorf("price must be positive")
	}
	if msg.Side != "buy" && msg.Side != "sell" {
		return fmt.Errorf("side must be 'buy' or 'sell'")
	}

	// Create the order
	order := &Order{
		ID:        fmt.Sprintf("order_%d", time.Now().UnixNano()),
		TraderID:  msg.TraderID,
		Asset:     msg.Asset,
		Quantity:  msg.Quantity,
		Price:     msg.Price,
		Side:      msg.Side,
		Status:    "active",
		CreatedAt: time.Now(),
	}

	// Add to order book
	if msg.Side == "buy" {
		app.state.OrderBook.BuyOrders = append(app.state.OrderBook.BuyOrders, order)
	} else {
		app.state.OrderBook.SellOrders = append(app.state.OrderBook.SellOrders, order)
	}

	return nil
}

// ExecuteMatching executes order matching
func (app *OCXApplication) ExecuteMatching(ctx context.Context, msg *MsgMatchOrder) error {
	// Find the orders
	var buyOrder, sellOrder *Order
	for _, order := range app.state.OrderBook.BuyOrders {
		if order.ID == msg.BuyOrderID {
			buyOrder = order
			break
		}
	}
	for _, order := range app.state.OrderBook.SellOrders {
		if order.ID == msg.SellOrderID {
			sellOrder = order
			break
		}
	}

	if buyOrder == nil || sellOrder == nil {
		return fmt.Errorf("orders not found")
	}

	// Validate match
	if buyOrder.Asset != sellOrder.Asset {
		return fmt.Errorf("asset mismatch")
	}
	if buyOrder.Price < sellOrder.Price {
		return fmt.Errorf("price mismatch")
	}

	// Create session
	session := &Session{
		ID:          fmt.Sprintf("session_%d", time.Now().UnixNano()),
		BuyerID:     buyOrder.TraderID,
		SellerID:    sellOrder.TraderID,
		Asset:       buyOrder.Asset,
		Quantity:    msg.Quantity,
		Price:       msg.Price,
		Status:      "active",
		StartedAt:   time.Now(),
		ReceiptHash: fmt.Sprintf("receipt_%d", time.Now().UnixNano()),
	}

	app.state.ActiveSessions[session.ID] = session

	// Update order status
	buyOrder.Status = "filled"
	sellOrder.Status = "filled"

	return nil
}

// ExecuteProvisioning executes session provisioning
func (app *OCXApplication) ExecuteProvisioning(ctx context.Context, msg *MsgProvisionSession) error {
	// Find the session
	session, exists := app.state.ActiveSessions[msg.SessionID]
	if !exists {
		return fmt.Errorf("session not found")
	}

	// Validate provisioning
	if session.BuyerID != msg.BuyerID || session.SellerID != msg.SellerID {
		return fmt.Errorf("session participant mismatch")
	}

	// Update session status
	session.Status = "provisioned"

	return nil
}

// ExecuteSettlement executes session settlement
func (app *OCXApplication) ExecuteSettlement(ctx context.Context, msg *MsgSettleSession) error {
	// Find the session
	session, exists := app.state.ActiveSessions[msg.SessionID]
	if !exists {
		return fmt.Errorf("session not found")
	}

	// Validate settlement
	if session.ReceiptHash != msg.Receipt {
		return fmt.Errorf("receipt mismatch")
	}

	// Complete the session
	now := time.Now()
	session.CompletedAt = &now
	session.Status = "completed"

	return nil
}

// GetState returns the current state
func (app *OCXApplication) GetState() *OCXState {
	return app.state
}

// GetLastHeight returns the last processed height
func (app *OCXApplication) GetLastHeight() int64 {
	return app.state.LastHeight
}