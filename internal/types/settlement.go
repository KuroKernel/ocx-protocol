package types

import "time"

// Amount represents a monetary amount (ISO 20022 Amount)
type Amount struct {
	Currency      string `json:"currency"`
	Value         string `json:"value"`
	DecimalPlaces int    `json:"decimal_places"`
}

// ExchangeRate represents an exchange rate (ISO 20022 ExchangeRate)
type ExchangeRate struct {
	BaseCurrency          string `json:"base_currency"`
	TargetCurrency        string `json:"target_currency"`
	Rate                  string `json:"rate"`
	RateType              string `json:"rate_type"`
	ContractIdentification string `json:"contract_identification,omitempty"`
}

// Party represents a party in the settlement (ISO 20022 Party)
type Party struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Type            string                 `json:"type"` // Individual, Organization
	Address         *Address               `json:"address"`
	Identification  *PartyIdentification   `json:"identification"`
	Account         *Account               `json:"account"`
	Jurisdiction    string                 `json:"jurisdiction"`
	SanctionsStatus string                 `json:"sanctions_status"`
	KYCStatus       string                 `json:"kyc_status"`
	RiskRating      string                 `json:"risk_rating"`
}

// FinancialInstitution represents a financial institution (ISO 20022 FinancialInstitution)
type FinancialInstitution struct {
	ID                  string   `json:"id"`
	Name                string   `json:"name"`
	BIC                 string   `json:"bic"`
	LEI                 string   `json:"lei"`
	Address             *Address `json:"address"`
	Jurisdiction        string   `json:"jurisdiction"`
	SupportedCurrencies []string `json:"supported_currencies"`
	SupportedRails      []string `json:"supported_rails"`
}

// Address represents an address (ISO 20022 PostalAddress)
type Address struct {
	StreetName         string `json:"street_name,omitempty"`
	BuildingNumber     string `json:"building_number,omitempty"`
	PostalCode         string `json:"postal_code,omitempty"`
	TownName           string `json:"town_name,omitempty"`
	Country            string `json:"country"`
	CountrySubDivision string `json:"country_sub_division,omitempty"`
}

// PartyIdentification represents party identification (ISO 20022 PartyIdentification)
type PartyIdentification struct {
	Type      string    `json:"type"`
	Value     string    `json:"value"`
	Issuer    string    `json:"issuer,omitempty"`
	IssueDate time.Time `json:"issue_date,omitempty"`
	ExpiryDate time.Time `json:"expiry_date,omitempty"`
}

// Account represents an account (ISO 20022 Account)
type Account struct {
	ID            string                `json:"id"`
	Type          string                `json:"type"`
	Currency      string                `json:"currency"`
	Institution   *FinancialInstitution `json:"institution,omitempty"`
	IBAN          string                `json:"iban,omitempty"`
	AccountNumber string                `json:"account_number,omitempty"`
	RoutingNumber string                `json:"routing_number,omitempty"`
	SWIFT         string                `json:"swift,omitempty"`
}

// RemittanceInformation represents remittance information (ISO 20022 RemittanceInformation)
type RemittanceInformation struct {
	Unstructured string                 `json:"unstructured,omitempty"`
	Structured   *StructuredRemittance  `json:"structured,omitempty"`
}

// StructuredRemittance represents structured remittance information
type StructuredRemittance struct {
	ReferredDocumentInformation  *ReferredDocumentInformation  `json:"referred_document_information,omitempty"`
	ReferredDocumentAmount       *Amount                       `json:"referred_document_amount,omitempty"`
	CreditorReferenceInformation *CreditorReferenceInformation `json:"creditor_reference_information,omitempty"`
}

// ReferredDocumentInformation represents referred document information
type ReferredDocumentInformation struct {
	Type        *DocumentType `json:"type,omitempty"`
	Number      string        `json:"number,omitempty"`
	RelatedDate time.Time     `json:"related_date,omitempty"`
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

// Fee represents a fee (ISO 20022 Fee)
type Fee struct {
	Type     string  `json:"type"`
	Amount   *Amount `json:"amount"`
	Currency string  `json:"currency"`
	Rate     string  `json:"rate,omitempty"`
	Basis    string  `json:"basis,omitempty"`
}
