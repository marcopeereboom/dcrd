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
	"github.com/decred/dcrd/dcrutil/v3"
	"github.com/decred/dcrd/txscript/v3"
	"github.com/decred/dcrd/wire"
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
// TxIn[0]     OP_TSPEND
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
				builder = txscript.NewScriptBuilder()
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
