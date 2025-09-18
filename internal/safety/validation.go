package safety

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// SafetyValidator validates code against safety standards
type SafetyValidator struct {
	config *SafetyConfig
}

// NewSafetyValidator creates a new safety validator
func NewSafetyValidator(config *SafetyConfig) *SafetyValidator {
	return &SafetyValidator{
		config: config,
	}
}

// ValidateFile validates a single Go file
func (sv *SafetyValidator) ValidateFile(filename string) (*ValidationResult, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file %s: %w", filename, err)
	}

	result := &ValidationResult{
		IsValid: true,
	}

	// Check function lengths
	sv.checkFunctionLengths(node, result)

	// Check for unsafe loops
	sv.checkUnsafeLoops(node, result)

	// Check for unhandled errors
	sv.checkUnhandledErrors(node, result)

	if len(result.Violations) > 0 {
		result.IsValid = false
	}

	return result, nil
}

// checkFunctionLengths checks if functions exceed the maximum line limit
func (sv *SafetyValidator) checkFunctionLengths(node ast.Node, result *ValidationResult) {
	ast.Inspect(node, func(n ast.Node) bool {
		if fn, ok := n.(*ast.FuncDecl); ok {
			result.FunctionCount++
			if fn.Body != nil {
				lineCount := sv.getFunctionLineCount(fn)
				if lineCount > sv.config.MaxFunctionLines {
					result.Violations = append(result.Violations,
						fmt.Sprintf("Function '%s' exceeds %d lines (%d lines)",
							fn.Name.Name, sv.config.MaxFunctionLines, lineCount))
					result.LongFunctions = append(result.LongFunctions, fn.Name.Name)
				}
			}
		}
		return true
	})
}

// getFunctionLineCount calculates the number of lines in a function
func (sv *SafetyValidator) getFunctionLineCount(fn *ast.FuncDecl) int {
	// Placeholder implementation - would need to count actual lines
	return 0
}

// checkUnsafeLoops checks for loops without hard limits (placeholder)
func (sv *SafetyValidator) checkUnsafeLoops(node ast.Node, result *ValidationResult) {
	// Placeholder implementation
}

// checkUnhandledErrors checks for unhandled errors (placeholder)
func (sv *SafetyValidator) checkUnhandledErrors(node ast.Node, result *ValidationResult) {
	// Placeholder implementation
}

// NewSafetyReport creates a new safety report
func NewSafetyReport() *SafetyReport {
	report := &SafetyReport{}
	report.FileReports = make(map[string]*ValidationResult)
	return report
}

// AddFileResult adds a validation result for a file
func (sr *SafetyReport) AddFileResult(filename string, result *ValidationResult) {
	sr.FileReports[filename] = result
	sr.Summary.TotalFiles++
	if result.IsValid {
		sr.Summary.ValidFiles++
	} else {
		sr.Summary.InvalidFiles++
		sr.Summary.TotalViolations += len(result.Violations)
		sr.Summary.UnhandledErrors += len(result.UnhandledErrors)
		sr.Summary.LongFunctions += len(result.LongFunctions)
		sr.Summary.UnsafeLoops += len(result.UnsafeLoops)
		// Categorize other violations
		for _, violation := range result.Violations {
			if strings.Contains(violation, "heap") {
				sr.Summary.HeapViolations++
			} else if strings.Contains(violation, "scope") {
				sr.Summary.ScopeViolations++
			} else {
				sr.Summary.Other++
			}
		}
	}
}

// GetSafetyReport generates a comprehensive safety report for multiple files
func (sv *SafetyValidator) GetSafetyReport(files []string) (*SafetyReport, error) {
	report := NewSafetyReport()
	
	for _, file := range files {
		result, err := sv.ValidateFile(file)
		if err != nil {
			// Log error but continue with other files
			fmt.Printf("Warning: Failed to validate %s: %v\n", file, err)
			continue
		}
		
		report.AddFileResult(file, result)
	}
	
	return report, nil
}
