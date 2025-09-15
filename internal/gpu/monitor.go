package gpu

import (
	"errors"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type Metrics struct {
	Utilization int
	Temperature int
	MemoryUsed  int
	MemoryTotal int
	PowerW      int
	At          time.Time
}

func sampleOnce() (*Metrics, error) {
	out, err := exec.Command("nvidia-smi",
		"--query-gpu=utilization.gpu,temperature.gpu,memory.used,memory.total,power.draw",
		"--format=csv,noheader,nounits").Output()
	if err != nil {
		return nil, errors.New("nvidia-smi unavailable")
	}

	parts := splitCSV(strings.TrimSpace(string(out)))
	if len(parts) < 5 {
		return nil, errors.New("unexpected metrics format")
	}

	u, _ := strconv.Atoi(parts[0])
	t, _ := strconv.Atoi(parts[1])
	mu, _ := strconv.Atoi(parts[2])
	mt, _ := strconv.Atoi(parts[3])
	p, _ := strconv.Atoi(parts[4])

	return &Metrics{
		Utilization: u,
		Temperature: t,
		MemoryUsed:  mu,
		MemoryTotal: mt,
		PowerW:      p,
		At:          time.Now(),
	}, nil
}

func splitCSV(s string) []string {
	fs := strings.Split(s, ",")
	for i := range fs {
		fs[i] = strings.TrimSpace(fs[i])
	}
	return fs
}
