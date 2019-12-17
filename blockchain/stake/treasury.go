// Copyright (c) 2019 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package stake

import (
	"github.com/decred/dcrd/blockchain/stake/v2/internal/dbnamespace"
	"github.com/decred/dcrd/blockchain/stake/v2/internal/ticketdb"
	"github.com/decred/dcrd/database/v2"
	"github.com/decred/dcrd/dcrutil/v2"
	"github.com/decred/dcrd/txscript/v2"
	"github.com/decred/dcrd/wire"
)

const (
	MaxOutputsPerTAdd = 1
)

// checkTAdd verifies that the provided MsgTx is a valid TADD.
func checkTAdd(mtx *wire.MsgTx) error {
	// A TADD consists of one OP_TADD in PkScript[0] followed by 0 or more
	// stake change outputs.

	// First output must be a TADD
	if len(mtx.TxOut) == 0 {
		return stakeRuleError(ErrTreasuryTAddInvalid,
			"invalid TADD script")
	}
	if len(mtx.TxOut[0].PkScript) != 1 ||
		mtx.TxOut[0].PkScript[0] != txscript.OP_TADD {
		return stakeRuleError(ErrTreasuryTAddInvalid,
			"invalid TADD script")
	}

	// Make sure we only have stake change outputs.
	for _, tx := range mtx.TxOut[1:] {
		if txscript.GetScriptClass(tx.Version, tx.PkScript) !=
			txscript.StakeSubChangeTy {
			return stakeRuleError(ErrTreasuryTAddInvalid,
				"invalid TADD script")
		}
	}

	// XXX add more rules here

	return nil
}

// IsTAdd returns true if the provided transaction is a proper TADD.
func IsTAdd(tx *wire.MsgTx) bool {
	return checkTAdd(tx) == nil
}

// checkTSpend verifies if a MsgTx is a valid TSPEND.
func checkTSpend(mtx *wire.MsgTx) error {
	// XXX this is not right but we need a stub

	// A TSPEND consists of one OP_TSPEND in PkScript[0] followed by a
	// signature of sorts.
	// The remaining outputs should be of the stake gen type.

	// First output must be a TSPEND
	if len(mtx.TxOut) == 0 {
		return stakeRuleError(ErrTreasuryTSpendInvalid,
			"invalid TSPEND script")
	}
	if len(mtx.TxOut[0].PkScript) != 1 ||
		mtx.TxOut[0].PkScript[0] != txscript.OP_TSPEND {
		return stakeRuleError(ErrTreasuryTSpendInvalid,
			"invalid TSPEND script")
	}

	// XXX add more rules here

	return nil
}

// IsTSpend returns true if the provided transaction is a proper TSPEND.
func IsTSpend(tx *wire.MsgTx) bool {
	return checkTSpend(tx) == nil
}

// AddTreasuryBucket creates the treasury database if it doesn't exist.
func AddTreasuryBucket(db database.DB) error {
	return db.Update(func(dbTx database.Tx) error {
		_, err := dbTx.Metadata().CreateBucketIfNotExists(dbnamespace.TreasuryBucketName)
		return err
	})
}

// WriteTreasury inserts the current balance and the future treasury add/spend
// into the database.
func WriteTreasury(dbTx database.Tx, block *dcrutil.Block) error {
	msgBlock := block.MsgBlock()
	ts := dbnamespace.TreasuryState{
		Balance: 0, // XXX
		Values:  make([]int64, 0, len(msgBlock.Transactions)*2),
	}
	for _, v := range msgBlock.STransactions {
		if IsTAdd(v) {
			// This is a TAdd, pull values out of block.
			for _, vv := range v.TxOut {
				ts.Values = append(ts.Values, vv.Value)
			}
			continue
		}
		if IsTSpend(v) {
			// This is a TSpend, pull values out of block.
			for _, vv := range v.TxIn {
				ts.Values = append(ts.Values, -vv.ValueIn)
			}
			continue
		}
	}

	hash := block.Hash()
	return ticketdb.DbPutTreasury(dbTx, *hash, ts)
}
