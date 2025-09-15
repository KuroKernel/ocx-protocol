package gpu

import (
	"errors"
	"os/exec"
	"strconv"
	"strings"
)

type Info struct {
	Name        string
	MemoryMB    int
	Driver      string
	Temperature int
	Utilization int
}

func GetInfo() (*Info, error) {
	out, err := exec.Command("nvidia-smi",
		"--query-gpu=name,memory.total,driver_version,temperature.gpu,utilization.gpu",
		"--format=csv,noheader,nounits").Output()
	if err != nil {
		return nil, errors.New("nvidia-smi unavailable")
	}

	parts := splitCSVLine(string(out))
	if len(parts) < 5 {
		return nil, errors.New("unexpected gpu info format")
	}

	mem, _ := strconv.Atoi(parts[1])
	temp, _ := strconv.Atoi(parts[3])
	util, _ := strconv.Atoi(parts[4])

	return &Info{
		Name:        parts[0],
		MemoryMB:    mem,
		Driver:      parts[2],
		Temperature: temp,
		Utilization: util,
	}, nil
}

func splitCSVLine(s string) []string {
	line := strings.TrimSpace(s)
	fields := strings.Split(line, ",")
	for i := range fields {
		fields[i] = strings.TrimSpace(fields[i])
	}
	return fields
}
