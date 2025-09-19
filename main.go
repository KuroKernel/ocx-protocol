// main.go — OCX Protocol Server
// go 1.22+

package main

import (
	"flag"
	"fmt"
)

func main() {
	var port = flag.String("port", "8080", "Port to listen on")
	var demo = flag.Bool("demo", false, "Run killer applications demo")
	flag.Parse()

	if *demo {
		fmt.Println("🚀 Running OCX Killer Applications Demo...")
		fmt.Println("==========================================")
		fmt.Println()
		
		// Run the killer demo
		runKillerDemo()
		return
	}

	gateway := NewGateway()
	gateway.StartServer(*port)
}

func runKillerDemo() {
	// This would normally call the killer demo program
	// For now, we'll just show the available programs
	fmt.Println("Available Killer Applications:")
	fmt.Println("1. AlphaFold Protein Folding Simulator")
	fmt.Println("2. LLVM Compiler Optimization Testing")
	fmt.Println("3. Bitcoin Mining Difficulty Adjustment")
	fmt.Println("4. Doom Engine Physics Simulation")
	fmt.Println("5. Chromium WebGL Benchmark")
	fmt.Println()
	fmt.Println("Run './ocx-killer-demo' to execute all programs!")
}
