// Copyright (c) 2019 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package stake

import (
	"github.com/decred/dcrd/txscript/v3"
	"github.com/decred/dcrd/wire"
)

// checkTAdd verifies that the provided MsgTx is a valid TADD.
// Note: this function does not recognize treasurybase TADDs.
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

	// only 1 stake change  output allowed.
	if len(mtx.TxOut) == 2 {
		if txscript.GetScriptClass(mtx.TxOut[1].Version,
			mtx.TxOut[1].PkScript) != txscript.StakeSubChangeTy {
			return stakeRuleError(ErrTreasuryTAddInvalid,
				"second output must be an OP_SSTXCHANGE script")
		}
	}

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
	// A TADD consists of one OP_TADD in PkScript[0] followed by an
	// OP_RETURN <random> in  PkScript[1].
	if len(mtx.TxOut) != 2 {
		return stakeRuleError(ErrTreasuryTAddInvalid,
			"invalid treasurybase script")
	}

	// Verify all TxOut script versions.
	for k := range mtx.TxOut {
		if mtx.TxOut[k].Version != consensusVersion {
			return stakeRuleError(ErrTreasuryTAddInvalid,
				"invalid script version found in treasurybase")
		}
	}

	// First output must be a TADD
	if len(mtx.TxOut[0].PkScript) != 1 ||
		mtx.TxOut[0].PkScript[0] != txscript.OP_TADD {
		return stakeRuleError(ErrTreasuryTAddInvalid,
			"first treasurybase output must be a TADD")
	}

	// only 1 stake change  output allowed.
	if mtx.TxOut[1].PkScript[0] != txscript.OP_RETURN {
		return stakeRuleError(ErrTreasuryTAddInvalid,
			"second output must be an OP_RETURN script")
	}

	// Look for random 32 byte value.
	tokenizer := txscript.MakeScriptTokenizer(mtx.TxOut[1].Version,
		mtx.TxOut[1].PkScript[1:])
	if tokenizer.Next() && tokenizer.Done() &&
		tokenizer.Opcode() != txscript.OP_DATA_32 &&
		len(tokenizer.Data()) != 32 {
		return stakeRuleError(ErrTreasuryTAddInvalid,
			"second output must be an OP_RETURN script followed "+
				"by 32 bytes")
	}

	// Make sure chainhash etc is treasurybase.

	return nil
}

// IsTreasuryBase returns true if the provided transaction is a treasury base
// transaction.
func IsTreasuryBase(tx *wire.MsgTx) bool {
	return checkTreasuryBase(tx) == nil
}
