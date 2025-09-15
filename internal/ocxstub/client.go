package ocxstub

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Client struct {
	Server string
}

func New(s string) *Client {
	return &Client{Server: s}
}

type Offer struct {
	ID           string
	PricePerHour float64
}

type Order struct {
	ID string
}

type Lease struct {
	ID, InstanceID, Address, SSHUser string
}

func (c *Client) CreateOffer(p float64) *Offer {
	return &Offer{ID: id("offer"), PricePerHour: p}
}

func (c *Client) PlaceOrder(offerID string, gpus, hours int, budget float64) *Order {
	return &Order{ID: id("order")}
}

func (c *Client) WaitMatch(orderID string, d time.Duration) {
	time.Sleep(d)
}

func (c *Client) Provision(orderID string) *Lease {
	user := os.Getenv("USER")
	ip := localIP()
	return &Lease{
		ID:         id("lease"),
		InstanceID: id("local-nvidia"),
		Address:    ip + ":22",
		SSHUser:    user,
	}
}

func (c *Client) RunWorkload() error {
	code := `#include <stdio.h>
#include <cuda_runtime.h>
int main(){
    int n; 
    cudaGetDeviceCount(&n); 
    printf("CUDA Devices: %d\n", n);
    return 0;
}`
	_ = os.WriteFile("/tmp/gpu_test.cu", []byte(code), 0644)
	if err := exec.Command("nvcc", "/tmp/gpu_test.cu", "-o", "/tmp/gpu_test").Run(); err != nil {
		return nil // best-effort, no failure if nvcc missing
	}
	_, _ = exec.Command("/tmp/gpu_test").Output()
	return nil
}

func (c *Client) Release(leaseID string) {
	// Stub implementation
}

func (c *Client) Settle(orderID string, amount float64) {
	// Stub implementation
}

func id(p string) string {
	return fmt.Sprintf("%s_%d", p, time.Now().UnixNano())
}

func localIP() string {
	out, err := exec.Command("hostname", "-I").Output()
	if err != nil {
		return "127.0.0.1"
	}
	fs := strings.Fields(string(out))
	if len(fs) > 0 {
		return fs[0]
	}
	return "127.0.0.1"
}
