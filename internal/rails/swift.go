package rails

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ocx.local/internal/settlement"
)

// SWIFTRail implements SWIFT/ISO 20022 settlement rail
type SWIFTRail struct {
	railID              string
	supportedCurrencies []string
	jurisdictions       []string
	config              *SWIFTConfig
}

// SWIFTConfig represents SWIFT configuration
type SWIFTConfig struct {
	BIC                  string   `json:"bic"`
	LEI                  string   `json:"lei"`
	InstitutionName      string   `json:"institution_name"`
	SupportedCurrencies  []string `json:"supported_currencies"`
	SupportedJurisdictions []string `json:"supported_jurisdictions"`
	APIEndpoint          string   `json:"api_endpoint"`
	APIKey               string   `json:"api_key"`
	CertPath             string   `json:"cert_path"`
	KeyPath              string   `json:"key_path"`
}

// NewSWIFTRail creates a new SWIFT rail
func NewSWIFTRail(config *SWIFTConfig) *SWIFTRail {
	return &SWIFTRail{
		railID:              "swift",
		supportedCurrencies: config.SupportedCurrencies,
		jurisdictions:       config.SupportedJurisdictions,
		config:              config,
	}
}

// GetRailID returns the rail ID
func (s *SWIFTRail) GetRailID() string {
	return s.railID
}

// GetSupportedCurrencies returns supported currencies
func (s *SWIFTRail) GetSupportedCurrencies() []string {
	return s.supportedCurrencies
}

// GetJurisdictions returns supported jurisdictions
func (s *SWIFTRail) GetJurisdictions() []string {
	return s.jurisdictions
}

// SupportsCurrency checks if currency is supported
func (s *SWIFTRail) SupportsCurrency(currency string) bool {
	for _, c := range s.supportedCurrencies {
		if c == currency {
			return true
		}
	}
	return false
}

// SupportsJurisdiction checks if jurisdiction is supported
func (s *SWIFTRail) SupportsJurisdiction(jurisdiction string) bool {
	for _, j := range s.jurisdictions {
		if j == jurisdiction {
			return true
		}
	}
	return false
}

// ProcessSettlement processes a SWIFT settlement
func (s *SWIFTRail) ProcessSettlement(ctx context.Context, instruction *settlement.SettlementInstruction) (*settlement.SettlementResult, error) {
	// 1. Create SWIFT message
	swiftMessage, err := s.createSWIFTMessage(instruction)
	if err != nil {
		return nil, err
	}
	
	// 2. Send to SWIFT network
	response, err := s.sendToSWIFT(ctx, swiftMessage)
	if err != nil {
		return nil, err
	}
	
	// 3. Parse response
	result, err := s.parseSWIFTResponse(response, instruction)
	if err != nil {
		return nil, err
	}
	
	return result, nil
}

// GetStatus gets the status of a SWIFT settlement
func (s *SWIFTRail) GetStatus(ctx context.Context, settlementID string) (*settlement.SettlementStatus, error) {
	// In production, this would query the SWIFT network
	// For now, we'll return a mock status
	
	status := &settlement.SettlementStatus{
		SettlementID:     settlementID,
		Status:           "completed",
		StatusReason:     "SWIFT settlement completed successfully",
		LastUpdated:      time.Now(),
		EstimatedCompletion: time.Now(),
	}
	
	return status, nil
}

// createSWIFTMessage creates a SWIFT/ISO 20022 message
func (s *SWIFTRail) createSWIFTMessage(instruction *settlement.SettlementInstruction) (*SWIFTMessage, error) {
	// Create FIToFICstmrCdtTrf (pacs.008.001.08) message
	message := &SWIFTMessage{
		MessageType:      "pacs.008.001.08",
		MessageID:        instruction.MessageID,
		CreationDateTime: time.Now(),
		GroupHeader: &GroupHeader{
			MessageID:        instruction.MessageID,
			CreationDateTime: time.Now(),
			NumberOfTransactions: 1,
			ControlSum:       instruction.InstructedAmount.Value,
			InitiatingParty:  s.createPartyIdentification(instruction.InitiatingParty),
		},
		CreditTransferTransactionInformation: []*CreditTransferTransactionInformation{
			{
				PaymentIdentification: &PaymentIdentification{
					InstructionID:  instruction.InstructionID,
					EndToEndID:     instruction.EndToEndID,
					TransactionID:  instruction.TransactionID,
				},
				InterbankSettlementAmount: &settlement.Amount{
					Currency:      instruction.InstructedAmount.Currency,
					Value:         instruction.InstructedAmount.Value,
					DecimalPlaces: instruction.InstructedAmount.DecimalPlaces,
				},
				ChargeBearer: "SLEV", // Service Level
				Debtor:       s.createParty(instruction.Debtor),
				DebtorAgent:  s.createFinancialInstitution(instruction.DebtorAgent),
				Creditor:     s.createParty(instruction.Creditor),
				CreditorAgent: s.createFinancialInstitution(instruction.CreditorAgent),
				RemittanceInformation: s.createRemittanceInformation(instruction.RemittanceInfo),
			},
		},
	}
	
	return message, nil
}

// SWIFTMessage represents a SWIFT/ISO 20022 message
type SWIFTMessage struct {
	MessageType                           string                                 `json:"message_type"`
	MessageID                             string                                 `json:"message_id"`
	CreationDateTime                      time.Time                              `json:"creation_date_time"`
	GroupHeader                           *GroupHeader                           `json:"group_header"`
	CreditTransferTransactionInformation  []*CreditTransferTransactionInformation `json:"credit_transfer_transaction_information"`
}

// GroupHeader represents the group header (ISO 20022 GroupHeader)
type GroupHeader struct {
	MessageID             string                 `json:"message_id"`
	CreationDateTime      time.Time              `json:"creation_date_time"`
	NumberOfTransactions  int                    `json:"number_of_transactions"`
	ControlSum            string                 `json:"control_sum"`
	InitiatingParty       *PartyIdentification   `json:"initiating_party"`
}

// CreditTransferTransactionInformation represents credit transfer transaction information
type CreditTransferTransactionInformation struct {
	PaymentIdentification      *PaymentIdentification      `json:"payment_identification"`
	InterbankSettlementAmount  *settlement.Amount          `json:"interbank_settlement_amount"`
	ChargeBearer               string                      `json:"charge_bearer"`
	Debtor                     *Party                      `json:"debtor"`
	DebtorAgent                *FinancialInstitution       `json:"debtor_agent"`
	Creditor                   *Party                      `json:"creditor"`
	CreditorAgent              *FinancialInstitution       `json:"creditor_agent"`
	RemittanceInformation      *RemittanceInformation      `json:"remittance_information"`
}

// PaymentIdentification represents payment identification
type PaymentIdentification struct {
	InstructionID  string `json:"instruction_id"`
	EndToEndID     string `json:"end_to_end_id"`
	TransactionID  string `json:"transaction_id"`
}

// PartyIdentification represents party identification
type PartyIdentification struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// Party represents a party
type Party struct {
	Name           string                 `json:"name"`
	PostalAddress  *settlement.Address    `json:"postal_address"`
	Identification *PartyIdentification   `json:"identification"`
	Account        *settlement.Account    `json:"account"`
}

// FinancialInstitution represents a financial institution
type FinancialInstitution struct {
	BIC            string                 `json:"bic"`
	LEI            string                 `json:"lei"`
	Name           string                 `json:"name"`
	PostalAddress  *settlement.Address    `json:"postal_address"`
}

// RemittanceInformation represents remittance information
type RemittanceInformation struct {
	Unstructured   string                 `json:"unstructured,omitempty"`
	Structured     *StructuredRemittance  `json:"structured,omitempty"`
}

// StructuredRemittance represents structured remittance information
type StructuredRemittance struct {
	ReferredDocumentInformation *ReferredDocumentInformation `json:"referred_document_information,omitempty"`
	ReferredDocumentAmount      *settlement.Amount           `json:"referred_document_amount,omitempty"`
	CreditorReferenceInformation *CreditorReferenceInformation `json:"creditor_reference_information,omitempty"`
}

// ReferredDocumentInformation represents referred document information
type ReferredDocumentInformation struct {
	Type         *DocumentType `json:"type,omitempty"`
	Number       string        `json:"number,omitempty"`
	RelatedDate  time.Time     `json:"related_date,omitempty"`
}

// DocumentType represents a document type
type DocumentType struct {
	CodeOrProprietary string `json:"code_or_proprietary"`
	Issuer            string `json:"issuer,omitempty"`
}

// CreditorReferenceInformation represents creditor reference information
type CreditorReferenceInformation struct {
	Type      *CreditorReferenceType `json:"type,omitempty"`
	Reference string                 `json:"reference,omitempty"`
}

// CreditorReferenceType represents creditor reference type
type CreditorReferenceType struct {
	CodeOrProprietary string `json:"code_or_proprietary"`
	Issuer            string `json:"issuer,omitempty"`
}

// createPartyIdentification creates a party identification
func (s *SWIFTRail) createPartyIdentification(party *settlement.Party) *PartyIdentification {
	if party == nil {
		return nil
	}
	
	return &PartyIdentification{
		Type:  "BIC",
		Value: party.Identification.Value,
	}
}

// createParty creates a party
func (s *SWIFTRail) createParty(party *settlement.Party) *Party {
	if party == nil {
		return nil
	}
	
	return &Party{
		Name:           party.Name,
		PostalAddress:  party.Address,
		Identification: s.createPartyIdentification(party),
		Account:        party.Account,
	}
}

// createFinancialInstitution creates a financial institution
func (s *SWIFTRail) createFinancialInstitution(fi *settlement.FinancialInstitution) *FinancialInstitution {
	if fi == nil {
		return nil
	}
	
	return &FinancialInstitution{
		BIC:           fi.BIC,
		LEI:           fi.LEI,
		Name:          fi.Name,
		PostalAddress: fi.Address,
	}
}

// createRemittanceInformation creates remittance information
func (s *SWIFTRail) createRemittanceInformation(remInfo *settlement.RemittanceInformation) *RemittanceInformation {
	if remInfo == nil {
		return nil
	}
	
	return &RemittanceInformation{
		Unstructured: remInfo.Unstructured,
		Structured:   s.createStructuredRemittance(remInfo.Structured),
	}
}

// createStructuredRemittance creates structured remittance information
func (s *SWIFTRail) createStructuredRemittance(structured *settlement.StructuredRemittance) *StructuredRemittance {
	if structured == nil {
		return nil
	}
	
	return &StructuredRemittance{
		ReferredDocumentInformation: s.createReferredDocumentInformation(structured.ReferredDocumentInformation),
		ReferredDocumentAmount:      structured.ReferredDocumentAmount,
		CreditorReferenceInformation: s.createCreditorReferenceInformation(structured.CreditorReferenceInformation),
	}
}

// createReferredDocumentInformation creates referred document information
func (s *SWIFTRail) createReferredDocumentInformation(refDoc *settlement.ReferredDocumentInformation) *ReferredDocumentInformation {
	if refDoc == nil {
		return nil
	}
	
	return &ReferredDocumentInformation{
		Type:        s.createDocumentType(refDoc.Type),
		Number:      refDoc.Number,
		RelatedDate: refDoc.RelatedDate,
	}
}

// createDocumentType creates a document type
func (s *SWIFTRail) createDocumentType(docType *settlement.DocumentType) *DocumentType {
	if docType == nil {
		return nil
	}
	
	return &DocumentType{
		CodeOrProprietary: docType.CodeOrProprietary,
		Issuer:            docType.Issuer,
	}
}

// createCreditorReferenceInformation creates creditor reference information
func (s *SWIFTRail) createCreditorReferenceInformation(credRef *settlement.CreditorReferenceInformation) *CreditorReferenceInformation {
	if credRef == nil {
		return nil
	}
	
	return &CreditorReferenceInformation{
		Type:      s.createCreditorReferenceType(credRef.Type),
		Reference: credRef.Reference,
	}
}

// createCreditorReferenceType creates creditor reference type
func (s *SWIFTRail) createCreditorReferenceType(refType *settlement.CreditorReferenceType) *CreditorReferenceType {
	if refType == nil {
		return nil
	}
	
	return &CreditorReferenceType{
		CodeOrProprietary: refType.CodeOrProprietary,
		Issuer:            refType.Issuer,
	}
}

// sendToSWIFT sends a message to the SWIFT network
func (s *SWIFTRail) sendToSWIFT(ctx context.Context, message *SWIFTMessage) (*SWIFTResponse, error) {
	// In production, this would send to the actual SWIFT network
	// For now, we'll create a mock response
	
	response := &SWIFTResponse{
		MessageID:        message.MessageID,
		Status:           "accepted",
		TransactionID:    fmt.Sprintf("swift_%d", time.Now().UnixNano()),
		ProcessingDate:   time.Now(),
		ValueDate:        time.Now().Add(24 * time.Hour),
		Fees: []*settlement.Fee{
			{
				Type:   "SWIFT_FEE",
				Amount: &settlement.Amount{Currency: "USD", Value: "25.00", DecimalPlaces: 2},
			},
		},
	}
	
	return response, nil
}

// SWIFTResponse represents a SWIFT response
type SWIFTResponse struct {
	MessageID      string                 `json:"message_id"`
	Status         string                 `json:"status"`
	TransactionID  string                 `json:"transaction_id"`
	ProcessingDate time.Time              `json:"processing_date"`
	ValueDate      time.Time              `json:"value_date"`
	Fees           []*settlement.Fee      `json:"fees"`
}

// parseSWIFTResponse parses a SWIFT response
func (s *SWIFTRail) parseSWIFTResponse(response *SWIFTResponse, instruction *settlement.SettlementInstruction) (*settlement.SettlementResult, error) {
	result := &settlement.SettlementResult{
		SettlementID:         fmt.Sprintf("swift_%d", time.Now().UnixNano()),
		InstructionID:        instruction.InstructionID,
		Status:               response.Status,
		RailUsed:             s.railID,
		TransactionReference: response.TransactionID,
		SettlementDate:       response.ProcessingDate,
		ValueDate:            response.ValueDate,
		ActualAmount:         instruction.InstructedAmount,
		Fees:                 response.Fees,
		CreatedAt:            time.Now(),
	}
	
	return result, nil
}
