//go:build linux
// +build linux

package deterministicvm

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

// perf_event_open syscall numbers vary by arch; syscall package routes it.
type perfEventAttr struct {
	Type               uint32
	Size               uint32
	Config             uint64
	SamplePeriod       uint64
	SampleType         uint64
	ReadFormat         uint64
	Flags              uint64
	WakeupEvents       uint32
	BpType             uint32
	BpAddr, BpLen      uint64
	BranchSampleType   uint64
	SampleRegsUser     uint64
	SampleStackUser    uint32
	ClockID            int32
	SampleRegsIntr     uint64
	AuxWatermark       uint32
	SampleMaxStack     uint16
	_                  uint16 // align
}

const (
	perfTypeHardware     = 0x0
	perfCountHWCPUCycles = 0x00

	perfFlagDisabled         = 1 << 0
	perfFlagExcludeKernel    = 1 << 5
	perfFlagExcludeHypervisor = 1 << 6
	
	// IOCTL commands for perf events
	PERF_EVENT_IOC_RESET   = 0x2403
	PERF_EVENT_IOC_ENABLE  = 0x2400
	PERF_EVENT_IOC_DISABLE = 0x2401
)

type perfCounter struct {
	fd int
}

func openPerfCycles(pid int) (*perfCounter, error) {
	attr := perfEventAttr{
		Type:     perfTypeHardware,
		Size:     uint32(unsafe.Sizeof(perfEventAttr{})),
		Config:   perfCountHWCPUCycles,
		Flags:    perfFlagDisabled | perfFlagExcludeKernel | perfFlagExcludeHypervisor,
	}
	// pid>0: per-task; cpu=-1; group_fd=-1; flags=0
	fd, _, errno := syscall.Syscall6(syscall.SYS_PERF_EVENT_OPEN,
		uintptr(unsafe.Pointer(&attr)), uintptr(pid), ^uintptr(0), ^uintptr(0), 0, 0)
	if int(fd) < 0 {
		if errno != 0 {
			return nil, errno
		}
		return nil, errors.New("perf_event_open failed")
	}
	return &perfCounter{fd: int(fd)}, nil
}

func (p *perfCounter) Close() { 
	if p.fd >= 0 { 
		syscall.Close(p.fd) 
	} 
}

func (p *perfCounter) ResetEnable() error {
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, uintptr(p.fd), PERF_EVENT_IOC_RESET, 0); e != 0 {
		return e
	}
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, uintptr(p.fd), PERF_EVENT_IOC_ENABLE, 0); e != 0 {
		return e
	}
	return nil
}

func (p *perfCounter) Disable() error {
	_, _, e := syscall.Syscall(syscall.SYS_IOCTL, uintptr(p.fd), PERF_EVENT_IOC_DISABLE, 0)
	if e != 0 { 
		return e 
	}
	return nil
}

func (p *perfCounter) ReadCount() (uint64, error) {
	var buf [8]byte
	n, err := syscall.Read(p.fd, buf[:])
	if err != nil {
		return 0, err
	}
	if n != 8 {
		return 0, fmt.Errorf("short read from perf: %d", n)
	}
	return binary.LittleEndian.Uint64(buf[:]), nil
}

// ----- Fallbacks -----

// readProcStatJiffies returns (utime+stime) jiffies from /proc/<pid>/stat
func readProcStatJiffies(pid int) (uint64, error) {
	b, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil { 
		return 0, err 
	}
	// fields: https://man7.org/linux/man-pages/man5/proc.5.html
	s := string(b)
	// comm may contain spaces in ( )
	rparen := strings.LastIndex(s, ")")
	if rparen < 0 { 
		return 0, errors.New("bad /proc stat") 
	}
	fields := strings.Fields(s[rparen+2:])
	if len(fields) < 15 { 
		return 0, errors.New("short /proc stat") 
	}
	// utime=14th, stime=15th field after comm
	ut, _ := strconv.ParseUint(fields[13], 10, 64)
	st, _ := strconv.ParseUint(fields[14], 10, 64)
	return ut + st, nil
}

func jiffiesToNanos(jiffies uint64) uint64 {
	// Most kernels: _SC_CLK_TCK = 100
	hz := uint64(100)
	return (jiffies * uint64(time.Second)) / hz
}

// readCgroupCPUUsec reads cpu.stat usage_usec for the task's cgroup (v2).
func readCgroupCPUUsec(pid int) (uint64, error) {
	path := findTaskCgroupPath(pid, "cpu")
	b, err := os.ReadFile(path + "/cpu.stat")
	if err != nil { 
		return 0, err 
	}
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "usage_usec ") {
			fields := strings.Fields(line)
			if len(fields) == 2 {
				return strconv.ParseUint(fields[1], 10, 64)
			}
		}
	}
	return 0, errors.New("usage_usec not found")
}

// findTaskCgroupPath finds the cgroup path for a given task and controller
func findTaskCgroupPath(pid int, controller string) string {
	// v2: controller ignored; tasks share unified path
	if isUnifiedV2("/sys/fs/cgroup") {
		// find effective cgroup for the task
		if b, err := os.ReadFile(fmt.Sprintf("/proc/%d/cgroup", pid)); err == nil {
			// 0::<path>
			for _, line := range strings.Split(string(b), "\n") {
				parts := strings.SplitN(line, ":", 3)
				if len(parts) == 3 && parts[0] == "0" {
					return fmt.Sprintf("/sys/fs/cgroup%s", parts[2])
				}
			}
		}
		return "/sys/fs/cgroup"
	}
	// v1: per-controller mount
	return fmt.Sprintf("/sys/fs/cgroup/%s", controller)
}

// isUnifiedV2 checks if the system is using cgroup v2
func isUnifiedV2(root string) bool {
	// v2 has cgroup.controllers
	_, err := os.Stat(fmt.Sprintf("%s/cgroup.controllers", root))
	return err == nil
}