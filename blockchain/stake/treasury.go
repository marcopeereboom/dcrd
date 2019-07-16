// Copyright (c) 2019 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package stake

import (
	"bytes"
	"fmt"
	"math"

	"github.com/decred/dcrd/txscript/v3"
	"github.com/decred/dcrd/wire"
)

// This file contains the functions that verify that treasury transactions
// strictly adhere to the specified format.
//
// == User sends to treasury ==
// TxIn:  Normal TxIn signature scripts
// TxOut[0] OP_TADD
// TxOut[1] optional OP_SSTXCHANGE
//
// == Treasurybase add ==
// TxIn[0]:  Stakebase
// TxOut[0] OP_TADD
// TxOut[1] OP_RETURN <random>
//
// == Spend from treasury ==
// TxIn[0]     <signature> <pi pubkey> OP_TSPEND
// TxOut[0]    OP_RETURN <random>
// TxOut[1..N] OP_TGEN <paytopubkeyhash || paytoscripthash>

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

	// only 1 stake change output allowed.
	if len(mtx.TxOut) == 2 {
		// Script length has been already verified.
		if !txscript.IsStakeChangeScript(mtx.TxOut[1].Version,
			mtx.TxOut[1].PkScript) {
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

// CheckTSpend verifies if a MsgTx is a valid TSPEND and returns the DER
// encodes signature without the sighash byte and the compressed public key if
// successful. This function DOES NOT check the signature or if the public key
// is a well known PI key. This is a convenience function to obtain the
// signature and public key without iterating over the same MsgTx over and over
// again.
func CheckTSpend(mtx *wire.MsgTx) ([]byte, []byte, error) {
	// A valid TSPEND consists of a single TxIn that contains a signature,
	// a public key and an OP_TSPEND opcode.
	//
	// There must be at least two outputs. The first must contain an
	// OP_RETURN followed by a 32 byte data push of a random number. This
	// is used to randomize the transaction hash.
	// The second output must be a TGEN tagged P2SH or P2PH script.
	if len(mtx.TxIn) != 1 || len(mtx.TxOut) < 2 {
		return nil, nil, stakeRuleError(ErrTSpendInvalidLength,
			fmt.Sprintf("invalid TSPEND script lengths in: %v "+
				"out: %v", len(mtx.TxIn), len(mtx.TxOut)))
	}

	// Check to make sure that all output scripts are the consensus version.
	for k, txOut := range mtx.TxOut {
		if txOut.Version != consensusVersion {
			return nil, nil, stakeRuleError(ErrTSpendInvalidVersion,
				fmt.Sprintf("invalid script version found in "+
					"TxOut: %v", k))
		}

		// Make there is a script.
		if len(txOut.PkScript) == 0 {
			return nil, nil, stakeRuleError(ErrTSpendInvalidScriptLength,
				fmt.Sprintf("invalid TxOut script length %v: "+
					"%v", k, len(txOut.PkScript)))
		}
	}

	// Pull out signature, pubkey and OP_TSPEND.
	var (
		signature, pubKey []byte
	)
	tokenizer := txscript.MakeScriptTokenizer(0,
		mtx.TxIn[0].SignatureScript)
	for i := 0; i <= 3; i++ {
		// Expect a token
		if !tokenizer.Next() {
			if i == 3 {
				// State machine complete.
				break
			}
			return nil, nil, stakeRuleError(ErrTSpendInvalidTokenCount,
				fmt.Sprintf("TSPEND token count: %v", i))
		}

		opcode := tokenizer.Opcode()
		data := tokenizer.Data()
		switch i {
		case 0:
			// Check signature encoding
			var err1, err2 error
			if len(data) > 0 {
				hashType := txscript.SigHashType(data[len(data)-1])
				sigBytes := data[:len(data)-1]
				err1 = txscript.CheckHashTypeEncoding(hashType)
				err2 = txscript.CheckSignatureEncoding(sigBytes)
				if err1 == nil && err2 == nil {
					signature = sigBytes
					continue
				}
			}
			return nil, nil, stakeRuleError(ErrTSpendInvalidSignature,
				fmt.Sprintf("TSPEND invalid signature: "+
					"%v %v (%v) (%v)",
					i, len(data), err1, err2))
		case 1:
			// Check public key encoding
			if txscript.IsStrictCompressedPubKeyEncoding(data) {
				pubKey = data
				continue
			}
			return nil, nil, stakeRuleError(ErrTSpendInvalidPubkey,
				fmt.Sprintf("TSPEND invalid pubkey %v", i))
		case 2:
			// Check for OP_TSPEND
			if opcode == txscript.OP_TSPEND {
				continue
			}
			return nil, nil, stakeRuleError(ErrTSpendInvalidOpcode,
				fmt.Sprintf("TSPEND invalid opcode: 0x%x",
					opcode))
		}
	}

	// Make sure TxOut[0] contains an OP_RETURN followed by a 32 byte data
	// push
	if !txscript.IsStrictNullData(mtx.TxOut[0].Version,
		mtx.TxOut[0].PkScript, 32) {
		return nil, nil, stakeRuleError(ErrTSpendInvalidTransaction,
			"First TSPEND output should have been an OP_RETURN "+
				"followed by a 32 byte data push")
	}

	// Verify that the TxOut's contains a P2PH or P2PH scripts.
	for k, txOut := range mtx.TxOut[1:] {
		// All tx outs are tagged with OP_TGEN
		if txOut.PkScript[0] != txscript.OP_TGEN {
			return nil, nil, stakeRuleError(ErrTSpendInvalidTGen,
				fmt.Sprintf("Output is not tagged with "+
					"OP_TGEN: %v", k))
		}
		if !(txscript.IsPubKeyHashScript(txOut.PkScript[1:]) ||
			txscript.IsPayToScriptHash(txOut.PkScript[1:])) {
			return nil, nil, stakeRuleError(ErrTSpendInvalidSpendScript,
				fmt.Sprintf("Output is not P2PH: %v", k))
		}
	}

	return signature, pubKey, nil
}

// checkTSpend verifies if a MsgTx is a valid TSPEND.
func checkTSpend(mtx *wire.MsgTx) error {
	_, _, err := CheckTSpend(mtx)
	return err
}

// IsTSpend returns true if the provided transaction is a proper TSPEND.
func IsTSpend(tx *wire.MsgTx) bool {
	return checkTSpend(tx) == nil
}

// checkTreasuryBase verifies that the provided MsgTx is a treasury base.
func checkTreasuryBase(mtx *wire.MsgTx) error {
	// A TADD consists of one OP_TADD in PkScript[0] followed by an
	// OP_RETURN <random> in  PkScript[1].
	if len(mtx.TxIn) != 1 || len(mtx.TxOut) != 2 {
		return stakeRuleError(ErrTreasuryBaseInvalidCount,
			fmt.Sprintf("invalid treasurybase out script count: "+
				"%v %v", len(mtx.TxIn), len(mtx.TxOut)))
	}

	// Verify all TxOut script versions.
	for k := range mtx.TxOut {
		if mtx.TxOut[k].Version != consensusVersion {
			return stakeRuleError(ErrTreasuryBaseInvalidVersion,
				fmt.Sprintf("invalid script version found in "+
					"treasurybase: %v", k))
		}
	}

	// First output must be a TADD
	if len(mtx.TxOut[0].PkScript) != 1 ||
		mtx.TxOut[0].PkScript[0] != txscript.OP_TADD {
		return stakeRuleError(ErrTreasuryBaseInvalidOpcode0,
			"first treasurybase output must be a TADD")
	}

	// Required OP_RETURN, OP_DATA_12 + 12 bytes = 14 bytes total.
	if len(mtx.TxOut[1].PkScript) != 14 ||
		mtx.TxOut[1].PkScript[0] != txscript.OP_RETURN ||
		mtx.TxOut[1].PkScript[1] != txscript.OP_DATA_12 {
		return stakeRuleError(ErrTreasuryBaseInvalidOpcode1,
			"second treasurybase output must be an OP_RETURN "+
				" OP_DATA_12 script")
	}
	// XXX validate extra nonce in OP_DATA_12 payload.

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
