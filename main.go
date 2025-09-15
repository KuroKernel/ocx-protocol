// main.go — OCX Protocol Server
// go 1.22+

package main

import (
	"flag"
)

func main() {
	var port = flag.String("port", "8080", "Port to listen on")
	flag.Parse()

	gateway := NewGateway()
	gateway.StartServer(*port)
}
