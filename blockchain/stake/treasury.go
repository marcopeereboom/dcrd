// Copyright (c) 2019 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package stake

import (
	"bytes"
	"fmt"
	"math"

	"github.com/decred/dcrd/dcrec/secp256k1/v2"
	"github.com/decred/dcrd/txscript/v3"
	"github.com/decred/dcrd/wire"
)

// checkTAdd verifies that the provided MsgTx is a valid TADD.
// Note: this function does not recognize treasurybase TADDs.
func checkTAdd(mtx *wire.MsgTx) error {
	// A TADD consists of one OP_TADD in PkScript[0] followed by 0 or 1
	// stake change outputs.
	if !(len(mtx.TxOut) == 1 || len(mtx.TxOut) == 2) {
		return stakeRuleError(ErrTAddInvalidCount,
			fmt.Sprintf("invalid TADD script out count: %v",
				len(mtx.TxOut)))
	}

	// Verify all TxOut script versions and lengths.
	for k := range mtx.TxOut {
		if mtx.TxOut[k].Version != consensusVersion {
			return stakeRuleError(ErrTAddInvalidVersion,
				fmt.Sprintf("invalid script version found "+
					"in TADD TxOut: %v", k))
		}

		if len(mtx.TxOut[k].PkScript) == 0 {
			return stakeRuleError(ErrTAddInvalidScriptLength,
				fmt.Sprintf("zero script length found in "+
					"TADD: %v", k))
		}
	}

	// First output must be a TADD
	if len(mtx.TxOut[0].PkScript) != 1 {
		return stakeRuleError(ErrTAddInvalidLength,
			fmt.Sprintf("TADD script length is not 1 byte, got %v",
				len(mtx.TxOut[0].PkScript)))
	}
	if mtx.TxOut[0].PkScript[0] != txscript.OP_TADD {
		return stakeRuleError(ErrTAddInvalidOpcode,
			fmt.Sprintf("first output must be a TADD, got 0x%x",
				mtx.TxOut[0].PkScript[0]))
	}

	// only 1 stake change  output allowed.
	if len(mtx.TxOut) == 2 {
		// Script length has been already verified.
		if txscript.GetScriptClass(mtx.TxOut[1].Version,
			mtx.TxOut[1].PkScript) != txscript.StakeSubChangeTy {
			return stakeRuleError(ErrTAddInvalidChange,
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
	// A TSPEND consists of one OP_TSPEND <pi compressed pubkey> in
	// TxIn[0].SignatureScript, one OP_RETURN transaction hash and at least
	// one P2PH TxOut script.
	if len(mtx.TxIn) != 1 ||
		!(len(mtx.TxOut) == 1 || len(mtx.TxOut) == 2) {
		return stakeRuleError(ErrTSpendInvalidLength,
			fmt.Sprintf("invalid TSPEND script lengths in: %v "+
				"out: %v", len(mtx.TxIn), len(mtx.TxOut)))
	}

	// Check to make sure that all output scripts are the consensus version.
	for k, txOut := range mtx.TxOut {
		if txOut.Version != consensusVersion {
			return stakeRuleError(ErrTSpendInvalidVersion,
				fmt.Sprintf("invalid script version found in "+
					"TxOut: %v", k))
		}
	}

	// Verify expected length of SignatureScript.
	if len(mtx.TxIn[0].SignatureScript) != 35 {
		return stakeRuleError(ErrTSpendInvalidSignature,
			fmt.Sprintf("invalid TSPEND signature length: %v",
				len(mtx.TxIn[0].SignatureScript)))
	}

	// Make sure SignatureScript starts with OP_TSPEND.
	if mtx.TxIn[0].SignatureScript[0] != txscript.OP_TSPEND ||
		mtx.TxIn[0].SignatureScript[1] != txscript.OP_DATA_33 {
		return stakeRuleError(ErrTSpendInvalidOpcode,
			fmt.Sprintf("first script must start with an "+
				"OP_TSPEND followed by OP_DATA_33 but got %x %x",
				mtx.TxIn[0].SignatureScript[0],
				mtx.TxIn[0].SignatureScript[1]))
	}

	// Verify pubkey is valid.
	_, err := secp256k1.ParsePubKey(mtx.TxIn[0].SignatureScript[2:])
	if err != nil {
		return stakeRuleError(ErrTSpendInvalidPubkey,
			fmt.Sprintf("TSPEND invalid pubkey: %v", err))
	}

	// Verify that the TxOut's contains P2PH scripts.
	for k, txOut := range mtx.TxOut {
		// Make there is a script.
		if len(txOut.PkScript) == 0 {
			return stakeRuleError(ErrTSpendInvalidScriptLength,
				fmt.Sprintf("TSPEND out script length is 0: %v",
					len(txOut.PkScript)))
		}

		if k == 0 {
			// Check for OP_RETURN
			if txscript.GetScriptClass(txOut.Version, txOut.PkScript) !=
				txscript.NullDataTy {
				return stakeRuleError(ErrTSpendInvalidTransaction,
					"First TSPEND output should have been "+
						"an OP_RETURN data push, but "+
						"was not")
			}
			continue
		}

		// All tx outs are tagged with OP_TGEN
		if txOut.PkScript[0] != txscript.OP_TGEN {
			return stakeRuleError(ErrTSpendInvalidTGen,
				fmt.Sprintf("Output is not tagged with "+
					"OP_TGEN: %v", k))
		}
		sc := txscript.GetScriptClass(txOut.Version, txOut.PkScript[1:])
		if !(sc == txscript.ScriptHashTy || sc == txscript.PubKeyHashTy) {
			return stakeRuleError(ErrTSpendInvalidP2SH,
				fmt.Sprintf("Output is not P2PH: %v", k))
		}
	}

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
		return stakeRuleError(ErrTreasuryBaseInvalid,
			"invalid treasurybase out script count")
	}

	// Verify all TxOut script versions.
	for k := range mtx.TxOut {
		if mtx.TxOut[k].Version != consensusVersion {
			return stakeRuleError(ErrTreasuryBaseInvalid,
				"invalid script version found in treasurybase")
		}
	}

	// First output must be a TADD
	if len(mtx.TxOut[0].PkScript) != 1 ||
		mtx.TxOut[0].PkScript[0] != txscript.OP_TADD {
		return stakeRuleError(ErrTreasuryBaseInvalid,
			"first treasurybase output must be a TADD")
	}

	// Required OP_RETURN
	if mtx.TxOut[1].PkScript[0] != txscript.OP_RETURN {
		return stakeRuleError(ErrTreasuryBaseInvalid,
			"second output must be an OP_RETURN script")
	}

	// Look for coinbase 12 byte extra nonce.
	// XXX validate extra nonce.
	tokenizer := txscript.MakeScriptTokenizer(mtx.TxOut[1].Version,
		mtx.TxOut[1].PkScript[1:])
	if tokenizer.Next() && tokenizer.Done() &&
		tokenizer.Opcode() != txscript.OP_DATA_12 &&
		len(tokenizer.Data()) != 12 {
		return stakeRuleError(ErrTreasuryBaseInvalid,
			"second output must be an OP_RETURN script followed "+
				"by 12 bytes")
	}

	// Make sure chainhash etc is treasurybase.
	// A treasury base must only have one transaction input.
	if len(mtx.TxIn) != 1 {
		return stakeRuleError(ErrTreasuryBaseInvalid,
			"invalid treasurybase in script count")
	}

	// The previous output of a coin base must have a max value index and a
	// zero hash.
	prevOut := &mtx.TxIn[0].PreviousOutPoint
	if prevOut.Index != math.MaxUint32 ||
		!bytes.Equal(prevOut.Hash[:], zeroHash[:]) {
		return stakeRuleError(ErrTreasuryBaseInvalid,
			"invalid treasurybase constants")
	}

	return nil
}

// CheckTreasuryBase verifies that the provided MsgTx is a treasury base.
// XXX this is an exported function for the time being. We probably do not want
// to do that for release.
func CheckTreasuryBase(mtx *wire.MsgTx) error {
	return checkTreasuryBase(mtx)
}

// IsTreasuryBase returns true if the provided transaction is a treasury base
// transaction.
func IsTreasuryBase(tx *wire.MsgTx) bool {
	return checkTreasuryBase(tx) == nil
}
