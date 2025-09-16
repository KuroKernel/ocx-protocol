package query

// Common types for query package
type Condition struct {
	Field    string
	Operator string
	Value    interface{}
}

type Predicate struct {
	Column   string
	Operator string
	Value    interface{}
}

// Convert Predicate to Condition
func (p Predicate) ToCondition() Condition {
	return Condition{
		Field:    p.Column,
		Operator: p.Operator,
		Value:    p.Value,
	}
}
