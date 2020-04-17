module github.com/decred/dcrd/testscript

go 1.12

require (
	github.com/davecgh/go-spew v1.1.1
	github.com/dchest/blake256 v1.1.0 // indirect
	github.com/decred/dcrd/blockchain/v2 v2.0.2 // indirect
	github.com/decred/dcrd/chaincfg/chainhash v1.0.2
	github.com/decred/dcrd/txscript/v2 v2.0.0
	github.com/decred/dcrd/wire v1.2.0
	github.com/decred/slog v1.0.0
)

replace github.com/decred/dcrd/txscript/v2 => ../txscript
