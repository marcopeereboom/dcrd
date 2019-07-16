// Copyright (c) 2014 The btcsuite developers
// Copyright (c) 2015-2018 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockchain

import (
	"testing"
)

// TestErrorCodeStringer tests the stringized output for the ErrorCode type.
func TestErrorCodeStringer(t *testing.T) {
	tests := []struct {
		in   ErrorCode
		want string
	}{
		{ErrDuplicateBlock, "ErrDuplicateBlock"},
		{ErrMissingParent, "ErrMissingParent"},
		{ErrBlockTooBig, "ErrBlockTooBig"},
		{ErrWrongBlockSize, "ErrWrongBlockSize"},
		{ErrBlockVersionTooOld, "ErrBlockVersionTooOld"},
		{ErrBadStakeVersion, "ErrBadStakeVersion"},
		{ErrInvalidTime, "ErrInvalidTime"},
		{ErrTimeTooOld, "ErrTimeTooOld"},
		{ErrTimeTooNew, "ErrTimeTooNew"},
		{ErrDifficultyTooLow, "ErrDifficultyTooLow"},
		{ErrUnexpectedDifficulty, "ErrUnexpectedDifficulty"},
		{ErrHighHash, "ErrHighHash"},
		{ErrBadMerkleRoot, "ErrBadMerkleRoot"},
		{ErrBadCommitmentRoot, "ErrBadCommitmentRoot"},
		{ErrBadCheckpoint, "ErrBadCheckpoint"},
		{ErrForkTooOld, "ErrForkTooOld"},
		{ErrCheckpointTimeTooOld, "ErrCheckpointTimeTooOld"},
		{ErrNoTransactions, "ErrNoTransactions"},
		{ErrTooManyTransactions, "ErrTooManyTransactions"},
		{ErrNoTxInputs, "ErrNoTxInputs"},
		{ErrNoTxOutputs, "ErrNoTxOutputs"},
		{ErrTxTooBig, "ErrTxTooBig"},
		{ErrBadTxOutValue, "ErrBadTxOutValue"},
		{ErrDuplicateTxInputs, "ErrDuplicateTxInputs"},
		{ErrBadTxInput, "ErrBadTxInput"},
		{ErrMissingTxOut, "ErrMissingTxOut"},
		{ErrUnfinalizedTx, "ErrUnfinalizedTx"},
		{ErrDuplicateTx, "ErrDuplicateTx"},
		{ErrOverwriteTx, "ErrOverwriteTx"},
		{ErrImmatureSpend, "ErrImmatureSpend"},
		{ErrSpendTooHigh, "ErrSpendTooHigh"},
		{ErrBadFees, "ErrBadFees"},
		{ErrTooManySigOps, "ErrTooManySigOps"},
		{ErrFirstTxNotCoinbase, "ErrFirstTxNotCoinbase"},
		{ErrFirstTxNotOpReturn, "ErrFirstTxNotOpReturn"},
		{ErrCoinbaseHeight, "ErrCoinbaseHeight"},
		{ErrMultipleCoinbases, "ErrMultipleCoinbases"},
		{ErrStakeTxInRegularTree, "ErrStakeTxInRegularTree"},
		{ErrRegTxInStakeTree, "ErrRegTxInStakeTree"},
		{ErrBadCoinbaseScriptLen, "ErrBadCoinbaseScriptLen"},
		{ErrBadCoinbaseValue, "ErrBadCoinbaseValue"},
		{ErrBadCoinbaseOutpoint, "ErrBadCoinbaseOutpoint"},
		{ErrBadCoinbaseFraudProof, "ErrBadCoinbaseFraudProof"},
		{ErrBadCoinbaseAmountIn, "ErrBadCoinbaseAmountIn"},
		{ErrBadStakebaseAmountIn, "ErrBadStakebaseAmountIn"},
		{ErrBadStakebaseScriptLen, "ErrBadStakebaseScriptLen"},
		{ErrBadStakebaseScrVal, "ErrBadStakebaseScrVal"},
		{ErrScriptMalformed, "ErrScriptMalformed"},
		{ErrScriptValidation, "ErrScriptValidation"},
		{ErrNotEnoughStake, "ErrNotEnoughStake"},
		{ErrStakeBelowMinimum, "ErrStakeBelowMinimum"},
		{ErrNonstandardStakeTx, "ErrNonstandardStakeTx"},
		{ErrNotEnoughVotes, "ErrNotEnoughVotes"},
		{ErrTooManyVotes, "ErrTooManyVotes"},
		{ErrFreshStakeMismatch, "ErrFreshStakeMismatch"},
		{ErrTooManySStxs, "ErrTooManySStxs"},
		{ErrInvalidEarlyStakeTx, "ErrInvalidEarlyStakeTx"},
		{ErrTicketUnavailable, "ErrTicketUnavailable"},
		{ErrVotesOnWrongBlock, "ErrVotesOnWrongBlock"},
		{ErrVotesMismatch, "ErrVotesMismatch"},
		{ErrIncongruentVotebit, "ErrIncongruentVotebit"},
		{ErrInvalidSSRtx, "ErrInvalidSSRtx"},
		{ErrRevocationsMismatch, "ErrRevocationsMismatch"},
		{ErrTooManyRevocations, "ErrTooManyRevocations"},
		{ErrTicketCommitment, "ErrTicketCommitment"},
		{ErrInvalidVoteInput, "ErrInvalidVoteInput"},
		{ErrBadNumPayees, "ErrBadNumPayees"},
		{ErrBadPayeeScriptVersion, "ErrBadPayeeScriptVersion"},
		{ErrBadPayeeScriptType, "ErrBadPayeeScriptType"},
		{ErrMismatchedPayeeHash, "ErrMismatchedPayeeHash"},
		{ErrBadPayeeValue, "ErrBadPayeeValue"},
		{ErrSSGenSubsidy, "ErrSSGenSubsidy"},
		{ErrImmatureTicketSpend, "ErrImmatureTicketSpend"},
		{ErrTicketInputScript, "ErrTicketInputScript"},
		{ErrInvalidRevokeInput, "ErrInvalidRevokeInput"},
		{ErrSSRtxPayees, "ErrSSRtxPayees"},
		{ErrTxSStxOutSpend, "ErrTxSStxOutSpend"},
		{ErrRegTxCreateStakeOut, "ErrRegTxCreateStakeOut"},
		{ErrInvalidFinalState, "ErrInvalidFinalState"},
		{ErrPoolSize, "ErrPoolSize"},
		{ErrForceReorgWrongChain, "ErrForceReorgWrongChain"},
		{ErrForceReorgMissingChild, "ErrForceReorgMissingChild"},
		{ErrBadStakebaseValue, "ErrBadStakebaseValue"},
		{ErrDiscordantTxTree, "ErrDiscordantTxTree"},
		{ErrStakeFees, "ErrStakeFees"},
		{ErrNoStakeTx, "ErrNoStakeTx"},
		{ErrBadBlockHeight, "ErrBadBlockHeight"},
		{ErrBlockOneTx, "ErrBlockOneTx"},
		{ErrBlockOneInputs, "ErrBlockOneInputs"},
		{ErrBlockOneOutputs, "ErrBlockOneOutputs"},
		{ErrNoTax, "ErrNoTax"},
		{ErrExpiredTx, "ErrExpiredTx"},
		{ErrExpiryTxSpentEarly, "ErrExpiryTxSpentEarly"},
		{ErrFraudAmountIn, "ErrFraudAmountIn"},
		{ErrFraudBlockHeight, "ErrFraudBlockHeight"},
		{ErrFraudBlockIndex, "ErrFraudBlockIndex"},
		{ErrZeroValueOutputSpend, "ErrZeroValueOutputSpend"},
		{ErrInvalidEarlyVoteBits, "ErrInvalidEarlyVoteBits"},
		{ErrInvalidEarlyFinalState, "ErrInvalidEarlyFinalState"},
		{ErrKnownInvalidBlock, "ErrKnownInvalidBlock"},
		{ErrInvalidAncestorBlock, "ErrInvalidAncestorBlock"},
		{ErrInvalidTemplateParent, "ErrInvalidTemplateParent"},
		{ErrMultipleTreasuryBases, "ErrMultipleTreasuryBases"},
		{ErrUnknownPiKey, "ErrUnknownPiKey"},
		{ErrInvalidPiSignature, "ErrInvalidPiSignature"},
		{ErrNotTVI, "ErrNotTVI"},
		{ErrInvalidTSpendWindow, "ErrInvalidTSpendWindow"},
		{ErrNotEnoughTSpendVotes, "ErrNotEnoughTSpendVotes"},
		{ErrTSpendExists, "ErrTSpendExists"},
		{ErrInvalidExpenditure, "ErrInvalidExpenditure"},
		{0xffff, "Unknown ErrorCode (65535)"},
	}

	// Detect additional error codes that don't have the stringer added.
	if len(tests)-1 != int(numErrorCodes) {
		t.Errorf("It appears an error code was added without adding an " +
			"associated stringer test")
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		result := test.in.String()
		if result != test.want {
			t.Errorf("String #%d\n got: %s want: %s", i, result,
				test.want)
			continue
		}
	}
}

// TestRuleError tests the error output for the RuleError type.
func TestRuleError(t *testing.T) {
	tests := []struct {
		in   RuleError
		want string
	}{
		{
			RuleError{Description: "duplicate block"},
			"duplicate block",
		},
		{
			RuleError{Description: "human-readable error"},
			"human-readable error",
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		result := test.in.Error()
		if result != test.want {
			t.Errorf("Error #%d\n got: %s want: %s", i, result,
				test.want)
			continue
		}
	}
}

// TestIsErrorCode ensures IsErrorCode works as intended.
func TestIsErrorCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code ErrorCode
		want bool
	}{{
		name: "ErrUnexpectedDifficulty testing for ErrUnexpectedDifficulty",
		err:  ruleError(ErrUnexpectedDifficulty, ""),
		code: ErrUnexpectedDifficulty,
		want: true,
	}, {
		name: "ErrHighHash testing for ErrHighHash",
		err:  ruleError(ErrHighHash, ""),
		code: ErrHighHash,
		want: true,
	}, {
		name: "ErrHighHash error testing for ErrUnexpectedDifficulty",
		err:  ruleError(ErrHighHash, ""),
		code: ErrUnexpectedDifficulty,
		want: false,
	}, {
		name: "ErrHighHash error testing for unknown error code",
		err:  ruleError(ErrHighHash, ""),
		code: 0xffff,
		want: false,
	}, {
		name: "nil error testing for ErrUnexpectedDifficulty",
		err:  nil,
		code: ErrUnexpectedDifficulty,
		want: false,
	}}
	for _, test := range tests {
		result := IsErrorCode(test.err, test.code)
		if result != test.want {
			t.Errorf("%s: unexpected result -- got: %v want: %v", test.name,
				result, test.want)
			continue
		}
	}
}
