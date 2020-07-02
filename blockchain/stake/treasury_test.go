// Copyright (c) 2020 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package stake

import (
	"bytes"
	"encoding/hex"
	"math"
	"math/rand"
	"testing"

	"github.com/decred/dcrd/chaincfg/chainhash"
	"github.com/decred/dcrd/chaincfg/v3"
	"github.com/decred/dcrd/dcrutil/v3"
	"github.com/decred/dcrd/txscript/v3"
	"github.com/decred/dcrd/wire"
)

// Private and public keys for tests.
var (
	// Serialized private key.
	//privateKey []byte = []byte{
	//	0x76, 0x87, 0x56, 0x13, 0x94, 0xcc, 0xc6, 0x11,
	//	0x01, 0x51, 0xbd, 0x9f, 0x26, 0xd4, 0x22, 0x8e,
	//	0xb2, 0xd5, 0x7b, 0xe1, 0x28, 0xc0, 0x36, 0x12,
	//	0xe3, 0x9a, 0x84, 0x4a, 0x3e, 0xcd, 0x3c, 0xcf,
	//}

	// Serialized compressed public key
	publicKey []byte = []byte{
		0x02, 0xa4, 0xf6, 0x45, 0x86, 0xe1, 0x72, 0xc3,
		0xd9, 0xa2, 0x0c, 0xfa, 0x6c, 0x7a, 0xc8, 0xfb,
		0x12, 0xf0, 0x11, 0x5b, 0x3f, 0x69, 0xc3, 0xc3,
		0x5a, 0xec, 0x93, 0x3a, 0x4c, 0x47, 0xc7, 0xd9,
		0x2c,
	}

	// Valid signature of chainhash.HashB([]byte("test message"))
	validSignature []byte = []byte{
		0x30, 0x45, 0x02, 0x21, 0x00, 0xc5, 0xab, 0xff,
		0xe9, 0x56, 0xf7, 0x70, 0x9c, 0x18, 0x30, 0x23,
		0xf7, 0xd9, 0xef, 0x49, 0xd0, 0xd0, 0x55, 0x0d,
		0xcd, 0x19, 0xef, 0xf1, 0x34, 0x6d, 0x6e, 0xcd,
		0xb1, 0x67, 0xa0, 0xe4, 0x0b, 0x02, 0x20, 0x42,
		0xd5, 0x01, 0x10, 0xfe, 0xac, 0x73, 0x6c, 0x63,
		0x2b, 0x6b, 0xfd, 0x8d, 0x21, 0x5a, 0x1a, 0x94,
		0x48, 0x85, 0x87, 0x8d, 0xfb, 0x49, 0xde, 0x09,
		0x50, 0xf4, 0x7a, 0xf6, 0xff, 0xf9, 0xc3,
		0x01, // SigHashAll
	}

	// OP_DATA_72 <signature> <pikey> OP_SSPEND
	tspendValidKey []byte = []byte{
		0x48, // OP_DATA_72 valid signature + sighash
		0x30, 0x45, 0x02, 0x21, 0x00, 0xc5, 0xab, 0xff,
		0xe9, 0x56, 0xf7, 0x70, 0x9c, 0x18, 0x30, 0x23,
		0xf7, 0xd9, 0xef, 0x49, 0xd0, 0xd0, 0x55, 0x0d,
		0xcd, 0x19, 0xef, 0xf1, 0x34, 0x6d, 0x6e, 0xcd,
		0xb1, 0x67, 0xa0, 0xe4, 0x0b, 0x02, 0x20, 0x42,
		0xd5, 0x01, 0x10, 0xfe, 0xac, 0x73, 0x6c, 0x63,
		0x2b, 0x6b, 0xfd, 0x8d, 0x21, 0x5a, 0x1a, 0x94,
		0x48, 0x85, 0x87, 0x8d, 0xfb, 0x49, 0xde, 0x09,
		0x50, 0xf4, 0x7a, 0xf6, 0xff, 0xf9, 0xc3,
		0x01, // SigHashAll
		0x21, // OP_DATA_33 valid public key
		0x02, 0xa4, 0xf6, 0x45, 0x86, 0xe1, 0x72, 0xc3,
		0xd9, 0xa2, 0x0c, 0xfa, 0x6c, 0x7a, 0xc8, 0xfb,
		0x12, 0xf0, 0x11, 0x5b, 0x3f, 0x69, 0xc3, 0xc3,
		0x5a, 0xec, 0x93, 0x3a, 0x4c, 0x47, 0xc7, 0xd9,
		0x2c,
		0xc2, // OP_TSPEND
	}
)

// generateKeys generates all the keys that are hard coded in this file.
//func generateKeys() {
//	key, _ := secp256k1.PrivKeyFromBytes(privateKey)
//	message := "test message"
//	messageHash := chainhash.HashB([]byte(message))
//	signature, err := key.Sign(messageHash)
//	if err != nil {
//		panic(err)
//	}
//	fmt.Printf("Sig 0x%x: %x", len(signature.Serialize()),
//		signature.Serialize())
//	for k, v := range signature.Serialize() {
//		if k%8 == 0 {
//			fmt.Printf("\n")
//		}
//		fmt.Printf("0x%02x,", v)
//	}
//	fmt.Printf("\n")
//}

//func init() {
//	generateKeys()
//	panic("x")
//}

// TestTreasuryIsFunctions goes through all valid treasury opcode combinations.
func TestTreasuryIsFunctions(t *testing.T) {
	tests := []struct {
		name     string
		createTx func() *wire.MsgTx
		is       func(*wire.MsgTx) bool
		expected bool
		check    func(*wire.MsgTx) error
	}{
		{
			name: "tadd from user, no change",
			createTx: func() *wire.MsgTx {
				builder := txscript.NewScriptBuilder()
				builder.AddOp(txscript.OP_TADD)
				script, err := builder.Script()
				if err != nil {
					panic(err)
				}
				msgTx := wire.NewMsgTx()
				msgTx.Version = wire.TxVersionTreasury
				msgTx.AddTxOut(wire.NewTxOut(0, script))
				return msgTx
			},
			is:       IsTAdd,
			expected: true,
			check:    checkTAdd,
		},
		{
			name: "check tadd from user, no change with istreasurybase",
			createTx: func() *wire.MsgTx {
				builder := txscript.NewScriptBuilder()
				builder.AddOp(txscript.OP_TADD)
				script, err := builder.Script()
				if err != nil {
					panic(err)
				}
				msgTx := wire.NewMsgTx()
				msgTx.Version = wire.TxVersionTreasury
				msgTx.AddTxOut(wire.NewTxOut(0, script))
				return msgTx
			},
			is:       IsTreasuryBase,
			expected: false,
			check:    checkTreasuryBase,
		},
		{
			// This is a valid stakebase but NOT a valid TADD.
			name: "tadd from user, with OP_RETURN",
			createTx: func() *wire.MsgTx {
				builder := txscript.NewScriptBuilder()
				builder.AddOp(txscript.OP_TADD)
				script, err := builder.Script()
				if err != nil {
					panic(err)
				}
				msgTx := wire.NewMsgTx()
				msgTx.Version = wire.TxVersionTreasury
				msgTx.AddTxOut(wire.NewTxOut(0, script))

				// OP_RETURN <data>
				payload := make([]byte, chainhash.HashSize)
				_, err = rand.Read(payload)
				if err != nil {
					panic(err)
				}
				builder = txscript.NewScriptBuilder()
				builder.AddOp(txscript.OP_RETURN)
				builder.AddData(payload)
				script, err = builder.Script()
				if err != nil {
					panic(err)
				}
				msgTx.AddTxOut(wire.NewTxOut(0, script))

				// treasurybase
				coinbaseFlags := "/dcrd/"
				coinbaseScript := make([]byte, len(coinbaseFlags)+2)
				copy(coinbaseScript[2:], coinbaseFlags)
				msgTx.AddTxIn(&wire.TxIn{
					// Stakebase transactions have no
					// inputs, so previous outpoint is zero
					// hash and max index.
					PreviousOutPoint: *wire.NewOutPoint(&chainhash.Hash{},
						wire.MaxPrevOutIndex, wire.TxTreeRegular),
					Sequence:        wire.MaxTxInSequenceNum,
					BlockHeight:     wire.NullBlockHeight,
					BlockIndex:      wire.NullBlockIndex,
					SignatureScript: coinbaseScript,
				})
				return msgTx
			},
			is:       IsTAdd,
			expected: false,
			check:    checkTAdd,
		},
		{
			name: "tadd from user, with change",
			createTx: func() *wire.MsgTx {
				builder := txscript.NewScriptBuilder()
				builder.AddOp(txscript.OP_TADD)
				script, err := builder.Script()
				if err != nil {
					panic(err)
				}
				msgTx := wire.NewMsgTx()
				msgTx.Version = wire.TxVersionTreasury
				msgTx.AddTxOut(wire.NewTxOut(0, script))

				p2shOpTrueAddr, err := dcrutil.NewAddressScriptHash([]byte{txscript.OP_TRUE},
					chaincfg.MainNetParams())
				if err != nil {
					panic(err)
				}
				changeScript, err := txscript.PayToSStxChange(p2shOpTrueAddr)
				if err != nil {
					panic(err)
				}
				msgTx.AddTxOut(wire.NewTxOut(0, changeScript))
				return msgTx
			},
			is:       IsTAdd,
			expected: true,
			check:    checkTAdd,
		},
		{
			name: "tadd from treasurybase",
			createTx: func() *wire.MsgTx {
				builder := txscript.NewScriptBuilder()
				builder.AddOp(txscript.OP_TADD)
				script, err := builder.Script()
				if err != nil {
					panic(err)
				}
				msgTx := wire.NewMsgTx()
				msgTx.Version = wire.TxVersionTreasury
				msgTx.AddTxOut(wire.NewTxOut(0, script))

				// OP_RETURN <data>
				payload := make([]byte, 12) // extra nonce size
				_, err = rand.Read(payload)
				if err != nil {
					panic(err)
				}
				builder = txscript.NewScriptBuilder()
				builder.AddOp(txscript.OP_RETURN)
				builder.AddData(payload)
				script, err = builder.Script()
				if err != nil {
					panic(err)
				}
				msgTx.AddTxOut(wire.NewTxOut(0, script))

				// treasurybase
				coinbaseFlags := "/dcrd/"
				coinbaseScript := make([]byte, len(coinbaseFlags)+2)
				copy(coinbaseScript[2:], coinbaseFlags)
				msgTx.AddTxIn(&wire.TxIn{
					// Stakebase transactions have no
					// inputs, so previous outpoint is zero
					// hash and max index.
					PreviousOutPoint: *wire.NewOutPoint(&chainhash.Hash{},
						wire.MaxPrevOutIndex, wire.TxTreeRegular),
					Sequence:        wire.MaxTxInSequenceNum,
					BlockHeight:     wire.NullBlockHeight,
					BlockIndex:      wire.NullBlockIndex,
					SignatureScript: coinbaseScript,
				})

				return msgTx
			},
			is:       IsTreasuryBase,
			expected: true,
			check:    checkTreasuryBase,
		},
		{
			name: "check treasury base with tadd",
			createTx: func() *wire.MsgTx {
				builder := txscript.NewScriptBuilder()
				builder.AddOp(txscript.OP_TADD)
				script, err := builder.Script()
				if err != nil {
					panic(err)
				}
				msgTx := wire.NewMsgTx()
				msgTx.Version = wire.TxVersionTreasury
				msgTx.AddTxOut(wire.NewTxOut(0, script))

				// OP_RETURN <data>
				payload := make([]byte, 12) // extra nonce size
				_, err = rand.Read(payload)
				if err != nil {
					panic(err)
				}
				builder = txscript.NewScriptBuilder()
				builder.AddOp(txscript.OP_RETURN)
				builder.AddData(payload)
				script, err = builder.Script()
				if err != nil {
					panic(err)
				}
				msgTx.AddTxOut(wire.NewTxOut(0, script))

				// treasurybase
				coinbaseFlags := "/dcrd/"
				coinbaseScript := make([]byte, len(coinbaseFlags)+2)
				copy(coinbaseScript[2:], coinbaseFlags)
				msgTx.AddTxIn(&wire.TxIn{
					// Stakebase transactions have no
					// inputs, so previous outpoint is zero
					// hash and max index.
					PreviousOutPoint: *wire.NewOutPoint(&chainhash.Hash{},
						wire.MaxPrevOutIndex, wire.TxTreeRegular),
					Sequence:        wire.MaxTxInSequenceNum,
					BlockHeight:     wire.NullBlockHeight,
					BlockIndex:      wire.NullBlockIndex,
					SignatureScript: coinbaseScript,
				})

				return msgTx
			},
			is:       IsTAdd,
			expected: false,
			check:    checkTAdd,
		},
		{
			name: "tspend",
			createTx: func() *wire.MsgTx {
				// OP_RETURN <32 byte random>
				payload := make([]byte, chainhash.HashSize)
				_, err := rand.Read(payload)
				if err != nil {
					panic(err)
				}
				builder := txscript.NewScriptBuilder()
				builder.AddOp(txscript.OP_RETURN)
				builder.AddData(payload)
				opretScript, err := builder.Script()
				if err != nil {
					panic(err)
				}
				msgTx := wire.NewMsgTx()
				msgTx.Version = wire.TxVersionTreasury
				msgTx.AddTxOut(wire.NewTxOut(0, opretScript))

				// OP_TGEN
				p2shOpTrueAddr, err := dcrutil.NewAddressScriptHash([]byte{txscript.OP_TRUE},
					chaincfg.MainNetParams())
				if err != nil {
					panic(err)
				}
				p2shOpTrueScript, err := txscript.PayToAddrScript(p2shOpTrueAddr)
				if err != nil {
					panic(err)
				}
				script := make([]byte, len(p2shOpTrueScript)+1)
				script[0] = txscript.OP_TGEN
				copy(script[1:], p2shOpTrueScript)
				msgTx.AddTxOut(wire.NewTxOut(0, script))

				// tspend
				builder = txscript.NewScriptBuilder()
				builder.AddData(validSignature)
				builder.AddData(publicKey)
				builder.AddOp(txscript.OP_TSPEND)
				tspendScript, err := builder.Script()
				if err != nil {
					panic(err)
				}

				msgTx.AddTxIn(&wire.TxIn{
					// Stakebase transactions have no
					// inputs, so previous outpoint is zero
					// hash and max index.
					PreviousOutPoint: *wire.NewOutPoint(&chainhash.Hash{},
						wire.MaxPrevOutIndex, wire.TxTreeRegular),
					Sequence:        wire.MaxTxInSequenceNum,
					BlockHeight:     wire.NullBlockHeight,
					BlockIndex:      wire.NullBlockIndex,
					SignatureScript: tspendScript,
				})

				return msgTx
			},
			is:       IsTSpend,
			expected: true,
			check:    checkTSpend,
		},
	}

	for i, test := range tests {
		if got := test.is(test.createTx()); got != test.expected {
			// Obtain error
			err := test.check(test.createTx())
			t.Fatalf("%v %v: failed got %v want %v error %v",
				i, test.name, got, test.expected, err)
		}
	}
}

// tspendTxInNoPubkey
var tspendTxInNoPubkey = wire.TxIn{
	PreviousOutPoint: wire.OutPoint{
		Hash:  chainhash.Hash{},
		Index: 0xffffffff,
		Tree:  wire.TxTreeRegular,
	},
	SignatureScript: []byte{
		0xc2, // OP_TSPEND
	},
	BlockHeight: wire.NullBlockHeight,
	BlockIndex:  wire.NullBlockIndex,
	Sequence:    0xffffffff,
}

// tspendTxInInvalidPubkey is a TxIn with an invalid key on the OP_TSPEND.
var tspendTxInInvalidPubkey = wire.TxIn{
	PreviousOutPoint: wire.OutPoint{
		Hash:  chainhash.Hash{},
		Index: 0xffffffff,
		Tree:  wire.TxTreeRegular,
	},
	SignatureScript: []byte{
		0xc2, // OP_TSPEND
		0x23, // OP_DATA_35
		0x03, // Valid pubkey version
		0x00, // invalid compressed key
	},
	BlockHeight: wire.NullBlockHeight,
	BlockIndex:  wire.NullBlockIndex,
	Sequence:    0xffffffff,
}

// tspendTxInInvalidOpcode is a TxIn with an invalid opcode where OP_TSPEND was
// supposed to be.
var tspendTxInInvalidOpcode = wire.TxIn{
	PreviousOutPoint: wire.OutPoint{
		Hash:  chainhash.Hash{},
		Index: 0xffffffff,
		Tree:  wire.TxTreeRegular,
	},
	SignatureScript: []byte{
		0x48, // OP_DATA_72 valid signature + sighash
		0x30, 0x45, 0x02, 0x21, 0x00, 0xc5, 0xab, 0xff,
		0xe9, 0x56, 0xf7, 0x70, 0x9c, 0x18, 0x30, 0x23,
		0xf7, 0xd9, 0xef, 0x49, 0xd0, 0xd0, 0x55, 0x0d,
		0xcd, 0x19, 0xef, 0xf1, 0x34, 0x6d, 0x6e, 0xcd,
		0xb1, 0x67, 0xa0, 0xe4, 0x0b, 0x02, 0x20, 0x42,
		0xd5, 0x01, 0x10, 0xfe, 0xac, 0x73, 0x6c, 0x63,
		0x2b, 0x6b, 0xfd, 0x8d, 0x21, 0x5a, 0x1a, 0x94,
		0x48, 0x85, 0x87, 0x8d, 0xfb, 0x49, 0xde, 0x09,
		0x50, 0xf4, 0x7a, 0xf6, 0xff, 0xf9, 0xc3,
		0x01, // SigHashAll
		0x21, // OP_DATA_33 valid public key
		0x02, 0xa4, 0xf6, 0x45, 0x86, 0xe1, 0x72, 0xc3,
		0xd9, 0xa2, 0x0c, 0xfa, 0x6c, 0x7a, 0xc8, 0xfb,
		0x12, 0xf0, 0x11, 0x5b, 0x3f, 0x69, 0xc3, 0xc3,
		0x5a, 0xec, 0x93, 0x3a, 0x4c, 0x47, 0xc7, 0xd9,
		0x2c,
		0x6a, // OP_RETURN instead of OP_TSPEND
	},
	BlockHeight: wire.NullBlockHeight,
	BlockIndex:  wire.NullBlockIndex,
	Sequence:    0xffffffff,
}

// tspendTxInInvalidPubkey2 is a TxIn with an invalid public key on the
// OP_TSPEND.
var tspendTxInInvalidPubkey2 = wire.TxIn{
	PreviousOutPoint: wire.OutPoint{
		Hash:  chainhash.Hash{},
		Index: 0xffffffff,
		Tree:  wire.TxTreeRegular,
	},
	SignatureScript: []byte{
		0x48, // OP_DATA_72 valid signature + sighash
		0x30, 0x45, 0x02, 0x21, 0x00, 0xc5, 0xab, 0xff,
		0xe9, 0x56, 0xf7, 0x70, 0x9c, 0x18, 0x30, 0x23,
		0xf7, 0xd9, 0xef, 0x49, 0xd0, 0xd0, 0x55, 0x0d,
		0xcd, 0x19, 0xef, 0xf1, 0x34, 0x6d, 0x6e, 0xcd,
		0xb1, 0x67, 0xa0, 0xe4, 0x0b, 0x02, 0x20, 0x42,
		0xd5, 0x01, 0x10, 0xfe, 0xac, 0x73, 0x6c, 0x63,
		0x2b, 0x6b, 0xfd, 0x8d, 0x21, 0x5a, 0x1a, 0x94,
		0x48, 0x85, 0x87, 0x8d, 0xfb, 0x49, 0xde, 0x09,
		0x50, 0xf4, 0x7a, 0xf6, 0xff, 0xf9, 0xc3,
		0x01, // SigHashAll
		0x21, // OP_DATA_33 INVALID public key
		0x00, 0xa4, 0xf6, 0x45, 0x86, 0xe1, 0x72, 0xc3,
		0xd9, 0xa2, 0x0c, 0xfa, 0x6c, 0x7a, 0xc8, 0xfb,
		0x12, 0xf0, 0x11, 0x5b, 0x3f, 0x69, 0xc3, 0xc3,
		0x5a, 0xec, 0x93, 0x3a, 0x4c, 0x47, 0xc7, 0xd9,
		0x2c,
		0xc2, // OP_TSPEND
	},
	BlockHeight: wire.NullBlockHeight,
	BlockIndex:  wire.NullBlockIndex,
	Sequence:    0xffffffff,
}

var tspendTxOutValidReturn = wire.TxOut{
	Value:   500000000,
	Version: 0,
	PkScript: []byte{
		0x6a, // OP_RETURN
		0x20, // OP_DATA_32
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	},
}

var tspendTxOutInvalidReturn = wire.TxOut{
	Value:   500000000,
	Version: 0,
	PkScript: []byte{
		0x6a, // OP_RETURN
		0x20, // OP_DATA_32
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // 1 byte short
	},
}

// tspendTxInValidPubkey is a TxIn with a public key on the OP_TSPEND.
var tspendTxInValidPubkey = wire.TxIn{
	PreviousOutPoint: wire.OutPoint{
		Hash:  chainhash.Hash{},
		Index: 0xffffffff,
		Tree:  wire.TxTreeRegular,
	},
	SignatureScript: tspendValidKey,
	BlockHeight:     wire.NullBlockHeight,
	BlockIndex:      wire.NullBlockIndex,
	Sequence:        0xffffffff,
}

// tspendInvalidInCount has an invalid TxIn count but a valid TxOut count.
var tspendInvalidInCount = &wire.MsgTx{
	SerType: wire.TxSerializeFull,
	Version: 3,
	TxIn:    []*wire.TxIn{},
	TxOut: []*wire.TxOut{
		{}, // 2 TxOuts is valid
		{},
	},
	LockTime: 0,
	Expiry:   0,
}

// tspendInvalidOutCount has a valid TxIn count but an invalid TxOut count.
var tspendInvalidOutCount = &wire.MsgTx{
	SerType: wire.TxSerializeFull,
	Version: 3,
	TxIn: []*wire.TxIn{
		&tspendTxInNoPubkey,
	},
	TxOut:    []*wire.TxOut{},
	LockTime: 0,
	Expiry:   0,
}

// tspendInvalidVersion has an invalid version in an out script
var tspendInvalidVersion = &wire.MsgTx{
	SerType: wire.TxSerializeFull,
	Version: 3,
	TxIn: []*wire.TxIn{
		&tspendTxInNoPubkey,
	},
	TxOut: []*wire.TxOut{
		{
			Version: 0,
			PkScript: []byte{
				0x6a, // OP_RETURN
			},
		},
		{
			Version: 1, // Fail
			PkScript: []byte{
				0xc3, // OP_TGEN
			},
		},
	},
	LockTime: 0,
	Expiry:   0,
}

// tspendInvalidSignature has an invalid version in the in script
var tspendInvalidSignature = &wire.MsgTx{
	SerType: wire.TxSerializeFull,
	Version: 3,
	TxIn: []*wire.TxIn{
		&tspendTxInNoPubkey,
	},
	TxOut: []*wire.TxOut{
		{
			PkScript: []byte{
				0x6a, // OP_RETURN
			},
		},
		{
			PkScript: []byte{
				0xc3, // OP_TGEN
			},
		},
	},
	LockTime: 0,
	Expiry:   0,
}

// tspendInvalidSignature2 has an invalid version in the in script
var tspendInvalidSignature2 = &wire.MsgTx{
	SerType: wire.TxSerializeFull,
	Version: 3,
	TxIn: []*wire.TxIn{
		&tspendTxInInvalidPubkey,
	},
	TxOut: []*wire.TxOut{
		{
			PkScript: []byte{
				0x6a, // OP_RETURN
			},
		},
		{
			PkScript: []byte{
				0xc3, // OP_TGEN
			},
		},
	},
	LockTime: 0,
	Expiry:   0,
}

// tspendInvalidOpcode has an invalid opcode in the first TxIn.
var tspendInvalidOpcode = &wire.MsgTx{
	SerType: wire.TxSerializeFull,
	Version: 3,
	TxIn: []*wire.TxIn{
		&tspendTxInInvalidOpcode,
	},
	TxOut: []*wire.TxOut{
		{
			PkScript: []byte{
				0x6a, // OP_RETURN
			},
		},
		{
			PkScript: []byte{
				0xc3, // OP_TGEN
			},
		},
	},
	LockTime: 0,
	Expiry:   0,
}

// tspendInvalidPubkey has an invalid public key on the TSPEND.
var tspendInvalidPubkey = &wire.MsgTx{
	SerType: wire.TxSerializeFull,
	Version: 3,
	TxIn: []*wire.TxIn{
		&tspendTxInInvalidPubkey2,
	},
	TxOut: []*wire.TxOut{
		{
			PkScript: []byte{
				0x6a, // OP_RETURN
			},
		},
		{
			PkScript: []byte{
				0xc3, // OP_TGEN
			},
		},
	},
	LockTime: 0,
	Expiry:   0,
}

// tspendInvalidScriptLength has an invalid TxOut that has a zero length.
var tspendInvalidScriptLength = &wire.MsgTx{
	SerType: wire.TxSerializeFull,
	Version: 3,
	TxIn: []*wire.TxIn{
		&tspendTxInValidPubkey,
	},
	TxOut: []*wire.TxOut{
		&tspendTxOutValidReturn,
		{},
	},
	LockTime: 0,
	Expiry:   0,
}

// tspendInvalidTransaction has an invalid hash on the OP_RETURN.
var tspendInvalidTransaction = &wire.MsgTx{
	SerType: wire.TxSerializeFull,
	Version: 3,
	TxIn: []*wire.TxIn{
		&tspendTxInValidPubkey,
	},
	TxOut: []*wire.TxOut{
		&tspendTxOutInvalidReturn,
		{
			PkScript: []byte{
				0xc3, // OP_TGEN
			},
		},
	},
	LockTime: 0,
	Expiry:   0,
}

// tspendInvalidTGen has an invalid TxOut that isn't tagged with an OP_TGEN.
var tspendInvalidTGen = &wire.MsgTx{
	SerType: wire.TxSerializeFull,
	Version: 3,
	TxIn: []*wire.TxIn{
		&tspendTxInValidPubkey,
	},
	TxOut: []*wire.TxOut{
		&tspendTxOutValidReturn,
		{
			PkScript: []byte{
				0x6a, // OP_RETURN instead of OP_TGEN
			}},
	},
	LockTime: 0,
	Expiry:   0,
}

// tspendInvalidP2SH has an invalid TxOut that doesn't have a valid P2SH
// script.
var tspendInvalidP2SH = &wire.MsgTx{
	SerType: wire.TxSerializeFull,
	Version: 3,
	TxIn: []*wire.TxIn{
		&tspendTxInValidPubkey,
	},
	TxOut: []*wire.TxOut{
		&tspendTxOutValidReturn,
		{
			PkScript: []byte{
				0xc3, // OP_TGEN
				0x00, // Invalid P2SH
			}},
	},
	LockTime: 0,
	Expiry:   0,
}

var tspendInvalidTxVersion = &wire.MsgTx{
	SerType: wire.TxSerializeFull,
	Version: 1, // Invalid version
	TxIn: []*wire.TxIn{
		&tspendTxInValidPubkey,
	},
	TxOut: []*wire.TxOut{
		&tspendTxOutValidReturn,
	},
	LockTime: 0,
	Expiry:   0,
}

func TestTSpendGenerated(t *testing.T) {
	rawScript := "01000000010000000000000000000000000000000000000000000000000000000000000000ffffffff00ffffffff0300000000000000000000226a20e7c95fc357b0c22b8978c6326274c583d871fef37f1e22e63809db02e9611e130065cd1d0000000000001ac376a91449a6061cda6d9243cecf3186af60caefc613904988ac0027b9290000000000001ac376a9145722802cd0905b4589d2094fcb6b76f746dcf18988ac000000000000000001c29786470000000000000000ffffffff6c483045022100e2b945d276d99e35e6b07e4ab03a6d99a159f5d1a9f3ece67af642c8d09d7f730220033694255a4912c8531039eb2bd9b02332f92eda95c781ffe91e820a616aba60012102a36b785d584555696b69d1b2bbeff4010332b301e3edd316d79438554cacb3e7c2"
	s, err := hex.DecodeString(rawScript)
	if err != nil {
		t.Fatal(err)
	}
	var tx wire.MsgTx
	err = tx.Deserialize(bytes.NewReader(s))
	if err != nil {
		t.Fatalf("Deserialize: %v", err)
	}
	tx.Version = wire.TxVersionTreasury

	err = checkTSpend(&tx)
	if err != nil {
		t.Fatalf("checkTSpend: %v", err)
	}

}

func TestTSpendErrors(t *testing.T) {
	tests := []struct {
		name     string
		tx       *wire.MsgTx
		expected error
	}{
		{
			name:     "tspendInvalidOutCount",
			tx:       tspendInvalidOutCount,
			expected: RuleError{ErrorCode: ErrTSpendInvalidLength},
		},
		{
			name:     "tspendInvalidInCount",
			tx:       tspendInvalidInCount,
			expected: RuleError{ErrorCode: ErrTSpendInvalidLength},
		},
		{
			name:     "tspendInvalidVersion",
			tx:       tspendInvalidVersion,
			expected: RuleError{ErrorCode: ErrTSpendInvalidVersion},
		},
		{
			name:     "tspendInvalidSignature",
			tx:       tspendInvalidSignature,
			expected: RuleError{ErrorCode: ErrTSpendInvalidSignature},
		},
		{
			name:     "tspendInvalidSignature2",
			tx:       tspendInvalidSignature2,
			expected: RuleError{ErrorCode: ErrTSpendInvalidSignature},
		},
		{
			name:     "tspendInvalidOpcode",
			tx:       tspendInvalidOpcode,
			expected: RuleError{ErrorCode: ErrTSpendInvalidOpcode},
		},
		{
			name:     "tspendInvalidPubkey",
			tx:       tspendInvalidPubkey,
			expected: RuleError{ErrorCode: ErrTSpendInvalidPubkey},
		},
		{
			name:     "tspendInvalidScriptLength",
			tx:       tspendInvalidScriptLength,
			expected: RuleError{ErrorCode: ErrTSpendInvalidScriptLength},
		},
		{
			name:     "tspendInvalidTransaction",
			tx:       tspendInvalidTransaction,
			expected: RuleError{ErrorCode: ErrTSpendInvalidTransaction},
		},
		{
			name:     "tspendInvalidTGen",
			tx:       tspendInvalidTGen,
			expected: RuleError{ErrorCode: ErrTSpendInvalidTGen},
		},
		{
			name:     "tspendInvalidP2SH",
			tx:       tspendInvalidP2SH,
			expected: RuleError{ErrorCode: ErrTSpendInvalidSpendScript},
		},
		{
			name:     "tspendInvalidTxVersion",
			tx:       tspendInvalidTxVersion,
			expected: RuleError{ErrorCode: ErrTSpendInvalidTxVersion},
		},
	}
	for i, tt := range tests {
		test := dcrutil.NewTx(tt.tx)
		test.SetTree(wire.TxTreeStake)
		test.SetIndex(0)
		err := checkTSpend(test.MsgTx())
		if err.(RuleError).GetCode() != tt.expected.(RuleError).GetCode() {
			t.Errorf("%v: checkTSpend should have returned %v but "+
				"instead returned %v: %v",
				tt.name, tt.expected.(RuleError).GetCode(),
				err.(RuleError).GetCode(), err)
		}
		if IsTSpend(test.MsgTx()) {
			t.Errorf("IsTSpend claimed an invalid tspend is valid"+
				" %v %v", i, tt.name)
		}
	}
}

// taddInvalidOutCount has a valid TxIn count but an invalid TxOut count.
var taddInvalidOutCount = &wire.MsgTx{
	SerType:  wire.TxSerializeFull,
	Version:  3,
	TxIn:     []*wire.TxIn{},
	TxOut:    []*wire.TxOut{},
	LockTime: 0,
	Expiry:   0,
}

// taddInvalidOutCount2 has a valid TxIn count but an invalid TxOut count.
var taddInvalidOutCount2 = &wire.MsgTx{
	SerType: wire.TxSerializeFull,
	Version: 3,
	TxIn:    []*wire.TxIn{},
	TxOut: []*wire.TxOut{
		{},
		{},
		{},
	},
	LockTime: 0,
	Expiry:   0,
}

// taddInvalidVersion has an invalid out script version.
var taddInvalidVersion = &wire.MsgTx{
	SerType: wire.TxSerializeFull,
	Version: 3,
	TxIn:    []*wire.TxIn{},
	TxOut: []*wire.TxOut{
		{Version: 1},
		{Version: 0},
	},
	LockTime: 0,
	Expiry:   0,
}

// taddInvalidScriptLength has a zero script length.
var taddInvalidScriptLength = &wire.MsgTx{
	SerType: wire.TxSerializeFull,
	Version: 3,
	TxIn:    []*wire.TxIn{},
	TxOut: []*wire.TxOut{
		{Version: 0},
		{Version: 0},
	},
	LockTime: 0,
	Expiry:   0,
}

// taddInvalidLength has an invalid out script.
var taddInvalidLength = &wire.MsgTx{
	SerType: wire.TxSerializeFull,
	Version: 3,
	TxIn:    []*wire.TxIn{},
	TxOut: []*wire.TxOut{
		{PkScript: []byte{
			0xc2, // OP_TSPEND instead of OP_TADD
			0x00, // Fail length test
		}},
	},
	LockTime: 0,
	Expiry:   0,
}

// taddInvalidLength has an invalid out script opcode.
var taddInvalidOpcode = &wire.MsgTx{
	SerType: wire.TxSerializeFull,
	Version: 3,
	TxIn:    []*wire.TxIn{},
	TxOut: []*wire.TxOut{
		{
			PkScript: []byte{
				0xc2, // OP_TSPEND instead of OP_TADD
			},
		},
	},
	LockTime: 0,
	Expiry:   0,
}

// taddInvalidChange has an invalid out chnage script.
var taddInvalidChange = &wire.MsgTx{
	SerType: wire.TxSerializeFull,
	Version: 3,
	TxIn:    []*wire.TxIn{},
	TxOut: []*wire.TxOut{
		{
			PkScript: []byte{
				0xc1, // OP_TADD
			},
		},
		{
			PkScript: []byte{
				0x00, // Not OP_SSTXCHANGE
			},
		},
	},
	LockTime: 0,
	Expiry:   0,
}

// taddInvalidTxVersion has an invalid transaction version.
var taddInvalidTxVersion = &wire.MsgTx{
	SerType: wire.TxSerializeFull,
	Version: 1, // Invalid
	TxIn:    []*wire.TxIn{},
	TxOut: []*wire.TxOut{
		{
			PkScript: []byte{
				0xc1, // OP_TADD
			},
		},
	},
	LockTime: 0,
	Expiry:   0,
}

// TestTAddErrors verifies that all TAdd errors can be hit and return the
// proper error.
func TestTAddErrors(t *testing.T) {
	tests := []struct {
		name     string
		tx       *wire.MsgTx
		expected error
	}{
		{
			name:     "taddInvalidOutCount",
			tx:       taddInvalidOutCount,
			expected: RuleError{ErrorCode: ErrTAddInvalidCount},
		},
		{
			name:     "taddInvalidOutCount2",
			tx:       taddInvalidOutCount2,
			expected: RuleError{ErrorCode: ErrTAddInvalidCount},
		},
		{
			name:     "taddInvalidVersion",
			tx:       taddInvalidVersion,
			expected: RuleError{ErrorCode: ErrTAddInvalidVersion},
		},
		{
			name:     "taddInvalidScriptLength",
			tx:       taddInvalidScriptLength,
			expected: RuleError{ErrorCode: ErrTAddInvalidScriptLength},
		},
		{
			name:     "taddInvalidLength",
			tx:       taddInvalidLength,
			expected: RuleError{ErrorCode: ErrTAddInvalidLength},
		},
		{
			name:     "taddInvalidOpcode",
			tx:       taddInvalidOpcode,
			expected: RuleError{ErrorCode: ErrTAddInvalidOpcode},
		},
		{
			name:     "taddInvalidChange",
			tx:       taddInvalidChange,
			expected: RuleError{ErrorCode: ErrTAddInvalidChange},
		},
		{
			name:     "taddInvalidTxVersion",
			tx:       taddInvalidTxVersion,
			expected: RuleError{ErrorCode: ErrTAddInvalidTxVersion},
		},
	}
	for i, tt := range tests {
		test := dcrutil.NewTx(tt.tx)
		test.SetTree(wire.TxTreeStake)
		test.SetIndex(0)
		err := checkTAdd(test.MsgTx())
		if err.(RuleError).GetCode() != tt.expected.(RuleError).GetCode() {
			t.Errorf("%v: checkTAdd should have returned %v but "+
				"instead returned %v: %v",
				tt.name, tt.expected.(RuleError).GetCode(),
				err.(RuleError).GetCode(), err)
		}
		if IsTAdd(test.MsgTx()) {
			t.Errorf("IsTAdd claimed an invalid tadd is valid"+
				" %v %v", i, tt.name)
		}
	}
}

// treasurybaseInvalidInCount has an invalid TxIn count.
var treasurybaseInvalidInCount = &wire.MsgTx{
	SerType: wire.TxSerializeFull,
	Version: 3,
	TxIn:    []*wire.TxIn{},
	TxOut: []*wire.TxOut{
		{},
		{},
	},
	LockTime: 0,
	Expiry:   0,
}

// treasurybaseInvalidOutCount has an invalid TxOut count.
var treasurybaseInvalidOutCount = &wire.MsgTx{
	SerType: wire.TxSerializeFull,
	Version: 3,
	TxIn: []*wire.TxIn{
		{},
	},
	TxOut:    []*wire.TxOut{},
	LockTime: 0,
	Expiry:   0,
}

// treasurybaseInvalidVersion has an invalid out script version.
var treasurybaseInvalidVersion = &wire.MsgTx{
	SerType: wire.TxSerializeFull,
	Version: 3,
	TxIn: []*wire.TxIn{
		{},
	},
	TxOut: []*wire.TxOut{
		{Version: 0},
		{Version: 2},
	},
	LockTime: 0,
	Expiry:   0,
}

// treasurybaseInvalidOpcode0 has an invalid out script opcode.
var treasurybaseInvalidOpcode0 = &wire.MsgTx{
	SerType: wire.TxSerializeFull,
	Version: 3,
	TxIn: []*wire.TxIn{
		{},
	},
	TxOut: []*wire.TxOut{
		{
			PkScript: []byte{
				0xc2, // OP_TSPEND instead of OP_TADD
			},
		},
		{
			PkScript: []byte{
				0x6a, // OP_RETURN
				0x0c, // OP_DATA_12
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00,
			},
		},
	},
	LockTime: 0,
	Expiry:   0,
}

// treasurybaseInvalidOpcode0Len has an invalid out script opcode length.
var treasurybaseInvalidOpcode0Len = &wire.MsgTx{
	SerType: wire.TxSerializeFull,
	Version: 3,
	TxIn: []*wire.TxIn{
		{},
	},
	TxOut: []*wire.TxOut{
		{
			PkScript: nil, // Invalid
		},
		{
			PkScript: []byte{
				0x6a, // OP_RETURN
				0x0c, // OP_DATA_12
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00,
			},
		},
	},
	LockTime: 0,
	Expiry:   0,
}

// treasurybaseInvalidOpcode1 has an invalid out script opcode.
var treasurybaseInvalidOpcode1 = &wire.MsgTx{
	SerType: wire.TxSerializeFull,
	Version: 3,
	TxIn: []*wire.TxIn{
		{},
	},
	TxOut: []*wire.TxOut{
		{
			PkScript: []byte{
				0xc1, // OP_TADD
			},
		},
		{
			PkScript: []byte{
				0xc1, // OP_TADD instead of OP_RETURN
				0x0c, // OP_DATA_32
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00,
			},
		},
	},
	LockTime: 0,
	Expiry:   0,
}

// treasurybaseInvalidOpcode1Len has an invalid out script opcode length.
var treasurybaseInvalidOpcode1Len = &wire.MsgTx{
	SerType: wire.TxSerializeFull,
	Version: 3,
	TxIn: []*wire.TxIn{
		{},
	},
	TxOut: []*wire.TxOut{
		{
			PkScript: []byte{
				0xc1, // OP_TADD
			},
		},
		{
			PkScript: nil,
		},
	},
	LockTime: 0,
	Expiry:   0,
}

// treasurybaseInvalidOpcodeDataPush has an invalid out script data push in
// script 1 opcode 1.
var treasurybaseInvalidOpcodeDataPush = &wire.MsgTx{
	SerType: wire.TxSerializeFull,
	Version: 3,
	TxIn: []*wire.TxIn{
		{},
	},
	TxOut: []*wire.TxOut{
		{
			PkScript: []byte{
				0xc1, // OP_TADD
			},
		},
		{
			PkScript: []byte{
				0x6a, // OP_RETURN
				0x0d, // OP_DATA_13 instead of OP_DATA_12
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00,
			},
		},
	},
	LockTime: 0,
	Expiry:   0,
}

// treasurybaseInvalid has invalid in script constants.
var treasurybaseInvalid = &wire.MsgTx{
	SerType: wire.TxSerializeFull,
	Version: 3,
	TxIn: []*wire.TxIn{
		{
			PreviousOutPoint: wire.OutPoint{
				Index: math.MaxUint32 - 1,
			},
		},
	},
	TxOut: []*wire.TxOut{
		{
			PkScript: []byte{
				0xc1, // OP_TADD
			},
		},
		{
			PkScript: []byte{
				0x6a, // OP_RETURN
				0x0c, // OP_DATA_12
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00,
			},
		},
	},
	LockTime: 0,
	Expiry:   0,
}

// treasurybaseInvalid2 has invalid in script constants.
var treasurybaseInvalid2 = &wire.MsgTx{
	SerType: wire.TxSerializeFull,
	Version: 3,
	TxIn: []*wire.TxIn{
		{
			PreviousOutPoint: wire.OutPoint{
				Index: math.MaxUint32,
				Hash:  chainhash.Hash{'m', 'o', 'o'},
			},
		},
	},
	TxOut: []*wire.TxOut{
		{
			PkScript: []byte{
				0xc1, // OP_TADD
			},
		},
		{
			PkScript: []byte{
				0x6a, // OP_RETURN
				0x0c, // OP_DATA_12
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00,
			},
		},
	},
	LockTime: 0,
	Expiry:   0,
}

// treasurybaseInvalidTxVersion has an invalid transaction version.
var treasurybaseInvalidTxVersion = &wire.MsgTx{
	SerType: wire.TxSerializeFull,
	Version: 1, // Invalid
	TxIn: []*wire.TxIn{
		{
			PreviousOutPoint: wire.OutPoint{
				Index: math.MaxUint32,
				Hash:  chainhash.Hash{'m', 'o', 'o'},
			},
		},
	},
	TxOut: []*wire.TxOut{
		{
			PkScript: []byte{
				0xc1, // OP_TADD
			},
		},
		{
			PkScript: []byte{
				0x6a, // OP_RETURN
				0x0c, // OP_DATA_12
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00,
			},
		},
	},
	LockTime: 0,
	Expiry:   0,
}

// TestTreasuryIsFunctions verifies that all TAdd errors can be hit and return the
// proper error.
func TestTreasuryBaseErrors(t *testing.T) {
	tests := []struct {
		name     string
		tx       *wire.MsgTx
		expected error
	}{
		{
			name:     "treasurybaseInvalidInCount",
			tx:       treasurybaseInvalidInCount,
			expected: RuleError{ErrorCode: ErrTreasuryBaseInvalidCount},
		},
		{
			name:     "treasurybaseInvalidOutCount",
			tx:       treasurybaseInvalidOutCount,
			expected: RuleError{ErrorCode: ErrTreasuryBaseInvalidCount},
		},
		{
			name:     "treasurybaseInvalidVersion",
			tx:       treasurybaseInvalidVersion,
			expected: RuleError{ErrorCode: ErrTreasuryBaseInvalidVersion},
		},
		{
			name:     "treasurybaseInvalidOpcode0",
			tx:       treasurybaseInvalidOpcode0,
			expected: RuleError{ErrorCode: ErrTreasuryBaseInvalidOpcode0},
		},
		{
			name:     "treasurybaseInvalidOpcode0Len",
			tx:       treasurybaseInvalidOpcode0Len,
			expected: RuleError{ErrorCode: ErrTreasuryBaseInvalidOpcode0},
		},
		{
			name:     "treasurybaseInvalidOpcode1",
			tx:       treasurybaseInvalidOpcode1,
			expected: RuleError{ErrorCode: ErrTreasuryBaseInvalidOpcode1},
		},
		{
			name:     "treasurybaseInvalidOpcode1Len",
			tx:       treasurybaseInvalidOpcode1Len,
			expected: RuleError{ErrorCode: ErrTreasuryBaseInvalidOpcode1},
		},
		{
			name:     "treasurybaseInvalidDataPush",
			tx:       treasurybaseInvalidOpcodeDataPush,
			expected: RuleError{ErrorCode: ErrTreasuryBaseInvalidOpcode1},
		},
		{
			name:     "treasurybaseInvalid",
			tx:       treasurybaseInvalid,
			expected: RuleError{ErrorCode: ErrTreasuryBaseInvalid},
		},
		{
			name:     "treasurybaseInvalid2",
			tx:       treasurybaseInvalid2,
			expected: RuleError{ErrorCode: ErrTreasuryBaseInvalid},
		},
		{
			name:     "treasurybaseInvalidTxVersion",
			tx:       treasurybaseInvalidTxVersion,
			expected: RuleError{ErrorCode: ErrTreasuryBaseInvalidTxVersion},
		},
	}
	for i, tt := range tests {
		test := dcrutil.NewTx(tt.tx)
		test.SetTree(wire.TxTreeStake)
		test.SetIndex(0)
		err := checkTreasuryBase(test.MsgTx())
		if err.(RuleError).GetCode() != tt.expected.(RuleError).GetCode() {
			t.Errorf("%v: checkTreasuryBase should have returned "+
				"%v but instead returned %v: %v",
				tt.name, tt.expected.(RuleError).GetCode(),
				err.(RuleError).GetCode(), err)
		}
		if IsTreasuryBase(test.MsgTx()) {
			t.Errorf("IsTreasuryBase claimed an invalid treasury "+
				"base is valid %v %v", i, tt.name)
		}
	}
}
