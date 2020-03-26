// Copyright (c) 2020 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package stake

import (
	"math/rand"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/decred/dcrd/chaincfg/chainhash"
	"github.com/decred/dcrd/chaincfg/v2"
	"github.com/decred/dcrd/dcrec/secp256k1/v2"
	"github.com/decred/dcrd/dcrutil/v3"
	"github.com/decred/dcrd/txscript/v3"
	"github.com/decred/dcrd/wire"
)

// Private and public keys for tests.
var (
	// Serialized private key.
	privateKey []byte = []byte{
		0x76, 0x87, 0x56, 0x13, 0x94, 0xcc, 0xc6, 0x11,
		0x01, 0x51, 0xbd, 0x9f, 0x26, 0xd4, 0x22, 0x8e,
		0xb2, 0xd5, 0x7b, 0xe1, 0x28, 0xc0, 0x36, 0x12,
		0xe3, 0x9a, 0x84, 0x4a, 0x3e, 0xcd, 0x3c, 0xcf,
	}
	// Serialized compressed public key
	publicKey []byte = []byte{
		0x02, 0xa4, 0xf6, 0x45, 0x86, 0xe1, 0x72, 0xc3,
		0xd9, 0xa2, 0x0c, 0xfa, 0x6c, 0x7a, 0xc8, 0xfb,
		0x12, 0xf0, 0x11, 0x5b, 0x3f, 0x69, 0xc3, 0xc3,
		0x5a, 0xec, 0x93, 0x3a, 0x4c, 0x47, 0xc7, 0xd9,
		0x2c,
	}
	// OP_TSPEND OP_DATA_33 publicKey
	tspendValidKey []byte = []byte{
		0xc2, // OP_TSPEND
		0x21, // OP_DATA_33
		0x02, 0xa4, 0xf6, 0x45, 0x86, 0xe1, 0x72, 0xc3,
		0xd9, 0xa2, 0x0c, 0xfa, 0x6c, 0x7a, 0xc8, 0xfb,
		0x12, 0xf0, 0x11, 0x5b, 0x3f, 0x69, 0xc3, 0xc3,
		0x5a, 0xec, 0x93, 0x3a, 0x4c, 0x47, 0xc7, 0xd9,
		0x2c,
	}
)

// TestTreasuryIsFunctions goes through all valid treasury opcode combinations.
//
// == User sends to treasury ==
// TxIn:  Normal TxIn signature script
// TxOut[0] OP_TADD
// TxOut[1] optional OP_SSTXCHANGE
//
// == Treasurybase add ==
// TxIn:  Stakebase
// TxOut[0] OP_TADD
// TxOut[1] OP_RETURN <random>
//
// == Spend from treasury ==
// TxIn[0]     OP_TSPEND <pi compressed pub key>
// TxOut[0]    OP_RETURN <random>
// TxOut[1..N] OP_TGEN <paytopubkeyhash || paytoscripthash>
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
				msgTx.AddTxOut(wire.NewTxOut(0, script))
				return msgTx
			},
			is:       IsTAdd,
			expected: true,
			check:    checkTAdd,
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
				msgTx.AddTxOut(wire.NewTxOut(0, opretScript))

				// OP_TGEN
				p2shOpTrueAddr, err := dcrutil.NewAddressScriptHash([]byte{txscript.OP_TRUE},
					chaincfg.MainNetParams())
				p2shOpTrueScript, err := txscript.PayToAddrScript(p2shOpTrueAddr)
				if err != nil {
					panic(err)
				}
				script := make([]byte, len(p2shOpTrueScript)+1)
				script[0] = txscript.OP_TGEN
				copy(script[1:], p2shOpTrueScript)
				msgTx.AddTxOut(wire.NewTxOut(0, script))

				// tspend
				key, err := secp256k1.GeneratePrivateKey()
				if err != nil {
					panic(err)
				}
				builder = txscript.NewScriptBuilder()
				builder.AddOp(txscript.OP_TSPEND)
				builder.AddData(key.PubKey().Serialize())
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
				t.Logf("%v", spew.Sdump(msgTx))

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
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // 35 bytes
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00,
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
		0xc2, // OP_TSPEND
		0x21, // OP_DATA_33

		0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // pubkey
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00,
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
	Version: 1,
	TxIn:    []*wire.TxIn{},
	TxOut: []*wire.TxOut{
		&wire.TxOut{}, // 2 TxOuts is valid
		&wire.TxOut{},
	},
	LockTime: 0,
	Expiry:   0,
}

// tspendInvalidOutCount has a valid TxIn count but an invalid TxOut count.
var tspendInvalidOutCount = &wire.MsgTx{
	SerType: wire.TxSerializeFull,
	Version: 1,
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
	Version: 1,
	TxIn: []*wire.TxIn{
		&tspendTxInNoPubkey,
	},
	TxOut: []*wire.TxOut{
		&wire.TxOut{Version: 0},
		&wire.TxOut{Version: 1},
	},
	LockTime: 0,
	Expiry:   0,
}

// tspendInvalidSignature has an invalid version in the in script
var tspendInvalidSignature = &wire.MsgTx{
	SerType: wire.TxSerializeFull,
	Version: 1,
	TxIn: []*wire.TxIn{
		&tspendTxInNoPubkey,
	},
	TxOut: []*wire.TxOut{
		&wire.TxOut{},
		&wire.TxOut{},
	},
	LockTime: 0,
	Expiry:   0,
}

// tspendInvalidSignature2 has an invalid version in the in script
var tspendInvalidSignature2 = &wire.MsgTx{
	SerType: wire.TxSerializeFull,
	Version: 1,
	TxIn: []*wire.TxIn{
		&tspendTxInInvalidPubkey,
	},
	TxOut: []*wire.TxOut{
		&wire.TxOut{},
		&wire.TxOut{},
	},
	LockTime: 0,
	Expiry:   0,
}

// tspendInvalidOpcode has an invalid opcode in the first TxIn.
var tspendInvalidOpcode = &wire.MsgTx{
	SerType: wire.TxSerializeFull,
	Version: 1,
	TxIn: []*wire.TxIn{
		&tspendTxInInvalidOpcode,
	},
	TxOut: []*wire.TxOut{
		&wire.TxOut{},
		&wire.TxOut{},
	},
	LockTime: 0,
	Expiry:   0,
}

// tspendInvalidPubkey has an invalid public key on the TSPEND.
var tspendInvalidPubkey = &wire.MsgTx{
	SerType: wire.TxSerializeFull,
	Version: 1,
	TxIn: []*wire.TxIn{
		&tspendTxInInvalidPubkey2,
	},
	TxOut: []*wire.TxOut{
		&wire.TxOut{},
		&wire.TxOut{},
	},
	LockTime: 0,
	Expiry:   0,
}

// tspendInvalidTGenLength has an invalid TxOut that has a zero length.
var tspendInvalidTGenLength = &wire.MsgTx{
	SerType: wire.TxSerializeFull,
	Version: 1,
	TxIn: []*wire.TxIn{
		&tspendTxInValidPubkey,
	},
	TxOut: []*wire.TxOut{
		&tspendTxOutValidReturn,
		&wire.TxOut{},
	},
	LockTime: 0,
	Expiry:   0,
}

// tspendInvalidTGen has an invalid TxOut that isn't tagged with an OP_TGEN.
var tspendInvalidTGen = &wire.MsgTx{
	SerType: wire.TxSerializeFull,
	Version: 1,
	TxIn: []*wire.TxIn{
		&tspendTxInValidPubkey,
	},
	TxOut: []*wire.TxOut{
		&tspendTxOutValidReturn,
		&wire.TxOut{
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
	Version: 1,
	TxIn: []*wire.TxIn{
		&tspendTxInValidPubkey,
	},
	TxOut: []*wire.TxOut{
		&tspendTxOutValidReturn,
		&wire.TxOut{
			PkScript: []byte{
				0xc3, // OP_TGEN
				0x00, // Invalid P2SH
			}},
	},
	LockTime: 0,
	Expiry:   0,
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
			name:     "tspendInvalidTGenLength",
			tx:       tspendInvalidTGenLength,
			expected: RuleError{ErrorCode: ErrTSpendInvalidTGenLength},
		},
		{
			name:     "tspendInvalidTGen",
			tx:       tspendInvalidTGen,
			expected: RuleError{ErrorCode: ErrTSpendInvalidTGen},
		},
		{
			name:     "tspendInvalidP2SH",
			tx:       tspendInvalidP2SH,
			expected: RuleError{ErrorCode: ErrTSpendInvalidP2SH},
		},
	}
	for i, tt := range tests {
		test := dcrutil.NewTx(tt.tx)
		test.SetTree(wire.TxTreeStake)
		test.SetIndex(0)
		err := checkTSpend(test.MsgTx())
		if err.(RuleError).GetCode() != tt.expected.(RuleError).GetCode() {
			spew.Dump(tt.tx)
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
	Version:  1,
	TxIn:     []*wire.TxIn{},
	TxOut:    []*wire.TxOut{},
	LockTime: 0,
	Expiry:   0,
}

// taddInvalidOutCount2 has a valid TxIn count but an invalid TxOut count.
var taddInvalidOutCount2 = &wire.MsgTx{
	SerType: wire.TxSerializeFull,
	Version: 1,
	TxIn:    []*wire.TxIn{},
	TxOut: []*wire.TxOut{
		&wire.TxOut{},
		&wire.TxOut{},
		&wire.TxOut{},
	},
	LockTime: 0,
	Expiry:   0,
}

// taddInvalidVersion has an invalid out script version.
var taddInvalidVersion = &wire.MsgTx{
	SerType: wire.TxSerializeFull,
	Version: 1,
	TxIn:    []*wire.TxIn{},
	TxOut: []*wire.TxOut{
		&wire.TxOut{Version: 0},
		&wire.TxOut{Version: 2},
	},
	LockTime: 0,
	Expiry:   0,
}

// taddInvalidLength has an invalid out script.
var taddInvalidLength = &wire.MsgTx{
	SerType: wire.TxSerializeFull,
	Version: 1,
	TxIn:    []*wire.TxIn{},
	TxOut: []*wire.TxOut{
		&wire.TxOut{},
	},
	LockTime: 0,
	Expiry:   0,
}

// taddInvalidLength has an invalid out script opcode.
var taddInvalidOpcode = &wire.MsgTx{
	SerType: wire.TxSerializeFull,
	Version: 1,
	TxIn:    []*wire.TxIn{},
	TxOut: []*wire.TxOut{
		&wire.TxOut{
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
	Version: 1,
	TxIn:    []*wire.TxIn{},
	TxOut: []*wire.TxOut{
		&wire.TxOut{
			PkScript: []byte{
				0xc1, // OP_TADD
			},
		},
		&wire.TxOut{
			PkScript: []byte{},
		},
	},
	LockTime: 0,
	Expiry:   0,
}

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
	}
	for i, tt := range tests {
		test := dcrutil.NewTx(tt.tx)
		test.SetTree(wire.TxTreeStake)
		test.SetIndex(0)
		err := checkTAdd(test.MsgTx())
		if err.(RuleError).GetCode() != tt.expected.(RuleError).GetCode() {
			spew.Dump(tt.tx)
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
