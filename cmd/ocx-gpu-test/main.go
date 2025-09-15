package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"time"

	"ocx.local/internal/gpu"
	"ocx.local/internal/ocxstub"
)

func main() {
	mode := flag.String("test", "quick", "quick|full|monitor")
	server := flag.String("server", "http://localhost:8080", "OCX server URL (optional)")
	monitorDur := flag.Duration("duration", 30*time.Second, "monitor duration")
	jsonLogs := flag.Bool("json", true, "json logs")
	flag.Parse()

	if *jsonLogs {
		log.SetFlags(0)
		log.SetOutput(jsonWriter{})
	}

	switch *mode {
	case "quick":
		if err := gpu.RunQuick(); err != nil {
			fatal(err)
		}
	case "full":
		c := ocxstub.New(*server)
		if err := gpu.RunFull(c); err != nil {
			fatal(err)
		}
	case "monitor":
		if err := gpu.RunMonitor(*monitorDur); err != nil {
			fatal(err)
		}
	default:
		fatalf("unknown mode: %s", *mode)
	}
}

type jsonWriter struct{}

func (jsonWriter) Write(p []byte) (int, error) {
	_ = json.NewEncoder(os.Stdout).Encode(map[string]string{
		"ts":  time.Now().Format(time.RFC3339Nano),
		"msg": string(p),
	})
	return len(p), nil
}

func fatal(err error) {
	log.Fatalf("error: %v", err)
}

func fatalf(f string, a ...any) {
	log.Fatalf(f, a...)
}
