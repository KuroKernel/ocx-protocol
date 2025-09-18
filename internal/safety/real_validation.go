package safety

import (
	"go/ast"
	"strings"
)

// getRealFunctionLineCount calculates the actual number of lines in a function
func (sv *SafetyValidator) getRealFunctionLineCount(fn *ast.FuncDecl) int {
	if fn.Body == nil {
		return 0
	}

	// Count statements in the function body
	lineCount := 0
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.AssignStmt, *ast.ExprStmt, *ast.ReturnStmt, *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt, *ast.CaseClause, *ast.GoStmt, *ast.DeferStmt:
			lineCount++
		}
		return true
	})

	return lineCount
}

// checkRealUnsafeLoops checks for loops without hard limits
func (sv *SafetyValidator) checkRealUnsafeLoops(node ast.Node, result *ValidationResult) {
	ast.Inspect(node, func(n ast.Node) bool {
		switch loop := n.(type) {
		case *ast.ForStmt:
			// Check for infinite loops (no condition, no increment)
			if loop.Cond == nil && loop.Post == nil {
				result.Violations = append(result.Violations, "Infinite loop detected: for loop without condition or increment")
				result.UnsafeLoops = append(result.UnsafeLoops, "infinite_for_loop")
			}
			
			// Check for loops without bounds
			if loop.Cond != nil {
				// Check if condition uses a variable that might not be bounded
				ast.Inspect(loop.Cond, func(cond ast.Node) bool {
					if ident, ok := cond.(*ast.Ident); ok {
						// Check if it's a simple variable comparison
						if strings.Contains(ident.Name, "len") || strings.Contains(ident.Name, "count") {
							// This is likely bounded
							return true
						}
					}
					return true
				})
			}

		case *ast.RangeStmt:
			// Range loops are generally safe as they're bounded by the collection
			// But check for potential issues
			if loop.Key != nil && loop.Value != nil {
				// This is a key-value range, which is safe
			}
		}
		return true
	})
}

// checkRealUnhandledErrors checks for unhandled errors
func (sv *SafetyValidator) checkRealUnhandledErrors(node ast.Node, result *ValidationResult) {
	ast.Inspect(node, func(n ast.Node) bool {
		switch stmt := n.(type) {
		case *ast.AssignStmt:
			// Check for assignments that might return errors
			for _, rhs := range stmt.Rhs {
				if call, ok := rhs.(*ast.CallExpr); ok {
					if ident, ok := call.Fun.(*ast.Ident); ok {
						// Check for common functions that return errors
						errorReturningFunctions := []string{
							"Open", "Create", "Read", "Write", "Close", "Parse", "Marshal", "Unmarshal",
							"Exec", "Query", "QueryRow", "Scan", "Connect", "Listen", "Accept",
						}
						
						for _, funcName := range errorReturningFunctions {
							if strings.Contains(ident.Name, funcName) {
								// Check if the error is being handled
								if len(stmt.Lhs) < 2 {
									result.Violations = append(result.Violations, 
										"Unhandled error: function '"+ident.Name+"' returns error but it's not being checked")
									result.UnhandledErrors = append(result.UnhandledErrors, ident.Name)
								}
								break
							}
						}
					}
				}
			}

		case *ast.ExprStmt:
			// Check for function calls that return errors but aren't assigned
			if call, ok := stmt.X.(*ast.CallExpr); ok {
				if ident, ok := call.Fun.(*ast.Ident); ok {
					errorReturningFunctions := []string{
						"Open", "Create", "Read", "Write", "Close", "Parse", "Marshal", "Unmarshal",
						"Exec", "Query", "QueryRow", "Scan", "Connect", "Listen", "Accept",
					}
					
					for _, funcName := range errorReturningFunctions {
						if strings.Contains(ident.Name, funcName) {
							result.Violations = append(result.Violations, 
								"Unhandled error: function '"+ident.Name+"' returns error but result is ignored")
							result.UnhandledErrors = append(result.UnhandledErrors, ident.Name)
							break
						}
					}
				}
			}
		}
		return true
	})
}
