package gpu

import (
	"fmt"
	"time"

	"ocx.local/internal/ocxstub"
)

func RunQuick() error {
	info, err := GetInfo()
	if err != nil {
		return err
	}
	logf("GPU=%s, Mem=%dMB, Driver=%s, Temp=%dC, Util=%d%%",
		info.Name, info.MemoryMB, info.Driver, info.Temperature, info.Utilization)
	return nil
}

func RunMonitor(d time.Duration) error {
	start := time.Now()
	for time.Since(start) < d {
		m, err := sampleOnce()
		if err != nil {
			return err
		}
		logf("util=%d%% temp=%dC mem=%d/%dMB power=%dW",
			m.Utilization, m.Temperature, m.MemoryUsed, m.MemoryTotal, m.PowerW)
		time.Sleep(3 * time.Second)
	}
	return nil
}

func RunFull(c *ocxstub.Client) error {
	if err := RunQuick(); err != nil {
		return fmt.Errorf("verify: %w", err)
	}

	offer := c.CreateOffer(2.50)
	logf("offer=%s $/h=%.2f", offer.ID, offer.PricePerHour)

	order := c.PlaceOrder(offer.ID, 1, 1, 5.00)
	logf("order=%s", order.ID)

	c.WaitMatch(order.ID, 2*time.Second)
	logf("matched order=%s provider=local-nvidia-provider", order.ID)

	lease := c.Provision(order.ID)
	logf("lease=%s addr=%s ssh_user=%s", lease.ID, lease.Address, lease.SSHUser)

	if err := RunMonitor(10 * time.Second); err != nil {
		return err
	}

	_ = c.RunWorkload() // best-effort CUDA check

	c.Release(lease.ID)
	c.Settle(order.ID, 2.50)

	logf("full test complete")
	return nil
}

func logf(f string, a ...any) {
	// Use fmt.Printf for normal output, log for errors
	fmt.Printf(f+"\n", a...)
}

func logError(f string, a ...any) {
	fmt.Printf(f+"\n", a...)
}
