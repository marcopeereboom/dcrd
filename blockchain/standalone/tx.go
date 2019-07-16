// Copyright (c) 2013-2016 The btcsuite developers
// Copyright (c) 2015-2019 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package standalone

import (
	"bytes"
	"math"

	"github.com/decred/dcrd/chaincfg/chainhash"
	"github.com/decred/dcrd/txscript/v3"
	"github.com/decred/dcrd/wire"
)

var (
	// zeroHash is the zero value for a chainhash.Hash and is defined as a
	// package level variable to avoid the need to create a new instance every
	// time a check is needed.
	zeroHash = chainhash.Hash{}
)

// IsCoinBaseTx determines whether or not a transaction is a coinbase.  A
// coinbase is a special transaction created by miners that has no inputs.
// This is represented in the block chain by a transaction with a single input
// that has a previous output transaction index set to the maximum value along
// with a zero hash.
func IsCoinBaseTx(tx *wire.MsgTx, isTreasuryEnabled bool) bool {
	// A coin base must only have one transaction input.
	if len(tx.TxIn) != 1 {
		return false
	}

	// The previous output of a coin base must have a max value index and a
	// zero hash.
	prevOut := &tx.TxIn[0].PreviousOutPoint
	if prevOut.Index != math.MaxUint32 || prevOut.Hash != zeroHash {
		return false
	}

	// We need to do additional testing when treasury is enabled or a
	// TSPEND will be recognized as a coinbase transaction. We will rely on
	// the fact that a pre-treasury coinbase has at least 2 outputs that
	// start with an OP_RETURN and at least one OP_TGEN. We also check the
	// last TxIn[0].SignatureScript byte for an OP_TSPEND.
	if isTreasuryEnabled {
		if len(tx.TxOut) < 2 {
			return false
		}
		l := len(tx.TxIn[0].SignatureScript)
		if tx.TxIn[0].SignatureScript[l-1] == txscript.OP_TSPEND &&
			tx.TxOut[0].PkScript[0] == txscript.OP_RETURN &&
			tx.TxOut[1].PkScript[0] == txscript.OP_TGEN {
			return false
		}
	}

	return true
}

// IsTreasuryBase does a minimal check to see if a transaction is a treasury
// base.
func IsTreasuryBase(tx *wire.MsgTx) bool {
	if len(tx.TxIn) != 1 || len(tx.TxOut) != 2 {
		return false
	}

	if len(tx.TxOut[0].PkScript) != 1 ||
		tx.TxOut[0].PkScript[0] != txscript.OP_TADD {
		return false
	}

	if len(tx.TxOut[1].PkScript) != 14 ||
		tx.TxOut[1].PkScript[0] != txscript.OP_RETURN ||
		tx.TxOut[1].PkScript[1] != txscript.OP_DATA_12 {
		return false
	}

	prevOut := &tx.TxIn[0].PreviousOutPoint
	if prevOut.Index != math.MaxUint32 ||
		!bytes.Equal(prevOut.Hash[:], zeroHash[:]) {
		return false
	}

	return true
}
