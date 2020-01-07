// Copyright (c) 2019 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package stake

import (
	"github.com/decred/dcrd/txscript/v3"
	"github.com/decred/dcrd/wire"
)

// checkTAdd verifies that the provided MsgTx is a valid TADD.
func checkTAdd(mtx *wire.MsgTx) error {
	// A TADD consists of one OP_TADD in PkScript[0] followed by 0 or 1
	// stake change outputs.
	if !(len(mtx.TxOut) == 1 || len(mtx.TxOut) == 2) {
		return stakeRuleError(ErrTreasuryTAddInvalid,
			"invalid TADD script")
	}

	for k := range mtx.TxOut {
		if mtx.TxOut[k].Version != consensusVersion {
			return stakeRuleError(ErrTreasuryTSpendInvalid,
				"invalid script version found in TADD TxOut")
		}
	}

	// First output must be a TADD
	if len(mtx.TxOut[0].PkScript) != 1 ||
		mtx.TxOut[0].PkScript[0] != txscript.OP_TADD {
		return stakeRuleError(ErrTreasuryTAddInvalid,
			"invalid TADD script")
	}

	// only 1 stake change output allowed.
	if len(mtx.TxOut) == 2 {
		tx := mtx.TxOut[1]
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

	// A TSPEND consists of one OP_TSPEND in TxIn[0].SignatureScript and
	// least P2SH TxOut script.
	if len(mtx.TxIn) != 1 || len(mtx.TxOut) <= 0 {
		return stakeRuleError(ErrTreasuryTAddInvalid,
			"invalid TSPEND script")
	}

	// Verify there is a TSPEND in SignatureScript.
	if len(mtx.TxIn[0].SignatureScript) != 1 ||
		mtx.TxIn[0].SignatureScript[0] != txscript.OP_TSPEND {
		return stakeRuleError(ErrTreasuryTSpendInvalid,
			"invalid TSPEND script")
	}

	// Check to make sure that all output scripts are the consensus version.
	for _, txOut := range mtx.TxOut {
		if txOut.Version != consensusVersion {
			return stakeRuleError(ErrTreasuryTSpendInvalid,
				"invalid script version found in txOut")
		}

		// Verify that the TxOut's contains P2SH scripts.
		sc := txscript.GetScriptClass(txOut.Version, txOut.PkScript)
		if !(sc == txscript.ScriptHashTy || sc == txscript.PubKeyHashTy) {
			return stakeRuleError(ErrTreasuryTSpendInvalid,
				"Output is not P2SH")
		}
	}

	// XXX add more rules here

	return nil
}

// IsTSpend returns true if the provided transaction is a proper TSPEND.
func IsTSpend(tx *wire.MsgTx) bool {
	return checkTSpend(tx) == nil
}
