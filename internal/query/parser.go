package query

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// OCX-QL Query Parser and Optimizer
// Implements the query language for compute resource discovery

// Query types
type QueryType int

const (
	SelectQuery QueryType = iota
	ReserveQuery
	AnalyzeQuery
)

// Base query structure
type Query struct {
	Type     QueryType
	Select   *SelectClause
	Reserve  *ReserveClause
	Analyze  *AnalyzeClause
	From     string
	Where    *WhereClause
	OrderBy  *OrderByClause
	Limit    int
}

// SELECT query components
type SelectClause struct {
	Fields []string
}

// RESERVE query components
type ReserveClause struct {
	ComputeSpec    map[string]interface{}
	ReserveOptions map[string]interface{}
	EscrowAmount   float64
	EscrowCurrency string
}

// ANALYZE query components
type AnalyzeClause struct {
	Target string
	Scope  *AnalyzeScope
	GroupBy []string
}

type AnalyzeScope struct {
	ProviderID    string
	Region        string
	HardwareType  string
	TimeRange     *TimeRange
}

type TimeRange struct {
	Duration time.Duration
	From     time.Time
	To       time.Time
}

// WHERE clause components
type WhereClause struct {
	Conditions []Condition
}

type Condition struct {
	Field    string
	Operator string
	Value    interface{}
}

// ORDER BY clause
type OrderByClause struct {
	Fields []OrderField
}

type OrderField struct {
	Field string
	Desc  bool
}

// Query execution plan
type QueryPlan struct {
	Steps         []ExecutionStep
	EstimatedCost float64
	EstimatedLatency time.Duration
}

type ExecutionStep struct {
	Operation StepType
	Table     string
	IndexHint string
	Predicate *Condition
	Limit     int
}

type StepType int

const (
	IndexScan StepType = iota
	SequentialScan
	HashJoin
	NestedLoopJoin
	Sort
	LimitStep
	Aggregate
)

// Parser for OCX-QL
type Parser struct {
	query string
	pos   int
}

// NewParser creates a new query parser
func NewParser(query string) *Parser {
	return &Parser{
		query: strings.TrimSpace(query),
		pos:   0,
	}
}

// Parse parses the OCX-QL query
func (p *Parser) Parse() (*Query, error) {
	p.skipWhitespace()
	
	if p.match("SELECT") {
		return p.parseSelectQuery()
	} else if p.match("RESERVE") {
		return p.parseReserveQuery()
	} else if p.match("ANALYZE") {
		return p.parseAnalyzeQuery()
	}
	
	return nil, fmt.Errorf("unknown query type")
}

// parseSelectQuery parses SELECT queries
func (p *Parser) parseSelectQuery() (*Query, error) {
	query := &Query{Type: SelectQuery}
	
	// Parse SELECT clause
	selectClause, err := p.parseSelectClause()
	if err != nil {
		return nil, err
	}
	query.Select = selectClause
	
	// Parse FROM clause
	if !p.match("FROM") {
		return nil, fmt.Errorf("expected FROM clause")
	}
	
	query.From = p.parseIdentifier()
	if query.From == "" {
		return nil, fmt.Errorf("expected table name after FROM")
	}
	
	// Parse WHERE clause (optional)
	if p.match("WHERE") {
		whereClause, err := p.parseWhereClause()
		if err != nil {
			return nil, err
		}
		query.Where = whereClause
	}
	
	// Parse ORDER BY clause (optional)
	if p.match("ORDER") {
		if !p.match("BY") {
			return nil, fmt.Errorf("expected BY after ORDER")
		}
		
		orderByClause, err := p.parseOrderByClause()
		if err != nil {
			return nil, err
		}
		query.OrderBy = orderByClause
	}
	
	// Parse LIMIT clause (optional)
	if p.match("LIMIT") {
		limit, err := p.parseInt()
		if err != nil {
			return nil, err
		}
		query.Limit = limit
	}
	
	return query, nil
}

// parseReserveQuery parses RESERVE queries
func (p *Parser) parseReserveQuery() (*Query, error) {
	query := &Query{Type: ReserveQuery}
	
	if !p.match("COMPUTE") {
		return nil, fmt.Errorf("expected COMPUTE after RESERVE")
	}
	
	// Parse compute specification
	computeSpec, err := p.parseComputeSpec()
	if err != nil {
		return nil, err
	}
	query.Reserve = &ReserveClause{
		ComputeSpec: computeSpec,
	}
	
	// Parse WITH options (optional)
	if p.match("WITH") {
		options, err := p.parseReserveOptions()
		if err != nil {
			return nil, err
		}
		query.Reserve.ReserveOptions = options
	}
	
	// Parse ESCROW amount (optional)
	if p.match("ESCROW") {
		amount, err := p.parseFloat()
		if err != nil {
			return nil, err
		}
		query.Reserve.EscrowAmount = amount
		
		currency := p.parseIdentifier()
		if currency == "" {
			currency = "USDC" // Default currency
		}
		query.Reserve.EscrowCurrency = currency
	}
	
	return query, nil
}

// parseAnalyzeQuery parses ANALYZE queries
func (p *Parser) parseAnalyzeQuery() (*Query, error) {
	query := &Query{Type: AnalyzeQuery}
	
	target := p.parseIdentifier()
	if target == "" {
		return nil, fmt.Errorf("expected analysis target")
	}
	query.Analyze = &AnalyzeClause{Target: target}
	
	// Parse FOR scope (optional)
	if p.match("FOR") {
		scope, err := p.parseAnalyzeScope()
		if err != nil {
			return nil, err
		}
		query.Analyze.Scope = scope
	}
	
	// Parse GROUP BY clause (optional)
	if p.match("GROUP") {
		if !p.match("BY") {
			return nil, fmt.Errorf("expected BY after GROUP")
		}
		
		groupBy, err := p.parseGroupByClause()
		if err != nil {
			return nil, err
		}
		query.Analyze.GroupBy = groupBy
	}
	
	return query, nil
}

// Helper parsing methods

func (p *Parser) parseSelectClause() (*SelectClause, error) {
	fields := []string{}
	
	for {
		field := p.parseIdentifier()
		if field == "" {
			break
		}
		fields = append(fields, field)
		
		if !p.match(",") {
			break
		}
	}
	
	if len(fields) == 0 {
		return nil, fmt.Errorf("expected field list")
	}
	
	return &SelectClause{Fields: fields}, nil
}

func (p *Parser) parseComputeSpec() (map[string]interface{}, error) {
	if !p.match("{") {
		return nil, fmt.Errorf("expected { after COMPUTE")
	}
	
	spec := make(map[string]interface{})
	
	for {
		key := p.parseIdentifier()
		if key == "" {
			break
		}
		
		if !p.match(":") {
			return nil, fmt.Errorf("expected : after key")
		}
		
		value := p.parseValue()
		spec[key] = value
		
		if !p.match(",") {
			break
		}
	}
	
	if !p.match("}") {
		return nil, fmt.Errorf("expected } to close compute spec")
	}
	
	return spec, nil
}

func (p *Parser) parseReserveOptions() (map[string]interface{}, error) {
	options := make(map[string]interface{})
	
	for {
		option := p.parseIdentifier()
		if option == "" {
			break
		}
		
		if p.match("POLICY") {
			value := p.parseString()
			options[option+"_policy"] = value
		} else if p.match("TIME") {
			value, err := p.parseDuration()
			if err != nil {
				return nil, err
			}
			options[option+"_time"] = value
		} else if p.match("REGIONS") {
			regions, err := p.parseStringArray()
			if err != nil {
				return nil, err
			}
			options[option+"_regions"] = regions
		} else {
			value := p.parseValue()
			options[option] = value
		}
		
		if !p.match(",") {
			break
		}
	}
	
	return options, nil
}

func (p *Parser) parseWhereClause() (*WhereClause, error) {
	conditions := []Condition{}
	
	for {
		field := p.parseIdentifier()
		if field == "" {
			break
		}
		
		operator := p.parseOperator()
		if operator == "" {
			return nil, fmt.Errorf("expected operator")
		}
		
		value := p.parseValue()
		
		conditions = append(conditions, Condition{
			Field:    field,
			Operator: operator,
			Value:    value,
		})
		
		if p.match("AND") {
			continue
		} else if p.match("OR") {
			// Handle OR logic (simplified for this example)
			continue
		} else {
			break
		}
	}
	
	return &WhereClause{Conditions: conditions}, nil
}

func (p *Parser) parseOrderByClause() (*OrderByClause, error) {
	fields := []OrderField{}
	
	for {
		field := p.parseIdentifier()
		if field == "" {
			break
		}
		
		desc := p.match("DESC")
		if p.match("ASC") {
			desc = false
		}
		
		fields = append(fields, OrderField{
			Field: field,
			Desc:  desc,
		})
		
		if !p.match(",") {
			break
		}
	}
	
	return &OrderByClause{Fields: fields}, nil
}

func (p *Parser) parseAnalyzeScope() (*AnalyzeScope, error) {
	scope := &AnalyzeScope{}
	
	scopeType := p.parseIdentifier()
	switch scopeType {
	case "PROVIDER":
		scope.ProviderID = p.parseIdentifier()
	case "REGION":
		scope.Region = p.parseString()
	case "HARDWARE_TYPE":
		scope.HardwareType = p.parseString()
	case "TIME_RANGE":
		timeRange, err := p.parseTimeRange()
		if err != nil {
			return nil, err
		}
		scope.TimeRange = timeRange
	default:
		return nil, fmt.Errorf("unknown scope type: %s", scopeType)
	}
	
	return scope, nil
}

func (p *Parser) parseTimeRange() (*TimeRange, error) {
	if p.match("LAST") {
		duration, err := p.parseDuration()
		if err != nil {
			return nil, err
		}
		return &TimeRange{Duration: duration}, nil
	} else if p.match("FROM") {
		from, err := p.parseTime()
		if err != nil {
			return nil, err
		}
		if !p.match("TO") {
			return nil, fmt.Errorf("expected TO after FROM")
		}
		to, err := p.parseTime()
		if err != nil {
			return nil, err
		}
		return &TimeRange{From: from, To: to}, nil
	}
	
	return nil, fmt.Errorf("expected time range specification")
}

func (p *Parser) parseGroupByClause() ([]string, error) {
	fields := []string{}
	
	for {
		field := p.parseIdentifier()
		if field == "" {
			break
		}
		fields = append(fields, field)
		
		if !p.match(",") {
			break
		}
	}
	
	return fields, nil
}

// Token parsing helpers

func (p *Parser) parseIdentifier() string {
	p.skipWhitespace()
	
	start := p.pos
	for p.pos < len(p.query) && (isAlphaNumeric(p.query[p.pos]) || p.query[p.pos] == '_') {
		p.pos++
	}
	
	if p.pos == start {
		return ""
	}
	
	return p.query[start:p.pos]
}

func (p *Parser) parseString() string {
	p.skipWhitespace()
	
	if p.pos >= len(p.query) || (p.query[p.pos] != '"' && p.query[p.pos] != '\'') {
		return ""
	}
	
	quote := p.query[p.pos]
	p.pos++ // Skip opening quote
	
	start := p.pos
	for p.pos < len(p.query) && p.query[p.pos] != quote {
		p.pos++
	}
	
	if p.pos >= len(p.query) {
		return ""
	}
	
	result := p.query[start:p.pos]
	p.pos++ // Skip closing quote
	
	return result
}

func (p *Parser) parseInt() (int, error) {
	p.skipWhitespace()
	
	start := p.pos
	for p.pos < len(p.query) && isDigit(p.query[p.pos]) {
		p.pos++
	}
	
	if p.pos == start {
		return 0, fmt.Errorf("expected integer")
	}
	
	return strconv.Atoi(p.query[start:p.pos])
}

func (p *Parser) parseFloat() (float64, error) {
	p.skipWhitespace()
	
	start := p.pos
	for p.pos < len(p.query) && (isDigit(p.query[p.pos]) || p.query[p.pos] == '.') {
		p.pos++
	}
	
	if p.pos == start {
		return 0, fmt.Errorf("expected float")
	}
	
	return strconv.ParseFloat(p.query[start:p.pos], 64)
}

func (p *Parser) parseDuration() (time.Duration, error) {
	value, err := p.parseInt()
	if err != nil {
		return 0, err
	}
	
	unit := p.parseIdentifier()
	switch unit {
	case "ms":
		return time.Duration(value) * time.Millisecond, nil
	case "s":
		return time.Duration(value) * time.Second, nil
	case "m":
		return time.Duration(value) * time.Minute, nil
	case "h":
		return time.Duration(value) * time.Hour, nil
	default:
		return 0, fmt.Errorf("unknown duration unit: %s", unit)
	}
}

func (p *Parser) parseTime() (time.Time, error) {
	timeStr := p.parseString()
	if timeStr == "" {
		return time.Time{}, fmt.Errorf("expected time string")
	}
	
	// Parse ISO 8601 format
	return time.Parse(time.RFC3339, timeStr)
}

func (p *Parser) parseValue() interface{} {
	p.skipWhitespace()
	
	if p.pos >= len(p.query) {
		return nil
	}
	
	// Try string first
	if p.query[p.pos] == '"' || p.query[p.pos] == '\'' {
		return p.parseString()
	}
	
	// Try number
	if isDigit(p.query[p.pos]) {
		start := p.pos
		for p.pos < len(p.query) && (isDigit(p.query[p.pos]) || p.query[p.pos] == '.') {
			p.pos++
		}
		if p.pos > start {
			if strings.Contains(p.query[start:p.pos], ".") {
				val, _ := strconv.ParseFloat(p.query[start:p.pos], 64)
				return val
			} else {
				val, _ := strconv.Atoi(p.query[start:p.pos])
				return val
			}
		}
	}
	
	// Try boolean
	if p.match("true") {
		return true
	}
	if p.match("false") {
		return false
	}
	
	// Try array
	if p.query[p.pos] == '[' {
		return p.parseArray()
	}
	
	// Default to identifier
	return p.parseIdentifier()
}

func (p *Parser) parseArray() []interface{} {
	if !p.match("[") {
		return nil
	}
	
	array := []interface{}{}
	
	for {
		value := p.parseValue()
		if value == nil {
			break
		}
		array = append(array, value)
		
		if !p.match(",") {
			break
		}
	}
	
	if !p.match("]") {
		return nil
	}
	
	return array
}

func (p *Parser) parseStringArray() ([]string, error) {
	if !p.match("[") {
		return nil, fmt.Errorf("expected [")
	}
	
	array := []string{}
	
	for {
		str := p.parseString()
		if str == "" {
			break
		}
		array = append(array, str)
		
		if !p.match(",") {
			break
		}
	}
	
	if !p.match("]") {
		return nil, fmt.Errorf("expected ]")
	}
	
	return array, nil
}

func (p *Parser) parseOperator() string {
	operators := []string{"!=", ">=", "<=", "=", ">", "<", "IN", "NOT IN", "LIKE"}
	
	for _, op := range operators {
		if p.match(op) {
			return op
		}
	}
	
	return ""
}

// Utility methods

func (p *Parser) match(token string) bool {
	p.skipWhitespace()
	
	if p.pos+len(token) > len(p.query) {
		return false
	}
	
	if strings.ToUpper(p.query[p.pos:p.pos+len(token)]) == strings.ToUpper(token) {
		p.pos += len(token)
		return true
	}
	
	return false
}

func (p *Parser) skipWhitespace() {
	for p.pos < len(p.query) && isWhitespace(p.query[p.pos]) {
		p.pos++
	}
}

func isAlphaNumeric(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func isWhitespace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}
