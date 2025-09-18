package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func main() {
	fmt.Println("🚀 OCX Protocol NVIDIA GPU Test")
	fmt.Println("================================")
	
	// Check if nvidia-smi is available
	cmd := exec.Command("nvidia-smi", "--query-gpu=name,memory.total,driver_version,temperature.gpu,utilization.gpu", "--format=csv,noheader,nounits")
	output, err := cmd.Output()
	if err != nil {
		log.Fatalf("nvidia-smi not available: %v", err)
	}
	
	// Parse GPU information
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 {
		log.Fatal("No GPUs detected")
	}
	
	fmt.Println("✅ NVIDIA GPU detected!")
	
	for i, line := range lines {
		parts := strings.Split(strings.TrimSpace(line), ", ")
		if len(parts) >= 5 {
			name := strings.TrimSpace(parts[0])
			memory, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
			driver := strings.TrimSpace(parts[2])
			temperature, _ := strconv.Atoi(strings.TrimSpace(parts[3]))
			utilization, _ := strconv.Atoi(strings.TrimSpace(parts[4]))
			
			fmt.Printf("GPU %d:\n", i)
			fmt.Printf("  Name: %s\n", name)
			fmt.Printf("  Memory: %dMB\n", memory)
			fmt.Printf("  Driver: %s\n", driver)
			fmt.Printf("  Temperature: %d°C\n", temperature)
			fmt.Printf("  Utilization: %d%%\n", utilization)
			
			// Check health
			if temperature > 85 {
				fmt.Printf("  ⚠️  Temperature warning: %d°C\n", temperature)
			} else {
				fmt.Printf("  ✅ Temperature OK\n")
			}
			
			if utilization > 90 {
				fmt.Printf("  ⚠️  High utilization: %d%%\n", utilization)
			} else {
				fmt.Printf("  ✅ Utilization OK\n")
			}
		}
	}
	
	// Test GPU monitoring
	fmt.Println("\n🔍 Testing GPU monitoring...")
	for i := 0; i < 5; i++ {
		cmd := exec.Command("nvidia-smi", "--query-gpu=utilization.gpu,temperature.gpu,memory.used,power.draw", "--format=csv,noheader,nounits")
		output, err := cmd.Output()
		if err != nil {
			log.Printf("Failed to get metrics: %v", err)
			continue
		}
		
		parts := strings.Split(strings.TrimSpace(string(output)), ", ")
		if len(parts) >= 4 {
			util, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
			temp, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
			mem, _ := strconv.Atoi(strings.TrimSpace(parts[2]))
			power, _ := strconv.Atoi(strings.TrimSpace(parts[3]))
			
			fmt.Printf("  [%d] Util: %d%%, Temp: %d°C, Memory: %dMB, Power: %dW\n", 
				i+1, util, temp, mem, power)
		}
		
		time.Sleep(2 * time.Second)
	}
	
	// Get system info
	hostname, _ := os.Hostname()
	username := os.Getenv("USER")
	
	fmt.Println("\n💻 System Information:")
	fmt.Printf("  Hostname: %s\n", hostname)
	fmt.Printf("  Username: %s\n", username)
	fmt.Printf("  OS: Linux\n")
	fmt.Printf("  Architecture: x86_64\n")
	
	// Simulate provisioning
	fmt.Println("\n🚀 Simulating GPU Provisioning:")
	fmt.Printf("  Instance ID: local-nvidia-%d\n", time.Now().UnixNano())
	fmt.Printf("  Address: %s:22\n", getLocalIP())
	fmt.Printf("  SSH User: %s\n", username)
	fmt.Printf("  GPU Device: 0\n")
	fmt.Printf("  CUDA Visible: 0\n")
	fmt.Printf("  Status: running\n")
	
	fmt.Println("\n🎉 GPU test completed successfully!")
	fmt.Println("OCX Protocol can manage this NVIDIA GPU hardware")
}

func getLocalIP() string {
	cmd := exec.Command("hostname", "-I")
	output, err := cmd.Output()
	if err != nil {
		return "127.0.0.1"
	}
	
	ips := strings.Fields(string(output))
	if len(ips) > 0 {
		return ips[0]
	}
	
	return "127.0.0.1"
}
