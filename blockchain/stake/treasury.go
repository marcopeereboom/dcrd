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

	// Verify all TxOut script versions.
	for k := range mtx.TxOut {
		if mtx.TxOut[k].Version != consensusVersion {
			return stakeRuleError(ErrTreasuryTAddInvalid,
				"invalid script version found in TADD TxOut")
		}
	}

	// First output must be a TADD
	if len(mtx.TxOut[0].PkScript) != 1 ||
		mtx.TxOut[0].PkScript[0] != txscript.OP_TADD {
		return stakeRuleError(ErrTreasuryTAddInvalid,
			"first output must be a TADD")
	}

	// only 1 stake change or op_return output allowed.
	if len(mtx.TxOut) == 2 {
		tx := mtx.TxOut[1]
		if !(txscript.GetScriptClass(tx.Version, tx.PkScript) ==
			txscript.StakeSubChangeTy ||
			mtx.TxOut[1].PkScript[0] == txscript.OP_RETURN) {
			return stakeRuleError(ErrTreasuryTAddInvalid,
				"second output must be OP_RETURN or OP_SSTXCHANGE")
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

	// A TSPEND consists of one OP_TSPEND in TxIn[0].SignatureScript,
	// one OP_RETURN transaction hash and at least one P2PH TxOut script.
	if len(mtx.TxIn) != 1 || len(mtx.TxOut) <= 0 {
		return stakeRuleError(ErrTreasuryTAddInvalid,
			"invalid TSPEND script")
	}

	// Check to make sure that all output scripts are the consensus version.
	for _, txOut := range mtx.TxOut {
		if txOut.Version != consensusVersion {
			return stakeRuleError(ErrTreasuryTSpendInvalid,
				"invalid script version found in txOut")
		}
	}

	// Verify there is a TSPEND in SignatureScript.
	if len(mtx.TxIn[0].SignatureScript) != 1 ||
		mtx.TxIn[0].SignatureScript[0] != txscript.OP_TSPEND {
		return stakeRuleError(ErrTreasuryTSpendInvalid,
			"invalid TSPEND script")
	}

	// Verify that the TxOut's contains P2PH scripts.
	for k, txOut := range mtx.TxOut {
		if k == 0 {
			// Check for OP_RETURN
			if txscript.GetScriptClass(txOut.Version, txOut.PkScript) !=
				txscript.NullDataTy {
				return stakeRuleError(ErrSSGenNoReference,
					"First TSPEND output should have been "+
						"an OP_RETURN data push, but "+
						"was not")
			}
			continue
		}
		// All tx outs are prefixed with OP_TGEN
		if txOut.PkScript[0] != txscript.OP_TGEN {
			return stakeRuleError(ErrTreasuryTSpendInvalid,
				"Output is not prefixed with OP_TGEN")
		}
		sc := txscript.GetScriptClass(txOut.Version, txOut.PkScript[1:])
		if !(sc == txscript.ScriptHashTy || sc == txscript.PubKeyHashTy) {
			return stakeRuleError(ErrTreasuryTSpendInvalid,
				"Output is not P2PH")
		}
	}

	// XXX add more rules here

	return nil
}

// IsTSpend returns true if the provided transaction is a proper TSPEND.
func IsTSpend(tx *wire.MsgTx) bool {
	return checkTSpend(tx) == nil
}

// checkTreasuryBase verifies that the provided MsgTx is a treasury base.
func checkTreasuryBase(mtx *wire.MsgTx) error {
	// Reused checkTAdd for the output side since it supports bot TADD
	// variants.
	err := checkTAdd(mtx)
	if err != nil {
		return err
	}

	// We have a valid TADD so not chec the input side and determine if
	// this is stakebase.
	// A coin base must only have one transaction input.
	if len(mtx.TxIn) != 1 {
		return stakeRuleError(ErrStakeBaseInvalid,
			"Input is not a stakebase")
	}

	// The previous output of a coin base must have a max value index and a
	// zero hash.
	// XXX fix this
	//prevOut := &mtx.TxIn[0].PreviousOutPoint
	//if prevOut.Index != math.MaxUint32 || prevOut.Hash != *zeroHash ||
	//	prevOut.Tree != wire.TxTreeStake {
	//	return stakeRuleError(ErrStakeBaseInvalid,
	//		"Previous out point is not a stakebase")
	//}

	return nil
}

// IsTreasuryBase returns true if the provided transaction is a treasury base
// transaction.
func IsTreasuryBase(tx *wire.MsgTx) bool {
	return checkTreasuryBase(tx) == nil
}
