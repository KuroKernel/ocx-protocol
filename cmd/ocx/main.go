package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "ocx",
	Short: "OCX - Deterministic execution and cryptographic receipts",
	Long: `OCX provides deterministic execution of artifacts with cryptographic receipts.

The system ensures that execution is reproducible across different environments
and provides cryptographic proof of execution results.`,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Commands are registered in their respective files:
	// - execute.go: executeCmd
	// - verify.go: verifyCmd (if exists)
	// - benchmark.go: benchmarkCmd
	// - conformance.go: conformanceCmd
	// - gen_vectors.go: genVectorsCmd
}