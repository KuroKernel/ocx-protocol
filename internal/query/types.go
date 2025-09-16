package query

// OCXQuery represents an OCX-QL query
type OCXQuery struct {
	Type       string            `json:"type"`
	Table      string            `json:"table"`
	Fields     []string          `json:"fields"`
	Conditions []QueryCondition  `json:"conditions"`
	OrderBy    string            `json:"order_by"`
	OrderDesc  bool              `json:"order_desc"`
	Limit      int               `json:"limit"`
}

// QueryCondition represents a query condition
type QueryCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}
