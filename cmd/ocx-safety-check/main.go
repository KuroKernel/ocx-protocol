package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"ocx.local/internal/safety"
)

func main() {
	var (
		configFile = flag.String("config", "", "Path to safety configuration file")
		outputFile = flag.String("output", "", "Path to output report file")
		
	)
	flag.Parse()

	// Get target directory (default to current directory)
	targetDir := "."
	if len(flag.Args()) > 0 {
		targetDir = flag.Args()[0]
	}

	// Load safety configuration
	config := safety.DefaultSafetyConfig()
	if *configFile != "" {
		// TODO: Load from file
		fmt.Printf("Loading config from %s (not implemented)\n", *configFile)
	}

	// Create safety validator
	validator := safety.NewSafetyValidator(config)

	// Find all Go files
	var goFiles []string
	err := filepath.Walk(targetDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, ".go") && !strings.Contains(path, "_test.go") {
			goFiles = append(goFiles, path)
		}
		return nil
	})
	if err != nil {
		log.Fatalf("Error walking directory: %v", err)
	}

	fmt.Printf("📁 Found %d Go files to analyze\n", len(goFiles))

	// Generate safety report
	report, err := validator.GetSafetyReport(goFiles)
	if err != nil {
		log.Fatalf("Error generating safety report: %v", err)
	}

	// Print summary
	fmt.Printf("\n📊 Safety Report Summary\n")
	fmt.Printf("========================\n")
	fmt.Printf("Total Files: %d\n", report.Summary.TotalFiles)
	fmt.Printf("Valid Files: %d\n", report.Summary.ValidFiles)
	fmt.Printf("Invalid Files: %d\n", report.Summary.InvalidFiles)
	fmt.Printf("Total Violations: %d\n", report.Summary.TotalViolations)

	// Print violation summary
	fmt.Printf("\n🚨 Violation Summary\n")
	fmt.Printf("===================\n")
	fmt.Printf("Long Functions: %d\n", report.Summary.LongFunctions)
	fmt.Printf("Unsafe Loops: %d\n", report.Summary.UnsafeLoops)
	fmt.Printf("Unhandled Errors: %d\n", report.Summary.UnhandledErrors)
	fmt.Printf("Heap Violations: %d\n", report.Summary.HeapViolations)
	fmt.Printf("Scope Violations: %d\n", report.Summary.ScopeViolations)
	fmt.Printf("Other Violations: %d\n", report.Summary.Other)

	// Print detailed results for invalid files
	if report.Summary.InvalidFiles > 0 {
		fmt.Printf("\n❌ Invalid Files:\n")
		fmt.Printf("================\n")
		for filename, result := range report.FileReports {
			if !result.IsValid {
				fmt.Printf("\n📄 %s\n", filename)
				fmt.Printf("   Violations: %d\n", len(result.Violations))
				for _, violation := range result.Violations {
					fmt.Printf("   - %s\n", violation)
				}
			}
		}
	}

	// Save report to file if requested
	if *outputFile != "" {
		// TODO: Save report to file
		fmt.Printf("\n💾 Report saved to %s (not implemented)\n", *outputFile)
	}

	// Exit with error code if violations found
	if report.Summary.TotalViolations > 0 {
		fmt.Printf("\n❌ Safety check failed with %d violations\n", report.Summary.TotalViolations)
		os.Exit(1)
	} else {
		fmt.Printf("\n✅ Safety check passed - no violations found\n")
		os.Exit(0)
	}
}
