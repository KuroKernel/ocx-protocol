module ocx.local

go 1.18

require (
	github.com/fxamacker/cbor/v2 v2.4.0
	github.com/lib/pq v1.10.9
	github.com/mattn/go-sqlite3 v1.14.32
)

require github.com/x448/float16 v0.8.4 // indirect

replace ocx.local/pkg/executor => ./pkg/executor

replace ocx.local/pkg/programs => ./pkg/programs

replace ocx.local/pkg/ocx => ./pkg/ocx

replace ocx.local/pkg/receipt => ./pkg/receipt

replace ocx.local/conformance => ./conformance

replace ocx.local/store => ./store
