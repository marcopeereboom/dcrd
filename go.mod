module github.com/decred/dcrd

go 1.13

require (
	github.com/btcsuite/winsvc v1.0.0
	github.com/decred/base58 v1.0.3
	github.com/decred/dcrd/addrmgr v1.1.0
	github.com/decred/dcrd/bech32 v1.0.0
	github.com/decred/dcrd/blockchain/stake/v3 v3.0.0-20200215031403-6b2ce76f0986
	github.com/decred/dcrd/blockchain/standalone/v2 v2.0.0
	github.com/decred/dcrd/blockchain/v3 v3.0.0-20200215031403-6b2ce76f0986
	github.com/decred/dcrd/certgen v1.1.0
	github.com/decred/dcrd/chaincfg/chainhash v1.0.2
	github.com/decred/dcrd/chaincfg/v3 v3.0.0-20200215031403-6b2ce76f0986
	github.com/decred/dcrd/connmgr/v3 v3.0.0-20200215031403-6b2ce76f0986
	github.com/decred/dcrd/crypto/ripemd160 v1.0.0
	github.com/decred/dcrd/database/v2 v2.0.1
	github.com/decred/dcrd/dcrec v1.0.0
	github.com/decred/dcrd/dcrec/secp256k1/v3 v3.0.0-20200608124004-b2f67c2dc475
	github.com/decred/dcrd/dcrjson/v3 v3.0.1
	github.com/decred/dcrd/dcrutil/v3 v3.0.0-20200503044000-76f6906e50e5
	github.com/decred/dcrd/fees/v2 v2.0.0
	github.com/decred/dcrd/gcs/v2 v2.0.1
	github.com/decred/dcrd/hdkeychain/v3 v3.0.0
	github.com/decred/dcrd/lru v1.0.0
	github.com/decred/dcrd/mempool/v4 v4.0.0-20200215031403-6b2ce76f0986
	github.com/decred/dcrd/mining/v3 v3.0.0-20200215031403-6b2ce76f0986
	github.com/decred/dcrd/peer/v2 v2.1.0
	github.com/decred/dcrd/rpc/jsonrpc/types/v2 v2.0.1-0.20200503044000-76f6906e50e5
	github.com/decred/dcrd/rpcclient/v6 v6.0.0-20200215031403-6b2ce76f0986
	github.com/decred/dcrd/txscript/v3 v3.0.0
	github.com/decred/dcrd/wire v1.3.0
	github.com/decred/go-socks v1.1.0
	github.com/decred/slog v1.0.0
	github.com/gorilla/websocket v1.4.2
	github.com/jessevdk/go-flags v1.4.0
	github.com/jrick/bitset v1.0.0
	github.com/jrick/logrotate v1.0.0
	github.com/kr/pretty v0.1.0 // indirect
	golang.org/x/crypto v0.0.0-20190820162420-60c769a6c586
	golang.org/x/net v0.0.0-20190813141303-74dc4d7220e7 // indirect
	golang.org/x/sync v0.0.0-20190423024810-112230192c58
	golang.org/x/text v0.3.2 // indirect
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
)

replace (
	github.com/decred/dcrd/addrmgr => ./addrmgr
	github.com/decred/dcrd/bech32 => ./bech32
	github.com/decred/dcrd/blockchain/stake/v3 => ./blockchain/stake
	github.com/decred/dcrd/blockchain/standalone/v2 => ./blockchain/standalone
	github.com/decred/dcrd/blockchain/v3 => ./blockchain
	github.com/decred/dcrd/certgen => ./certgen
	github.com/decred/dcrd/chaincfg/chainhash => ./chaincfg/chainhash
	github.com/decred/dcrd/chaincfg/v3 => ./chaincfg
	github.com/decred/dcrd/connmgr/v3 => ./connmgr
	github.com/decred/dcrd/crypto/blake256 => ./crypto/blake256
	github.com/decred/dcrd/crypto/ripemd160 => ./crypto/ripemd160
	github.com/decred/dcrd/database/v2 => ./database
	github.com/decred/dcrd/dcrec => ./dcrec
	github.com/decred/dcrd/dcrec/secp256k1/v3 => ./dcrec/secp256k1
	github.com/decred/dcrd/dcrjson/v3 => ./dcrjson
	github.com/decred/dcrd/dcrutil/v3 => ./dcrutil
	github.com/decred/dcrd/fees/v2 => ./fees
	github.com/decred/dcrd/gcs/v2 => ./gcs
	github.com/decred/dcrd/hdkeychain/v3 => ./hdkeychain
	github.com/decred/dcrd/limits => ./limits
	github.com/decred/dcrd/lru => ./lru
	github.com/decred/dcrd/mempool/v4 => ./mempool
	github.com/decred/dcrd/mining/v3 => ./mining
	github.com/decred/dcrd/peer/v2 => ./peer
	github.com/decred/dcrd/rpc/jsonrpc/types/v2 => ./rpc/jsonrpc/types
	github.com/decred/dcrd/rpcclient/v6 => ./rpcclient
	github.com/decred/dcrd/txscript/v3 => ./txscript
	github.com/decred/dcrd/wire => ./wire
)
