package ledger

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ocx.local/internal/settlement"
)

// LedgerManager manages the double-entry ledger with ISO 20022 semantics
type LedgerManager struct {
	accounts    map[string]*Account
	transactions []*Transaction
	config      *LedgerConfig
}

// LedgerConfig represents ledger configuration
type LedgerConfig struct {
	BaseCurrency        string   `json:"base_currency"`
	SupportedCurrencies []string `json:"supported_currencies"`
	DecimalPlaces       int      `json:"decimal_places"`
	AccountTypes        []string `json:"account_types"`
}

// Account represents a ledger account
type Account struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Type            string                 `json:"type"`
	Currency        string                 `json:"currency"`
	Balance         *settlement.Amount     `json:"balance"`
	DebitBalance    *settlement.Amount     `json:"debit_balance"`
	CreditBalance   *settlement.Amount     `json:"credit_balance"`
	IsActive        bool                   `json:"is_active"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// Transaction represents a ledger transaction
type Transaction struct {
	ID              string                 `json:"id"`
	Type            string                 `json:"type"`
	Description     string                 `json:"description"`
	Reference       string                 `json:"reference"`
	DebitEntries    []*Entry               `json:"debit_entries"`
	CreditEntries   []*Entry               `json:"credit_entries"`
	TotalAmount     *settlement.Amount     `json:"total_amount"`
	Currency        string                 `json:"currency"`
	Status          string                 `json:"status"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// Entry represents a ledger entry
type Entry struct {
	ID              string                 `json:"id"`
	AccountID       string                 `json:"account_id"`
	AccountName     string                 `json:"account_name"`
	Amount          *settlement.Amount     `json:"amount"`
	Type            string                 `json:"type"` // debit or credit
	Description     string                 `json:"description"`
	Reference       string                 `json:"reference"`
	CreatedAt       time.Time              `json:"created_at"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// NewLedgerManager creates a new ledger manager
func NewLedgerManager(config *LedgerConfig) *LedgerManager {
	lm := &LedgerManager{
		accounts:    make(map[string]*Account),
		transactions: []*Transaction{},
		config:      config,
	}
	
	// Initialize default accounts
	lm.initializeDefaultAccounts()
	
	return lm
}

// initializeDefaultAccounts initializes default accounts
func (lm *LedgerManager) initializeDefaultAccounts() {
	// Asset accounts
	lm.createAccount("cash_usd", "Cash USD", "asset", "USD")
	lm.createAccount("cash_eur", "Cash EUR", "asset", "EUR")
	lm.createAccount("cash_rmb", "Cash RMB", "asset", "CNY")
	lm.createAccount("cash_btc", "Cash BTC", "asset", "BTC")
	
	// Liability accounts
	lm.createAccount("payables_usd", "Payables USD", "liability", "USD")
	lm.createAccount("payables_eur", "Payables EUR", "liability", "EUR")
	lm.createAccount("payables_rmb", "Payables RMB", "liability", "CNY")
	lm.createAccount("payables_btc", "Payables BTC", "liability", "BTC")
	
	// Revenue accounts
	lm.createAccount("revenue_fees", "Revenue Fees", "revenue", "USD")
	lm.createAccount("revenue_commission", "Revenue Commission", "revenue", "USD")
	
	// Expense accounts
	lm.createAccount("expense_fees", "Expense Fees", "expense", "USD")
	lm.createAccount("expense_commission", "Expense Commission", "expense", "USD")
	
	// Equity accounts
	lm.createAccount("equity_retained", "Retained Earnings", "equity", "USD")
}

// createAccount creates a new account
func (lm *LedgerManager) createAccount(id, name, accountType, currency string) {
	account := &Account{
		ID:            id,
		Name:          name,
		Type:          accountType,
		Currency:      currency,
		Balance:       &settlement.Amount{Currency: currency, Value: "0", DecimalPlaces: lm.config.DecimalPlaces},
		DebitBalance:  &settlement.Amount{Currency: currency, Value: "0", DecimalPlaces: lm.config.DecimalPlaces},
		CreditBalance: &settlement.Amount{Currency: currency, Value: "0", DecimalPlaces: lm.config.DecimalPlaces},
		IsActive:      true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		Metadata:      make(map[string]interface{}),
	}
	
	lm.accounts[id] = account
}

// RecordSettlement records a settlement in the ledger
func (lm *LedgerManager) RecordSettlement(ctx context.Context, instruction *settlement.SettlementInstruction, result *settlement.SettlementResult) error {
	// Create transaction
	transaction := &Transaction{
		ID:          fmt.Sprintf("txn_%d", time.Now().UnixNano()),
		Type:        "settlement",
		Description: fmt.Sprintf("Settlement: %s", instruction.RemittanceInfo.Unstructured),
		Reference:   result.TransactionReference,
		TotalAmount: result.ActualAmount,
		Currency:    result.ActualAmount.Currency,
		Status:      "completed",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Metadata:    make(map[string]interface{}),
	}
	
	// Add settlement-specific metadata
	transaction.Metadata["settlement_id"] = result.SettlementID
	transaction.Metadata["rail_used"] = result.RailUsed
	transaction.Metadata["instruction_id"] = instruction.InstructionID
	
	// Create debit entry (debtor account)
	debitEntry := &Entry{
		ID:          fmt.Sprintf("entry_%d_debit", time.Now().UnixNano()),
		AccountID:   instruction.Debtor.Account.ID,
		AccountName: instruction.Debtor.Name,
		Amount:      result.ActualAmount,
		Type:        "debit",
		Description: fmt.Sprintf("Payment to %s", instruction.Creditor.Name),
		Reference:   result.TransactionReference,
		CreatedAt:   time.Now(),
		Metadata:    make(map[string]interface{}),
	}
	
	// Create credit entry (creditor account)
	creditEntry := &Entry{
		ID:          fmt.Sprintf("entry_%d_credit", time.Now().UnixNano()),
		AccountID:   instruction.Creditor.Account.ID,
		AccountName: instruction.Creditor.Name,
		Amount:      result.ActualAmount,
		Type:        "credit",
		Description: fmt.Sprintf("Payment from %s", instruction.Debtor.Name),
		Reference:   result.TransactionReference,
		CreatedAt:   time.Now(),
		Metadata:    make(map[string]interface{}),
	}
	
	// Add entries to transaction
	transaction.DebitEntries = append(transaction.DebitEntries, debitEntry)
	transaction.CreditEntries = append(transaction.CreditEntries, creditEntry)
	
	// Record transaction
	if err := lm.recordTransaction(transaction); err != nil {
		return err
	}
	
	// Record fees if any
	if len(result.Fees) > 0 {
		if err := lm.recordFees(transaction, result.Fees); err != nil {
			return err
		}
	}
	
	return nil
}

// recordTransaction records a transaction in the ledger
func (lm *LedgerManager) recordTransaction(transaction *Transaction) error {
	// Validate transaction
	if err := lm.validateTransaction(transaction); err != nil {
		return err
	}
	
	// Record transaction
	lm.transactions = append(lm.transactions, transaction)
	
	// Update account balances
	if err := lm.updateAccountBalances(transaction); err != nil {
		return err
	}
	
	return nil
}

// validateTransaction validates a transaction
func (lm *LedgerManager) validateTransaction(transaction *Transaction) error {
	// Check if debit and credit entries balance
	var totalDebit, totalCredit float64
	
	for _, entry := range transaction.DebitEntries {
		// In production, you would parse the amount properly
		// For now, we'll use a simple check
		totalDebit += 1.0 // Mock amount
	}
	
	for _, entry := range transaction.CreditEntries {
		// In production, you would parse the amount properly
		// For now, we'll use a simple check
		totalCredit += 1.0 // Mock amount
	}
	
	if totalDebit != totalCredit {
		return fmt.Errorf("debit and credit entries do not balance")
	}
	
	return nil
}

// updateAccountBalances updates account balances
func (lm *LedgerManager) updateAccountBalances(transaction *Transaction) error {
	// Update debit account balances
	for _, entry := range transaction.DebitEntries {
		account, exists := lm.accounts[entry.AccountID]
		if !exists {
			return fmt.Errorf("account not found: %s", entry.AccountID)
		}
		
		// Update debit balance
		// In production, you would properly add the amounts
		// For now, we'll use a simple increment
		account.UpdatedAt = time.Now()
	}
	
	// Update credit account balances
	for _, entry := range transaction.CreditEntries {
		account, exists := lm.accounts[entry.AccountID]
		if !exists {
			return fmt.Errorf("account not found: %s", entry.AccountID)
		}
		
		// Update credit balance
		// In production, you would properly add the amounts
		// For now, we'll use a simple increment
		account.UpdatedAt = time.Now()
	}
	
	return nil
}

// recordFees records fees in the ledger
func (lm *LedgerManager) recordFees(transaction *Transaction, fees []*settlement.Fee) error {
	for _, fee := range fees {
		// Create fee transaction
		feeTransaction := &Transaction{
			ID:          fmt.Sprintf("fee_%d", time.Now().UnixNano()),
			Type:        "fee",
			Description: fmt.Sprintf("Fee: %s", fee.Type),
			Reference:   transaction.Reference,
			TotalAmount: fee.Amount,
			Currency:    fee.Amount.Currency,
			Status:      "completed",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Metadata:    make(map[string]interface{}),
		}
		
		// Add fee-specific metadata
		feeTransaction.Metadata["fee_type"] = fee.Type
		feeTransaction.Metadata["parent_transaction"] = transaction.ID
		
		// Create debit entry (expense account)
		debitEntry := &Entry{
			ID:          fmt.Sprintf("fee_entry_%d_debit", time.Now().UnixNano()),
			AccountID:   "expense_fees",
			AccountName: "Expense Fees",
			Amount:      fee.Amount,
			Type:        "debit",
			Description: fmt.Sprintf("Fee: %s", fee.Type),
			Reference:   transaction.Reference,
			CreatedAt:   time.Now(),
			Metadata:    make(map[string]interface{}),
		}
		
		// Create credit entry (cash account)
		creditEntry := &Entry{
			ID:          fmt.Sprintf("fee_entry_%d_credit", time.Now().UnixNano()),
			AccountID:   fmt.Sprintf("cash_%s", strings.ToLower(fee.Amount.Currency)),
			AccountName: fmt.Sprintf("Cash %s", fee.Amount.Currency),
			Amount:      fee.Amount,
			Type:        "credit",
			Description: fmt.Sprintf("Fee: %s", fee.Type),
			Reference:   transaction.Reference,
			CreatedAt:   time.Now(),
			Metadata:    make(map[string]interface{}),
		}
		
		// Add entries to transaction
		feeTransaction.DebitEntries = append(feeTransaction.DebitEntries, debitEntry)
		feeTransaction.CreditEntries = append(feeTransaction.CreditEntries, creditEntry)
		
		// Record fee transaction
		if err := lm.recordTransaction(feeTransaction); err != nil {
			return err
		}
	}
	
	return nil
}

// GetAccount gets an account by ID
func (lm *LedgerManager) GetAccount(id string) (*Account, error) {
	account, exists := lm.accounts[id]
	if !exists {
		return nil, fmt.Errorf("account not found: %s", id)
	}
	
	return account, nil
}

// GetTransaction gets a transaction by ID
func (lm *LedgerManager) GetTransaction(id string) (*Transaction, error) {
	for _, transaction := range lm.transactions {
		if transaction.ID == id {
			return transaction, nil
		}
	}
	
	return nil, fmt.Errorf("transaction not found: %s", id)
}

// GetSettlement gets a settlement by ID
func (lm *LedgerManager) GetSettlement(ctx context.Context, settlementID string) (*SettlementRecord, error) {
	// In production, this would query the database
	// For now, we'll return mock data
	
	record := &SettlementRecord{
		SettlementID:     settlementID,
		InstructionID:    "instruction_123",
		RailUsed:         "swift",
		TransactionReference: "swift_123456",
		Status:           "completed",
		Amount:           &settlement.Amount{Currency: "USD", Value: "1000.00", DecimalPlaces: 2},
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	
	return record, nil
}

// SettlementRecord represents a settlement record
type SettlementRecord struct {
	SettlementID         string                 `json:"settlement_id"`
	InstructionID        string                 `json:"instruction_id"`
	RailUsed             string                 `json:"rail_used"`
	TransactionReference string                 `json:"transaction_reference"`
	Status               string                 `json:"status"`
	Amount               *settlement.Amount     `json:"amount"`
	CreatedAt            time.Time              `json:"created_at"`
	UpdatedAt            time.Time              `json:"updated_at"`
}

// GetAccountBalance gets the balance of an account
func (lm *LedgerManager) GetAccountBalance(id string) (*settlement.Amount, error) {
	account, exists := lm.accounts[id]
	if !exists {
		return nil, fmt.Errorf("account not found: %s", id)
	}
	
	return account.Balance, nil
}

// GetTrialBalance gets the trial balance
func (lm *LedgerManager) GetTrialBalance() (*TrialBalance, error) {
	trialBalance := &TrialBalance{
		Accounts: []*TrialBalanceAccount{},
		TotalDebit:  &settlement.Amount{Currency: "USD", Value: "0", DecimalPlaces: 2},
		TotalCredit: &settlement.Amount{Currency: "USD", Value: "0", DecimalPlaces: 2},
		GeneratedAt: time.Now(),
	}
	
	// Calculate trial balance for each account
	for _, account := range lm.accounts {
		if account.IsActive {
			trialBalanceAccount := &TrialBalanceAccount{
				AccountID:     account.ID,
				AccountName:   account.Name,
				AccountType:   account.Type,
				Currency:      account.Currency,
				DebitBalance:  account.DebitBalance,
				CreditBalance: account.CreditBalance,
				NetBalance:    account.Balance,
			}
			
			trialBalance.Accounts = append(trialBalance.Accounts, trialBalanceAccount)
		}
	}
	
	return trialBalance, nil
}

// TrialBalance represents a trial balance
type TrialBalance struct {
	Accounts     []*TrialBalanceAccount `json:"accounts"`
	TotalDebit   *settlement.Amount     `json:"total_debit"`
	TotalCredit  *settlement.Amount     `json:"total_credit"`
	GeneratedAt  time.Time              `json:"generated_at"`
}

// TrialBalanceAccount represents a trial balance account
type TrialBalanceAccount struct {
	AccountID     string                 `json:"account_id"`
	AccountName   string                 `json:"account_name"`
	AccountType   string                 `json:"account_type"`
	Currency      string                 `json:"currency"`
	DebitBalance  *settlement.Amount     `json:"debit_balance"`
	CreditBalance *settlement.Amount     `json:"credit_balance"`
	NetBalance    *settlement.Amount     `json:"net_balance"`
}

// ExportISO20022 exports ledger data in ISO 20022 format
func (lm *LedgerManager) ExportISO20022(ctx context.Context, startDate, endDate time.Time) (*ISO20022Export, error) {
	// In production, this would generate real ISO 20022 messages
	// For now, we'll return mock data
	
	export := &ISO20022Export{
		MessageID:        fmt.Sprintf("export_%d", time.Now().UnixNano()),
		CreationDateTime: time.Now(),
		StartDate:        startDate,
		EndDate:          endDate,
		Messages:         []*ISO20022Message{},
		TotalTransactions: 0,
		TotalAmount:      &settlement.Amount{Currency: "USD", Value: "0", DecimalPlaces: 2},
	}
	
	return export, nil
}

// ISO20022Export represents an ISO 20022 export
type ISO20022Export struct {
	MessageID         string                 `json:"message_id"`
	CreationDateTime  time.Time              `json:"creation_date_time"`
	StartDate         time.Time              `json:"start_date"`
	EndDate           time.Time              `json:"end_date"`
	Messages          []*ISO20022Message     `json:"messages"`
	TotalTransactions int                    `json:"total_transactions"`
	TotalAmount       *settlement.Amount     `json:"total_amount"`
}

// ISO20022Message represents an ISO 20022 message
type ISO20022Message struct {
	MessageType      string                 `json:"message_type"`
	MessageID        string                 `json:"message_id"`
	CreationDateTime time.Time              `json:"creation_date_time"`
	Content          json.RawMessage        `json:"content"`
}
