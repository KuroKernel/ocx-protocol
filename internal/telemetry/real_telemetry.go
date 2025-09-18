package telemetry

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// getRealCPUUsage gets actual CPU usage from /proc/stat
func (tc *TelemetryCollector) getRealCPUUsage() (float64, error) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return 0, fmt.Errorf("failed to read /proc/stat")
	}

	line := scanner.Text()
	fields := strings.Fields(line)
	if len(fields) < 8 {
		return 0, fmt.Errorf("invalid /proc/stat format")
	}

	// Parse CPU times
	var times []uint64
	for i := 1; i < 8; i++ {
		val, err := strconv.ParseUint(fields[i], 10, 64)
		if err != nil {
			return 0, err
		}
		times = append(times, val)
	}

	// Calculate CPU usage
	idle := times[3] + times[4] // idle + iowait
	nonIdle := times[0] + times[1] + times[2] + times[5] + times[6] // user + nice + system + irq + softirq
	total := idle + nonIdle

	// Wait a bit and read again
	time.Sleep(100 * time.Millisecond)
	
	file2, err := os.Open("/proc/stat")
	if err != nil {
		return 0, err
	}
	defer file2.Close()

	scanner2 := bufio.NewScanner(file2)
	if !scanner2.Scan() {
		return 0, fmt.Errorf("failed to read /proc/stat second time")
	}

	line2 := scanner2.Text()
	fields2 := strings.Fields(line2)
	if len(fields2) < 8 {
		return 0, fmt.Errorf("invalid /proc/stat format second time")
	}

	var times2 []uint64
	for i := 1; i < 8; i++ {
		val, err := strconv.ParseUint(fields2[i], 10, 64)
		if err != nil {
			return 0, err
		}
		times2 = append(times2, val)
	}

	idle2 := times2[3] + times2[4]
	nonIdle2 := times2[0] + times2[1] + times2[2] + times2[5] + times2[6]
	total2 := idle2 + nonIdle2

	// Calculate percentage
	totalDiff := total2 - total
	idleDiff := idle2 - idle
	
	if totalDiff == 0 {
		return 0, nil
	}

	cpuUsage := float64(totalDiff-idleDiff) / float64(totalDiff) * 100.0
	return cpuUsage, nil
}

// getRealMemoryUsage gets actual memory usage from /proc/meminfo
func (tc *TelemetryCollector) getRealMemoryUsage() (float64, float64, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	var memTotal, memAvailable float64
	scanner := bufio.NewScanner(file)
	
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		switch fields[0] {
		case "MemTotal:":
			if val, err := strconv.ParseFloat(fields[1], 64); err == nil {
				memTotal = val / 1024.0 // Convert KB to MB
			}
		case "MemAvailable:":
			if val, err := strconv.ParseFloat(fields[1], 64); err == nil {
				memAvailable = val / 1024.0 // Convert KB to MB
			}
		}
	}

	if memTotal == 0 {
		return 0, 0, fmt.Errorf("failed to get memory total")
	}

	memUsed := memTotal - memAvailable
	return memUsed, memTotal, nil
}

// getRealDiskIO gets actual disk I/O from /proc/diskstats
func (tc *TelemetryCollector) getRealDiskIO() (float64, float64, error) {
	file, err := os.Open("/proc/diskstats")
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	var totalRead, totalWrite float64
	scanner := bufio.NewScanner(file)
	
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 14 {
			continue
		}

		// Skip loop devices and partitions
		if strings.Contains(fields[2], "loop") || strings.Contains(fields[2], "dm-") {
			continue
		}

		// Read sectors (field 5) and write sectors (field 9)
		if readSectors, err := strconv.ParseFloat(fields[5], 64); err == nil {
			totalRead += readSectors * 512.0 / 1024.0 / 1024.0 // Convert to MB
		}
		if writeSectors, err := strconv.ParseFloat(fields[9], 64); err == nil {
			totalWrite += writeSectors * 512.0 / 1024.0 / 1024.0 // Convert to MB
		}
	}

	return totalRead, totalWrite, nil
}

// getRealNetworkIO gets actual network I/O from /proc/net/dev
func (tc *TelemetryCollector) getRealNetworkIO() (float64, float64, error) {
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	var totalRx, totalTx float64
	scanner := bufio.NewScanner(file)
	
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, ":") {
			continue
		}

		parts := strings.Split(line, ":")
		if len(parts) < 2 {
			continue
		}

		interfaceName := strings.TrimSpace(parts[0])
		// Skip loopback interface
		if interfaceName == "lo" {
			continue
		}

		fields := strings.Fields(parts[1])
		if len(fields) < 16 {
			continue
		}

		// RX bytes (field 0) and TX bytes (field 8)
		if rxBytes, err := strconv.ParseFloat(fields[0], 64); err == nil {
			totalRx += rxBytes / 1024.0 / 1024.0 // Convert to MB
		}
		if txBytes, err := strconv.ParseFloat(fields[8], 64); err == nil {
			totalTx += txBytes / 1024.0 / 1024.0 // Convert to MB
		}
	}

	return totalRx, totalTx, nil
}
