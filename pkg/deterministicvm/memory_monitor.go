//go:build linux
// +build linux

package deterministicvm

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// MemoryUsage reports PSS if available, else RSS in bytes.
type MemoryUsage struct {
	RSSBytes uint64
	PSSBytes uint64 // 0 if not available
}

func getMemoryUsage(pid int) (MemoryUsage, error) {
	// Prefer smaps_rollup (kernel >= 4.14 typically)
	if mu, err := readSmapsRollup(pid); err == nil {
		return mu, nil
	}
	// Fallback: /proc/<pid>/status VmRSS
	if mu, err := readStatusRSS(pid); err == nil {
		return mu, nil
	}
	// Final fallback: /proc/<pid>/statm (pages)
	if mu, err := readStatmRSS(pid); err == nil {
		return mu, nil
	}
	return MemoryUsage{}, errors.New("unable to read memory usage")
}

func readSmapsRollup(pid int) (MemoryUsage, error) {
	f, err := os.Open(fmt.Sprintf("/proc/%d/smaps_rollup", pid))
	if err != nil { 
		return MemoryUsage{}, err 
	}
	defer f.Close()
	var rss, pss uint64
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			switch {
			case strings.HasPrefix(line, "Rss:"):
				val, _ := strconv.ParseUint(fields[1], 10, 64)
				rss = val * 1024 // kB → bytes
			case strings.HasPrefix(line, "Pss:"):
				val, _ := strconv.ParseUint(fields[1], 10, 64)
				pss = val * 1024
			}
		}
	}
	if rss == 0 && pss == 0 { 
		return MemoryUsage{}, errors.New("empty smaps_rollup") 
	}
	return MemoryUsage{RSSBytes: rss, PSSBytes: pss}, nil
}

func readStatusRSS(pid int) (MemoryUsage, error) {
	f, err := os.Open(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil { 
		return MemoryUsage{}, err 
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "VmRSS:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				val, _ := strconv.ParseUint(fields[1], 10, 64)
				return MemoryUsage{RSSBytes: val * 1024}, nil
			}
		}
	}
	return MemoryUsage{}, errors.New("VmRSS not found")
}

func readStatmRSS(pid int) (MemoryUsage, error) {
	b, err := os.ReadFile(fmt.Sprintf("/proc/%d/statm", pid))
	if err != nil { 
		return MemoryUsage{}, err 
	}
	fields := strings.Fields(string(b))
	if len(fields) < 2 { 
		return MemoryUsage{}, errors.New("bad statm") 
	}
	pages, _ := strconv.ParseUint(fields[1], 10, 64)
	pageSize := uint64(os.Getpagesize())
	return MemoryUsage{RSSBytes: pages * pageSize}, nil
}