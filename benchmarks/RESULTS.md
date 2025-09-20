goos: linux
goarch: amd64
pkg: ocx.local/benchmarks
cpu: 13th Gen Intel(R) Core(TM) i5-13400F
BenchmarkReceiptGeneration-16         	 4039209	       299.1 ns/op	      64 B/op	       1 allocs/op
BenchmarkReceiptVerification-16       	1000000000	         0.5433 ns/op	       0 B/op	       0 allocs/op
BenchmarkDeterministicExecution-16    	  568468	      2124 ns/op	       0 B/op	       0 allocs/op
PASS
ok  	ocx.local/benchmarks	3.341s
