//go:build linux
// +build linux

package deterministicvm

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// CgroupLimits defines resource limits for a cgroup
type CgroupLimits struct {
	CPUQuotaMicros   int64  // e.g., 100000 = 100ms per 100ms (1 cpu)
	MemoryMaxBytes   int64  // bytes (<=0 means unlimited)
	PidsMax          int64  // <=0 unlimited
	IOReadBpsMax     string // optional: "8:0 10485760"
	IOWriteBpsMax    string // optional
}

// CgroupManager manages cgroup v2/v1 resources
type CgroupManager struct {
	root string // e.g., /sys/fs/cgroup
	v2   bool
	path string // full cgroup path for this vm
}

// NewCgroupManager creates a new cgroup manager
func NewCgroupManager(name string) (*CgroupManager, error) {
	root := "/sys/fs/cgroup"
	v2 := isUnifiedV2(root)
	path := filepath.Join(root, name)
	if err := os.MkdirAll(path, 0o755); err != nil && !os.IsExist(err) {
		return nil, fmt.Errorf("cgroup mkdir: %w", err)
	}
	return &CgroupManager{root: root, v2: v2, path: path}, nil
}

// Apply applies resource limits to a process
func (m *CgroupManager) Apply(pid int, lim CgroupLimits) error {
	if m.v2 {
		return m.applyV2(pid, lim)
	}
	return m.applyV1(pid, lim)
}

// applyV2 applies limits using cgroup v2
func (m *CgroupManager) applyV2(pid int, lim CgroupLimits) error {
	// enable controllers
	_ = os.WriteFile(filepath.Join(m.path, "cgroup.subtree_control"), []byte("+cpu +memory +pids +io"), 0o644)

	// CPU
	if lim.CPUQuotaMicros > 0 {
		// cpu.max: "<max> <period>"; e.g., "100000 100000" for 1 CPU
		val := fmt.Sprintf("%d 100000", lim.CPUQuotaMicros)
		if err := os.WriteFile(filepath.Join(m.path, "cpu.max"), []byte(val), 0o644); err != nil {
			return err
		}
	} else {
		_ = os.WriteFile(filepath.Join(m.path, "cpu.max"), []byte("max 100000"), 0o644)
	}

	// Memory
	if lim.MemoryMaxBytes > 0 {
		if err := os.WriteFile(filepath.Join(m.path, "memory.max"), []byte(strconv.FormatInt(lim.MemoryMaxBytes, 10)), 0o644); err != nil {
			return err
		}
	} else {
		_ = os.WriteFile(filepath.Join(m.path, "memory.max"), []byte("max"), 0o644)
	}

	// PIDs
	if lim.PidsMax > 0 {
		if err := os.WriteFile(filepath.Join(m.path, "pids.max"), []byte(strconv.FormatInt(lim.PidsMax, 10)), 0o644); err != nil {
			return err
		}
	} else {
		_ = os.WriteFile(filepath.Join(m.path, "pids.max"), []byte("max"), 0o644)
	}

	// IO (optional)
	if lim.IOReadBpsMax != "" {
		_ = os.WriteFile(filepath.Join(m.path, "io.max"), []byte("rbps "+lim.IOReadBpsMax+"\n"), 0o644)
	}
	if lim.IOWriteBpsMax != "" {
		_ = os.WriteFile(filepath.Join(m.path, "io.max"), []byte("wbps "+lim.IOWriteBpsMax+"\n"), 0o644)
	}

	// Add task
	return os.WriteFile(filepath.Join(m.path, "cgroup.procs"), []byte(strconv.Itoa(pid)), 0o644)
}

// applyV1 applies limits using cgroup v1
func (m *CgroupManager) applyV1(pid int, lim CgroupLimits) error {
	// v1 fallback (cpu,memory,pids)
	controllers := []string{"cpu", "memory", "pids"}
	for _, c := range controllers {
		base := filepath.Join(m.root, c, "xm-"+filepath.Base(m.path))
		if err := os.MkdirAll(base, 0o755); err != nil && !os.IsExist(err) {
			return err
		}
		switch c {
		case "cpu":
			// quota in us; period default 100000
			if lim.CPUQuotaMicros > 0 {
				_ = os.WriteFile(filepath.Join(base, "cpu.cfs_quota_us"), []byte(strconv.FormatInt(lim.CPUQuotaMicros, 10)), 0o644)
				_ = os.WriteFile(filepath.Join(base, "cpu.cfs_period_us"), []byte("100000"), 0o644)
			} else {
				_ = os.WriteFile(filepath.Join(base, "cpu.cfs_quota_us"), []byte("-1"), 0o644)
			}
		case "memory":
			if lim.MemoryMaxBytes > 0 {
				_ = os.WriteFile(filepath.Join(base, "memory.limit_in_bytes"), []byte(strconv.FormatInt(lim.MemoryMaxBytes, 10)), 0o644)
			} else {
				_ = os.WriteFile(filepath.Join(base, "memory.limit_in_bytes"), []byte("-1"), 0o644)
			}
		case "pids":
			if lim.PidsMax > 0 {
				_ = os.WriteFile(filepath.Join(base, "pids.max"), []byte(strconv.FormatInt(lim.PidsMax, 10)), 0o644)
			} else {
				_ = os.WriteFile(filepath.Join(base, "pids.max"), []byte("max"), 0o644)
			}
		}
		// attach task
		if err := os.WriteFile(filepath.Join(base, "tasks"), []byte(strconv.Itoa(pid)), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// GetPath returns the cgroup path
func (m *CgroupManager) GetPath() string {
	return m.path
}

// Cleanup removes the cgroup
func (m *CgroupManager) Cleanup() error {
	return os.RemoveAll(m.path)
}

// GetMemoryUsage returns current memory usage from cgroup
func (m *CgroupManager) GetMemoryUsage() (uint64, error) {
	if m.v2 {
		return m.getMemoryUsageV2()
	}
	return m.getMemoryUsageV1()
}

// getMemoryUsageV2 reads memory usage from cgroup v2
func (m *CgroupManager) getMemoryUsageV2() (uint64, error) {
	data, err := os.ReadFile(filepath.Join(m.path, "memory.current"))
	if err != nil {
		return 0, err
	}
	usage, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0, err
	}
	return usage, nil
}

// getMemoryUsageV1 reads memory usage from cgroup v1
func (m *CgroupManager) getMemoryUsageV1() (uint64, error) {
	data, err := os.ReadFile(filepath.Join(m.root, "memory", "xm-"+filepath.Base(m.path), "memory.usage_in_bytes"))
	if err != nil {
		return 0, err
	}
	usage, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0, err
	}
	return usage, nil
}