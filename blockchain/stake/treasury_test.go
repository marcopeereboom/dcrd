// Copyright (c) 2020 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package stake

import (
	"math/rand"
	"testing"

	"github.com/decred/dcrd/chaincfg/chainhash"
	"github.com/decred/dcrd/txscript/v3"
	"github.com/decred/dcrd/wire"
)

// TestTreasuryIsFunctions goes through all valid treasury opcode combinations.
//
// == User sends to treasury ==
// In:  Normal TxIn signature script
// Out: OP_TADD and optional OP_SSTXCHANGE
//
// == Treasurybase add ==
// In:  Stakebase
// Out: OP_TADD, OP_RETURN <random>
//
// == Spend from treasury ==
// In:  OP_TSPEND OP_PUSH <random>
// Out: one or more OP_TGEN <paytopubkeyhash || paytoscripthash>
//
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

				builder = txscript.NewScriptBuilder()
				builder.AddOp(txscript.OP_SSTXCHANGE)
				// XXX add stake change here
				script, err = builder.Script()
				if err != nil {
					panic(err)
				}
				msgTx.AddTxOut(wire.NewTxOut(0, script))
				return msgTx
			},
			is:       IsTAdd,
			expected: true,
			check:    checkTAdd,
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
