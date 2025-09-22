package main

import (
	"fmt"
	"sort"
)

func main() {
	keys := []string{"cycles", "input_hash", "issuer_id", "meta", "output_hash", "program_hash", "sig_alg", "signature", "timestamp_ms", "version"}
	sort.Strings(keys)
	fmt.Println(keys)
}
