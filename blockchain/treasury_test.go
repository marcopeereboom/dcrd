// Copyright (c) 2019 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockchain

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/decred/dcrd/blockchain/stake/v3"
	"github.com/decred/dcrd/blockchain/standalone/v2"
	"github.com/decred/dcrd/blockchain/v3/chaingen"
	"github.com/decred/dcrd/chaincfg/chainhash"
	"github.com/decred/dcrd/chaincfg/v3"
	"github.com/decred/dcrd/database/v2"
	"github.com/decred/dcrd/dcrutil/v3"
	"github.com/decred/dcrd/txscript/v3"
	"github.com/decred/dcrd/wire"
	"github.com/decred/slog"
)

const (
	defaultValues = "64000000000000000500000000000000010000000000000002000000000000000300000000000000fdfffffffffffffffeffffffffffffff"

	defaultEmptyValues = "640000000000000000010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"

	tooManyValues = "6400000000000000010100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
)

var (
	values  []byte
	empty   []byte
	many    []byte
	privKey []byte
)

func init() {
	var err error
	values, err = hex.DecodeString(defaultValues)
	if err != nil {
		panic(err)
	}
	empty, err = hex.DecodeString(defaultEmptyValues)
	if err != nil {
		panic(err)
	}
	many, err = hex.DecodeString(tooManyValues)
	if err != nil {
		panic(err)
	}

	// This key comes from chaincfg/regnetparams.go
	privKey, err = hex.DecodeString("68ab7efdac0eb99b1edf83b23374cc7a9c8d0a4183a2627afc8ea0437b20589e")
	if err != nil {
		panic(err)
	}
}

func TestSerializeTreasuryState(t *testing.T) {
	tests := []struct {
		name        string
		state       TreasuryState
		expectError bool
		expected    []byte
	}{
		{
			name: "equal",
			state: TreasuryState{
				Balance: 100,
				Values:  []int64{1, 2, 3, -3, -2},
			},
			expected: values,
		},
		{
			name: "just enough",
			state: TreasuryState{
				Balance: 100,
				Values:  make([]int64, TreasuryMaxEntriesPerBlock),
			},
			expected: empty,
		},
		{
			name: "negative",
			state: TreasuryState{
				Balance: -100,
				Values:  []int64{1, 2, 3, -3, -2},
			},
			expectError: true,
		},
		{
			name: "too many",
			state: TreasuryState{
				Balance: 100,
				Values:  make([]int64, TreasuryMaxEntriesPerBlock+1),
			},
			expectError: true,
		},
	}

	for i, test := range tests {
		b, err := serializeTreasuryState(test.state)
		t.Logf("%v %v %v", i, test.name, err)
		if test.expectError {
			if err == nil {
				t.Fatalf("%v %v (serialized): expected an error",
					i, test.name)
			}
			continue
		} else {
			if err != nil {
				t.Fatalf("%v %v (serialized) unexpected error: %v",
					i, test.name, err)
			}
		}
		if !bytes.Equal(test.expected, b) {
			t.Fatalf("%v %v (serialized): got %x expected %x",
				i, test.name, b, test.expected)
		}
		tso, err := deserializeTreasuryState(b)
		if err != nil {
			t.Fatalf("%v %v (deserialized): %v",
				i, test.name, err)
		}
		if !reflect.DeepEqual(test.state, *tso) {
			t.Fatalf("%v %v (equal): got %v expected %v",
				i, test.name, *tso, test.state)
		}
	}
}

func TestDeserializeTreasuryState(t *testing.T) {
	tests := []struct {
		name        string
		state       []byte
		expectError bool
		expected    TreasuryState
	}{
		{
			name:  "equal",
			state: values,
			expected: TreasuryState{
				Balance: 100,
				Values:  []int64{1, 2, 3, -3, -2},
			},
		},
		{
			name:        "empty",
			state:       nil,
			expectError: true,
		},
		{
			name:        "short",
			state:       values[0 : len(values)/2],
			expectError: true,
		},
		{
			name:        "one byte short",
			state:       values[0 : len(values)-1],
			expectError: true,
		},
		{
			name:        "too many",
			state:       many,
			expectError: true,
		},
	}

	for i, test := range tests {
		tso, err := deserializeTreasuryState(test.state)
		t.Logf("%v %v %v", i, test.name, err)
		if test.expectError {
			if err == nil {
				t.Fatalf("%v %v (deserialized): expected an error",
					i, test.name)
			}
			continue
		} else {
			if err != nil {
				t.Fatalf("%v %v (deserialized) unexpected error: %v",
					i, test.name, err)
			}
		}
		b, err := serializeTreasuryState(test.expected)
		if err != nil {
			t.Fatalf("%v %v (serialized): %v", i, test.name, err)
		}
		if !bytes.Equal(test.state, b) {
			t.Fatalf("%v %v (serialized): got %x expected %x",
				i, test.name, b, test.expected)
		}
		if !reflect.DeepEqual(*tso, test.expected) {
			t.Fatalf("%v %v (equal): got %v expected %v",
				i, test.name, *tso, test.expected)
		}
	}
}

// TestTreasuryDatabase tests treasury database functionality.
func TestTreasuryDatabase(t *testing.T) {
	// Create a new database to store treasury state.
	dbName := "ffldb_treasurydb_test"
	dbPath, err := ioutil.TempDir("", dbName)
	if err != nil {
		t.Fatalf("unable to create treasury db path: %v", err)
	}
	defer os.RemoveAll(dbPath)
	net := chaincfg.RegNetParams().Net
	testDb, err := database.Create(testDbType, dbPath, net)
	if err != nil {
		t.Fatalf("error creating treasury db: %v", err)
	}
	defer testDb.Close()

	// Initialize the database, then try to read the version.
	err = addTreasuryBucket(testDb)
	if err != nil {
		t.Fatalf("%v", err.Error())
	}

	// Write maxTreasuryState records out.
	maxTreasuryState := uint64(1024)
	for i := uint64(0); i < maxTreasuryState; i++ {
		// Create synthetic treasury state
		ts := TreasuryState{
			Balance: int64(i),
			Values:  []int64{int64(i), -int64(i)},
		}

		// Create hash of counter.
		b := make([]byte, 16)
		binary.LittleEndian.PutUint64(b[0:], i)
		hash := chainhash.HashH(b)

		err = testDb.Update(func(dbTx database.Tx) error {
			return dbPutTreasuryBalanceWriter(dbTx, hash, ts)
		})
		if err != nil {
			t.Fatalf("%v", err.Error())
		}
	}

	// Pull records back out.
	for i := uint64(0); i < maxTreasuryState; i++ {
		// Create synthetic treasury state
		ts := TreasuryState{
			Balance: int64(i),
			Values:  []int64{int64(i), -int64(i)},
		}

		// Create hash of counter.
		b := make([]byte, 16)
		binary.LittleEndian.PutUint64(b[0:], i)
		hash := chainhash.HashH(b)

		var tsr *TreasuryState
		err = testDb.View(func(dbTx database.Tx) error {
			tsr, err = dbFetchTreasuryBalance(dbTx, hash)
			return err
		})
		if err != nil {
			t.Fatalf("%v", err.Error())
		}

		if !reflect.DeepEqual(ts, *tsr) {
			t.Fatalf("not same treasury state got %v wanted %v",
				ts, *tsr)
		}
	}
}

// TestTspendDatabase tests tspend database functionality including
// serialization and deserialization.
func TestTSpendDatabase(t *testing.T) {
	// Create a new database to store treasury state.
	dbName := "ffldb_tspenddb_test"
	dbPath, err := ioutil.TempDir("", dbName)
	if err != nil {
		t.Fatalf("unable to create tspend db path: %v", err)
	}
	defer os.RemoveAll(dbPath)
	net := chaincfg.RegNetParams().Net
	testDb, err := database.Create(testDbType, dbPath, net)
	if err != nil {
		t.Fatalf("error creating tspend db: %v", err)
	}
	defer testDb.Close()

	// Initialize the database, then try to read the version.
	err = addTSpendBucket(testDb)
	if err != nil {
		t.Fatalf("%v", err.Error())
	}

	// Write maxTSpendState records out.
	maxTSpendState := uint64(8)
	txHash := chainhash.Hash{}
	for i := uint64(0); i < maxTSpendState; i++ {
		// Create hash of counter.
		b := make([]byte, 16)
		binary.LittleEndian.PutUint64(b[0:], i)
		blockHash := chainhash.HashH(b)

		err = testDb.Update(func(dbTx database.Tx) error {
			return dbUpdateTSpend(dbTx, txHash, blockHash)
		})
		if err != nil {
			t.Fatalf("%v", err.Error())
		}
	}

	// Pull records back out.
	var hashes []chainhash.Hash
	err = testDb.View(func(dbTx database.Tx) error {
		hashes, err = dbFetchTSpend(dbTx, txHash)
		return err
	})
	if err != nil {
		t.Fatalf("%v", err.Error())
	}

	for i := uint64(0); i < maxTSpendState; i++ {
		// Create hash of counter.
		b := make([]byte, 16)
		binary.LittleEndian.PutUint64(b[0:], i)
		hash := chainhash.HashH(b)
		if !hash.IsEqual(&hashes[i]) {
			t.Fatalf("not same tspend hash got %v wanted %v",
				hashes[i], hash)
		}
	}
}

// appendHashes takes a slice of chainhash and votebits and appends it all
// together for a TV script.
func appendHashes(tspendHashes []*chainhash.Hash, votes []stake.TreasuryVoteT) []byte {
	if len(tspendHashes) != len(votes) {
		panic(fmt.Sprintf("assert appendHashes %v != %v",
			len(tspendHashes), len(votes)))
	}
	blob := make([]byte, 0, 2+chainhash.HashSize*7+7)
	blob = append(blob, 'T', 'V')
	for k, v := range tspendHashes {
		blob = append(blob, v[:]...)
		blob = append(blob, byte(votes[k]))
	}
	return blob
}

// addTSpendVotes reurns a munge function that votes according to voteBits.
func addTSpendVotes(t *testing.T, tspendHashes []*chainhash.Hash, votes []stake.TreasuryVoteT, nrVotes uint16, skipAssert bool) func(*wire.MsgBlock) {
	if len(tspendHashes) != len(votes) {
		panic(fmt.Sprintf("assert addTSpendVotes %v != %v",
			len(tspendHashes), len(votes)))
	}
	return func(b *wire.MsgBlock) {
		// Find SSGEN and append Yes vote.
		for k, v := range b.STransactions {
			if !stake.IsSSGen(v, true) { // Yes treasury
				continue
			}
			if len(v.TxOut) != 3 {
				t.Fatalf("expected SSGEN.TxOut len 3 got %v",
					len(v.TxOut))
			}

			// Only allow privided number of votes.
			if uint16(k) > nrVotes {
				break
			}

			// Append vote: OP_RET OP_DATA <TV> <tspend hash> <vote bits>
			vote := appendHashes(tspendHashes, votes)
			s, err := txscript.NewScriptBuilder().AddOp(txscript.OP_RETURN).
				AddData(vote).Script()
			if err != nil {
				t.Fatal(err)
			}
			b.STransactions[k].TxOut = append(b.STransactions[k].TxOut,
				&wire.TxOut{
					PkScript: s,
				})
			// Only TxVersionTreasury supports optional votes.
			b.STransactions[k].Version = wire.TxVersionTreasury

			// See if we shouild skip asserts. This is used for
			// munging votes and bits.
			if skipAssert {
				continue
			}

			// Assert vote insertion worked.
			_, err = stake.GetSSGenTreasuryVotes(s)
			if err != nil {
				t.Fatalf("expected treasury vote: %v", err)
			}

			// Assert this remains a valid SSGEN.
			err = stake.CheckSSGen(b.STransactions[k], true) // Yes treasury
			if err != nil {
				t.Fatalf("expected SSGen: %v", err)
			}
		}
	}
}

const devsub = 5000000000

// replaceCoinbase is a munge function that takes the coinbase and removes the
// treasury payout and moves it to a TADD treasury agenda based version. It
// also bumps all STransactions indexes by 1 since we require treasurybase to
// be the 0th entry in the stake tree.
func replaceCoinbase(b *wire.MsgBlock) {
	// XXX do we need to do something with fees here?

	// Find coinbase tx and remove dev subsidy.
	coinbaseTx := b.Transactions[0]
	devSubsidy := coinbaseTx.TxOut[0].Value
	coinbaseTx.TxOut = coinbaseTx.TxOut[1:]
	coinbaseTx.Version = wire.TxVersionTreasury
	coinbaseTx.TxIn[0].ValueIn -= devSubsidy

	// Assert devsub value
	//if devSubsidy != devsub {
	//	panic(fmt.Sprintf("dev subsidy mismatch: got %v, expected %v",
	//		devSubsidy, devsub))
	//}

	// Create treasuryBase and insert it at position 0 of the stake
	// tree.
	oldSTransactions := b.STransactions
	b.STransactions = make([]*wire.MsgTx, len(b.STransactions)+1)
	for k, v := range oldSTransactions {
		b.STransactions[k+1] = v
	}
	treasurybaseTx := wire.NewMsgTx()
	treasurybaseTx.Version = wire.TxVersionTreasury
	treasurybaseTx.AddTxIn(&wire.TxIn{
		PreviousOutPoint: *wire.NewOutPoint(&chainhash.Hash{},
			wire.MaxPrevOutIndex, wire.TxTreeRegular),
		Sequence:        wire.MaxTxInSequenceNum,
		BlockHeight:     wire.NullBlockHeight,
		BlockIndex:      wire.NullBlockIndex,
		SignatureScript: coinbaseTx.TxIn[0].SignatureScript,
	})
	treasurybaseTx.TxIn[0].ValueIn = devSubsidy
	treasurybaseTx.AddTxOut(&wire.TxOut{
		Value:    devSubsidy,
		PkScript: []byte{txscript.OP_TADD},
	})
	// Extranonce.
	treasurybaseTx.AddTxOut(&wire.TxOut{
		Value:    0,
		PkScript: standardCoinbaseOpReturn(b.Header.Height),
	})
	retTx := dcrutil.NewTx(treasurybaseTx) // XXX why do I have to do this?
	retTx.SetTree(wire.TxTreeStake)        // XXX why do I have to do this?
	b.STransactions[0] = retTx.MsgTx()     // XXX why do I have to do this?
}

func TestTSpendVoteCount(t *testing.T) {
	// Use a set of test chain parameters which allow for quicker vote
	// activation as compared to various existing network params.
	params := quickVoteActivationParams()

	// Clone the parameters so they can be mutated, find the correct deployment
	// for the fix sequence locks agenda, and, finally, ensure it is always
	// available to vote by removing the time constraints to prevent test
	// failures when the real expiration time passes.
	const tVoteID = chaincfg.VoteIDTreasury
	params = cloneParams(params)
	tVersion, deployment, err := findDeployment(params, tVoteID)
	if err != nil {
		t.Fatal(err)
	}
	removeDeploymentTimeConstraints(deployment)

	// Dave off tvi and mul.
	tvi := params.TreasuryVoteInterval
	mul := params.TreasuryVoteIntervalMultiplier

	// Create a test harness initialized with the genesis block as the tip.
	g, teardownFunc := newChaingenHarness(t, params, "treasurytest")
	defer teardownFunc()

	// replaceTreasuryVersions is a munge function which modifies the
	// provided block by replacing the block, stake, and vote versions with the
	// fix sequence locks deployment version.
	replaceTreasuryVersions := func(b *wire.MsgBlock) {
		chaingen.ReplaceBlockVersion(int32(tVersion))(b)
		chaingen.ReplaceStakeVersion(tVersion)(b)
		chaingen.ReplaceVoteVersions(tVersion)(b)
	}

	// ---------------------------------------------------------------------
	// Generate and accept enough blocks with the appropriate vote bits set
	// to reach one block prior to the treasury agenda becoming active.
	// ---------------------------------------------------------------------

	g.AdvanceToStakeValidationHeight()
	g.AdvanceFromSVHToActiveAgenda(tVoteID)

	// Ensure treasury agenda is active.
	gotActive, err := g.chain.IsTreasuryAgendaActive()
	if err != nil {
		t.Fatalf("IsTreasuryAgendaActive: %v", err)
	}
	if !gotActive {
		t.Fatalf("IsTreasuryAgendaActive: expected enabled treasury")
	}

	startTip := g.TipName()

	// ---------------------------------------------------------------------
	// Create TSpend in "mempool"
	// ---------------------------------------------------------------------

	nextBlockHeight := g.Tip().Header.Height + 1
	tspendAmount := devsub
	tspendFee := 100
	expiry := standalone.CalculateTSpendExpiry(int64(nextBlockHeight), tvi,
		mul)
	start, err := standalone.CalculateTSpendWindowStart(expiry, tvi, mul)
	if err != nil {
		t.Fatal(err)
	}
	end, err := standalone.CalculateTSpendWindowEnd(expiry, tvi)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("nbh %v expiry %v start %v end %v",
		nextBlockHeight, expiry, start, end)

	tspend := g.CreateTreasuryTSpend(privKey, []chaingen.AddressAmountTuple{
		{
			Amount: dcrutil.Amount(tspendAmount - tspendFee),
		},
	},
		dcrutil.Amount(tspendFee), expiry)
	tspendHash := tspend.TxHash()
	t.Logf("tspend %v amount %v fee %v", tspendHash, tspendAmount-tspendFee,
		tspendFee)

	// ---------------------------------------------------------------------
	// Try to insert TSPEND while not on a TVI
	//
	//   ... -> bva19
	//                  \-> bnottvi0
	// ---------------------------------------------------------------------

	// Assert we are not on a TVI and generate block. This should fail.
	if standalone.IsTreasuryVoteInterval(uint64(nextBlockHeight), tvi) {
		t.Fatalf("expected !TVI %v", nextBlockHeight)
	}
	outs := g.OldestCoinbaseOuts()
	name := "bnottvi0"
	g.NextBlock(name, nil, outs[1:], replaceTreasuryVersions,
		replaceCoinbase,
		func(b *wire.MsgBlock) {
			// Add TSpend
			b.AddSTransaction(tspend)
		})
	g.RejectTipBlock(ErrNotTVI)

	// ---------------------------------------------------------------------
	// Generate enough blocks to get to TVI.
	//
	//   ... -> bva19 -> bpretvi0 -> bpretvi1
	//                  \-> bnottvi0
	// ---------------------------------------------------------------------

	// Generate votes up to TVI. This is legal however they should NOT be
	// counted in the totals since they are outside of the voting window.
	g.SetTip(startTip)
	voteCount := params.TicketsPerBlock
	for i := uint32(0); i < start-nextBlockHeight; i++ {
		name := fmt.Sprintf("bpretvi%v", i)
		g.NextBlock(name, nil, outs[1:], replaceTreasuryVersions,
			replaceCoinbase,
			addTSpendVotes(t, []*chainhash.Hash{&tspendHash},
				[]stake.TreasuryVoteT{stake.TreasuryVoteYes},
				voteCount, false))
		g.SaveTipCoinbaseOutsWithTreasury()
		g.AcceptTipBlock()
		outs = g.OldestCoinbaseOuts()
	}

	// ---------------------------------------------------------------------
	// Add TSpend on first block of window. This should fail with not
	// enough votes.
	//
	//   ... -> bpretvi1
	//         \-> btvinotenough0
	// ---------------------------------------------------------------------

	// Assert we are on a TVI and generate block. This should fail.
	startTip = g.TipName()
	if standalone.IsTreasuryVoteInterval(uint64(g.Tip().Header.Height),
		tvi) {
		t.Fatalf("expected !TVI %v", g.Tip().Header.Height)
	}
	name = "btvinotenough0"
	g.NextBlock(name, nil, outs[1:], replaceTreasuryVersions,
		replaceCoinbase,
		func(b *wire.MsgBlock) {
			// Add TSpend
			b.AddSTransaction(tspend)
		})
	g.RejectTipBlock(ErrNotEnoughTSpendVotes)

	// ---------------------------------------------------------------------
	// Generate 1 TVI of No votes and add TSpend,
	//
	//   ... -> bpretvi1 -> btvi0 -> ... -> btvi3
	// ---------------------------------------------------------------------
	g.SetTip(startTip)
	for i := uint64(0); i < tvi; i++ {
		name := fmt.Sprintf("btvi%v", i)
		g.NextBlock(name, nil, outs[1:], replaceTreasuryVersions,
			replaceCoinbase,
			addTSpendVotes(t, []*chainhash.Hash{&tspendHash},
				[]stake.TreasuryVoteT{stake.TreasuryVoteNo},
				voteCount, false))
		g.SaveTipCoinbaseOutsWithTreasury()
		g.AcceptTipBlock()
		outs = g.OldestCoinbaseOuts()
	}

	// Assert we are on a TVI and generate block. This should fail.
	startTip = g.TipName()
	if standalone.IsTreasuryVoteInterval(uint64(g.Tip().Header.Height),
		tvi) {
		t.Fatalf("expected !TVI %v", g.Tip().Header.Height)
	}
	name = "btvinotenough1"
	g.NextBlock(name, nil, outs[1:], replaceTreasuryVersions,
		replaceCoinbase,
		func(b *wire.MsgBlock) {
			// Add TSpend
			b.AddSTransaction(tspend)
		})
	g.RejectTipBlock(ErrNotEnoughTSpendVotes)

	// ---------------------------------------------------------------------
	// Generate one more TVI of no votes.
	//
	//   ... -> btvinotenough0 -> btvi4 -> ... -> btvi7
	// ---------------------------------------------------------------------

	g.SetTip(startTip)
	for i := uint64(0); i < tvi; i++ {
		name := fmt.Sprintf("btvi%v", tvi+i)
		g.NextBlock(name, nil, outs[1:], replaceTreasuryVersions,
			replaceCoinbase,
			addTSpendVotes(t, []*chainhash.Hash{&tspendHash},
				[]stake.TreasuryVoteT{stake.TreasuryVoteNo},
				voteCount, false))
		g.SaveTipCoinbaseOutsWithTreasury()
		g.AcceptTipBlock()
		outs = g.OldestCoinbaseOuts()
	}

	// Assert we are on a TVI and generate block. This should fail with No
	// vote (TSpend should not have been submitted).
	startTip = g.TipName()
	if standalone.IsTreasuryVoteInterval(uint64(g.Tip().Header.Height),
		tvi) {
		t.Fatalf("expected !TVI %v", g.Tip().Header.Height)
	}
	name = "btvienough0"
	g.NextBlock(name, nil, outs[1:], replaceTreasuryVersions,
		replaceCoinbase,
		func(b *wire.MsgBlock) {
			// Add TSpend
			b.AddSTransaction(tspend)
		})
	g.RejectTipBlock(ErrNotEnoughTSpendVotes)

	// Assert we have the correct number of votes and voting window.
	blk := dcrutil.NewBlock(&wire.MsgBlock{
		Header: wire.BlockHeader{
			Height: g.Tip().Header.Height,
		},
	})
	startBlock, endBlock, yesVotes, noVotes, err := g.chain.TSpendCountVotes(blk,
		g.chain.bestChain.Tip(), dcrutil.NewTx(tspend))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("startBlock %v endBlock %v yesVotes %v noVotes %v",
		startBlock, endBlock, yesVotes, noVotes)

	if start != startBlock {
		t.Fatalf("invalid start block got %v wanted %v", startBlock, start)
	}
	if end != endBlock {
		t.Fatalf("invalid end block got %v wanted %v", endBlock, end)
	}

	expectedYesVotes := 0 // We voted a bunch of times outside the window
	expectedNoVotes := tvi * mul * uint64(params.TicketsPerBlock)
	if expectedYesVotes != yesVotes {
		t.Fatalf("invalid yes votes got %v wanted %v",
			expectedYesVotes, yesVotes)
	}
	if expectedNoVotes != uint64(noVotes) {
		t.Fatalf("invalid no votes got %v wanted %v",
			expectedNoVotes, noVotes)
	}

	// ---------------------------------------------------------------------
	// Generate one more TVI and append expired TSpend.
	//
	//   ... -> bposttvi0 ... -> bposttvi3 ->
	//                                     \-> bexpired0
	// ---------------------------------------------------------------------

	g.SetTip(startTip)
	for i := uint64(0); i < tvi; i++ {
		name := fmt.Sprintf("bposttvi%v", i)
		g.NextBlock(name, nil, outs[1:], replaceTreasuryVersions,
			replaceCoinbase)
		g.SaveTipCoinbaseOutsWithTreasury()
		g.AcceptTipBlock()
		outs = g.OldestCoinbaseOuts()
	}

	// Assert TSpend expired
	startTip = g.TipName()
	if standalone.IsTreasuryVoteInterval(uint64(g.Tip().Header.Height),
		tvi) {
		t.Fatalf("expected !TVI %v", g.Tip().Header.Height)
	}
	name = "bexpired0"
	g.NextBlock(name, nil, outs[1:], replaceTreasuryVersions,
		replaceCoinbase,
		func(b *wire.MsgBlock) {
			// Add TSpend
			b.AddSTransaction(tspend)
		})
	g.RejectTipBlock(ErrExpiredTx)

	// ---------------------------------------------------------------------
	// Create TSpend in "mempool"
	//
	// Test corner of quorum-1 vote and exact quorum yes vote.
	// ---------------------------------------------------------------------

	// Use exact hight to validate that tspend starts on next tvi.
	expiry = standalone.CalculateTSpendExpiry(int64(g.Tip().Header.Height),
		tvi, mul)
	start, err = standalone.CalculateTSpendWindowStart(expiry, tvi, mul)
	if err != nil {
		t.Fatal(err)
	}
	end, err = standalone.CalculateTSpendWindowEnd(expiry, tvi)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("height %v expiry %v start %v end %v",
		g.Tip().Header.Height, expiry, start, end)
	// While here test that start is next tvi while on tvi.
	if g.Tip().Header.Height+uint32(tvi) != start {
		t.Fatalf("expected to see exactly next tvi got %v wanted %v",
			start, g.Tip().Header.Height+uint32(tvi))
	}

	tspend = g.CreateTreasuryTSpend(privKey, []chaingen.AddressAmountTuple{
		{
			Amount: dcrutil.Amount(tspendAmount - tspendFee),
		},
	},
		dcrutil.Amount(tspendFee), expiry)
	tspendHash = tspend.TxHash()
	t.Logf("tspend %v amount %v fee %v", tspendHash, tspendAmount-tspendFee,
		tspendFee)

	// Fast forward to next tvi and add no votes which should not count.
	g.SetTip(startTip)
	for i := uint64(0); i < tvi; i++ {
		name := fmt.Sprintf("bnovote%v", i)
		g.NextBlock(name, nil, outs[1:], replaceTreasuryVersions,
			replaceCoinbase,
			addTSpendVotes(t, []*chainhash.Hash{&tspendHash},
				[]stake.TreasuryVoteT{stake.TreasuryVoteNo},
				voteCount, false))
		g.SaveTipCoinbaseOutsWithTreasury()
		g.AcceptTipBlock()
		outs = g.OldestCoinbaseOuts()
	}

	// Hit quorum-1 yes votes.
	maxVotes := uint32(params.TicketsPerBlock) *
		(endBlock - startBlock)
	quorum := uint64(maxVotes) * params.TreasuryVoteQuorumMultiplier /
		params.TreasuryVoteQuorumDivisor
	totalVotes := uint16(quorum - 1)
	for i := uint64(0); i < tvi; i++ {
		t.Logf("totalVotes %v", totalVotes)
		name := fmt.Sprintf("byesvote%v", i)
		g.NextBlock(name, nil, outs[1:], replaceTreasuryVersions,
			replaceCoinbase,
			addTSpendVotes(t, []*chainhash.Hash{&tspendHash},
				[]stake.TreasuryVoteT{stake.TreasuryVoteYes},
				totalVotes, false))
		g.SaveTipCoinbaseOutsWithTreasury()
		g.AcceptTipBlock()
		outs = g.OldestCoinbaseOuts()

		if totalVotes > params.TicketsPerBlock {
			totalVotes -= params.TicketsPerBlock
		} else {
			totalVotes = 0
		}
	}

	// Verify we are one vote shy of quorum
	startTip = g.TipName()
	if standalone.IsTreasuryVoteInterval(uint64(g.Tip().Header.Height),
		tvi) {
		t.Fatalf("expected !TVI %v", g.Tip().Header.Height)
	}
	name = "bquorum0"
	g.NextBlock(name, nil, outs[1:], replaceTreasuryVersions,
		replaceCoinbase,
		func(b *wire.MsgBlock) {
			// Add TSpend
			b.AddSTransaction(tspend)
		})
	g.RejectTipBlock(ErrNotEnoughTSpendVotes)

	// Count votes.
	blk = dcrutil.NewBlock(&wire.MsgBlock{
		Header: wire.BlockHeader{
			Height: g.Tip().Header.Height,
		},
	})
	startBlock, endBlock, yesVotes, noVotes, err = g.chain.TSpendCountVotes(blk,
		g.chain.bestChain.Tip(), dcrutil.NewTx(tspend))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("startBlock %v endBlock %v yesVotes %v noVotes %v",
		startBlock, endBlock, yesVotes, noVotes)
	if int(quorum-1) != yesVotes {
		t.Fatalf("unexpected yesVote count got %v wanted %v",
			yesVotes, quorum-1)
	}

	// Hit exact yes vote quorum
	g.SetTip(startTip)
	totalVotes = uint16(1)
	for i := uint64(0); i < tvi; i++ {
		t.Logf("totalVotes %v", totalVotes)
		name := fmt.Sprintf("byesvote%v", tvi+i)
		g.NextBlock(name, nil, outs[1:], replaceTreasuryVersions,
			replaceCoinbase,
			addTSpendVotes(t, []*chainhash.Hash{&tspendHash},
				[]stake.TreasuryVoteT{stake.TreasuryVoteYes},
				totalVotes, false))
		g.SaveTipCoinbaseOutsWithTreasury()
		g.AcceptTipBlock()
		outs = g.OldestCoinbaseOuts()

		if totalVotes > params.TicketsPerBlock {
			totalVotes -= params.TicketsPerBlock
		} else {
			totalVotes = 0
		}
	}

	// Count votes.
	blk = dcrutil.NewBlock(&wire.MsgBlock{
		Header: wire.BlockHeader{
			Height: g.Tip().Header.Height,
		},
	})
	startBlock, endBlock, yesVotes, noVotes, err = g.chain.TSpendCountVotes(blk,
		g.chain.bestChain.Tip(), dcrutil.NewTx(tspend))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("startBlock %v endBlock %v yesVotes %v noVotes %v",
		startBlock, endBlock, yesVotes, noVotes)
	if int(quorum) != yesVotes {
		t.Fatalf("unexpected yesVote count got %v wanted %v",
			yesVotes, quorum)
	}

	// Verify TSpend can be added exactly on quorum.
	if standalone.IsTreasuryVoteInterval(uint64(g.Tip().Header.Height),
		tvi) {
		t.Fatalf("expected !TVI %v", g.Tip().Header.Height)
	}
	name = "bquorum1"
	g.NextBlock(name, nil, outs[1:], replaceTreasuryVersions,
		replaceCoinbase,
		func(b *wire.MsgBlock) {
			// Add TSpend
			b.AddSTransaction(tspend)
		})
	g.AcceptTipBlock()
}

func TestTSpendExpenditures(t *testing.T) {
	// Use a set of test chain parameters which allow for quicker vote
	// activation as compared to various existing network params.
	params := quickVoteActivationParams()

	// Clone the parameters so they can be mutated, find the correct deployment
	// for the fix sequence locks agenda, and, finally, ensure it is always
	// available to vote by removing the time constraints to prevent test
	// failures when the real expiration time passes.
	const tVoteID = chaincfg.VoteIDTreasury
	params = cloneParams(params)
	tVersion, deployment, err := findDeployment(params, tVoteID)
	if err != nil {
		t.Fatal(err)
	}
	removeDeploymentTimeConstraints(deployment)

	// Dave off tvi and mul.
	tvi := params.TreasuryVoteInterval
	mul := params.TreasuryVoteIntervalMultiplier

	// Create a test harness initialized with the genesis block as the tip.
	g, teardownFunc := newChaingenHarness(t, params, "treasurytest")
	defer teardownFunc()

	// replaceTreasuryVersions is a munge function which modifies the
	// provided block by replacing the block, stake, and vote versions with the
	// fix sequence locks deployment version.
	replaceTreasuryVersions := func(b *wire.MsgBlock) {
		chaingen.ReplaceBlockVersion(int32(tVersion))(b)
		chaingen.ReplaceStakeVersion(tVersion)(b)
		chaingen.ReplaceVoteVersions(tVersion)(b)
	}

	// ---------------------------------------------------------------------
	// Generate and accept enough blocks with the appropriate vote bits set
	// to reach one block prior to the treasury agenda becoming active.
	// ---------------------------------------------------------------------

	g.AdvanceToStakeValidationHeight()
	g.AdvanceFromSVHToActiveAgenda(tVoteID)

	// Ensure treasury agenda is active.
	gotActive, err := g.chain.IsTreasuryAgendaActive()
	if err != nil {
		t.Fatalf("IsTreasuryAgendaActive: %v", err)
	}
	if !gotActive {
		t.Fatalf("IsTreasuryAgendaActive: expected enabled treasury")
	}

	// ---------------------------------------------------------------------
	// Create TSPEND in mempool for exact amount of treasury + 1 atom
	// ---------------------------------------------------------------------
	nextBlockHeight := g.Tip().Header.Height + 1
	expiry := standalone.CalculateTSpendExpiry(int64(nextBlockHeight), tvi,
		mul)
	start, err := standalone.CalculateTSpendWindowStart(expiry, tvi, mul)
	if err != nil {
		t.Fatal(err)
	}
	end, err := standalone.CalculateTSpendWindowEnd(expiry, tvi)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("nbh %v expiry %v start %v end %v",
		nextBlockHeight, expiry, start, end)

	tspendAmount := devsub*(tvi*mul-uint64(params.CoinbaseMaturity)+
		uint64(start-nextBlockHeight)) + 1 // One atom too many
	tspendFee := uint64(0)
	tspend := g.CreateTreasuryTSpend(privKey, []chaingen.AddressAmountTuple{
		{
			Amount: dcrutil.Amount(tspendAmount - tspendFee),
		},
	},
		dcrutil.Amount(tspendFee), expiry)
	tspendHash := tspend.TxHash()
	t.Logf("tspend %v amount %v fee %v", tspendHash, tspendAmount-tspendFee,
		tspendFee)

	// ---------------------------------------------------------------------
	// Generate enough blocks to get to TVI.
	//
	//   ... -> bva19 -> bpretvi0 -> bpretvi1
	// ---------------------------------------------------------------------

	// Generate votes up to TVI. This is legal however they should NOT be
	// counted in the totals since they are outside of the voting window.
	outs := g.OldestCoinbaseOuts()
	for i := uint32(0); i < start-nextBlockHeight; i++ {
		name := fmt.Sprintf("bpretvi%v", i)
		g.NextBlock(name, nil, outs[1:], replaceTreasuryVersions,
			replaceCoinbase)
		g.SaveTipCoinbaseOutsWithTreasury()
		g.AcceptTipBlock()
		outs = g.OldestCoinbaseOuts()
	}

	// ---------------------------------------------------------------------
	// Generate a TVI worth of rewards and try to spend more.
	//
	//   ... -> b0 ... -> b7
	//                 \-> btoomuch0
	// ---------------------------------------------------------------------

	voteCount := params.TicketsPerBlock
	for i := uint64(0); i < tvi*mul; i++ {
		name := fmt.Sprintf("b%v", i)
		g.NextBlock(name, nil, outs[1:], replaceTreasuryVersions,
			replaceCoinbase,
			addTSpendVotes(t, []*chainhash.Hash{&tspendHash},
				[]stake.TreasuryVoteT{stake.TreasuryVoteYes},
				voteCount, false))
		g.SaveTipCoinbaseOutsWithTreasury()
		g.AcceptTipBlock()
		outs = g.OldestCoinbaseOuts()
	}

	// Assert treasury balance is 1 atom less than calculated amount.
	ts, err := getTreasuryState(g, g.Tip().BlockHash())
	if err != nil {
		t.Fatal(err)
	}
	if int64(tspendAmount-tspendFee)-ts.Balance != 1 {
		t.Fatalf("Assert treasury balance error: got %v want %v",
			ts.Balance, int64(tspendAmount-tspendFee)-ts.Balance)
	}

	// Try spending 1 atom more than treasury balance.
	name := "btoomuch0"
	g.NextBlock(name, nil, outs[1:], replaceTreasuryVersions,
		replaceCoinbase,
		func(b *wire.MsgBlock) {
			// Add TSpend
			b.AddSTransaction(tspend)
		})
	g.RejectTipBlock(ErrInvalidExpenditure)
}

func TestTSpendExpenditures2(t *testing.T) {
	// Use a set of test chain parameters which allow for quicker vote
	// activation as compared to various existing network params.
	params := quickVoteActivationParams()

	// Clone the parameters so they can be mutated, find the correct deployment
	// for the fix sequence locks agenda, and, finally, ensure it is always
	// available to vote by removing the time constraints to prevent test
	// failures when the real expiration time passes.
	const tVoteID = chaincfg.VoteIDTreasury
	params = cloneParams(params)
	tVersion, deployment, err := findDeployment(params, tVoteID)
	if err != nil {
		t.Fatal(err)
	}
	removeDeploymentTimeConstraints(deployment)

	// Dave off tvi and mul.
	tvi := params.TreasuryVoteInterval
	mul := params.TreasuryVoteIntervalMultiplier

	// Create a test harness initialized with the genesis block as the tip.
	g, teardownFunc := newChaingenHarness(t, params, "treasurytest")
	defer teardownFunc()

	// replaceTreasuryVersions is a munge function which modifies the
	// provided block by replacing the block, stake, and vote versions with the
	// fix sequence locks deployment version.
	replaceTreasuryVersions := func(b *wire.MsgBlock) {
		chaingen.ReplaceBlockVersion(int32(tVersion))(b)
		chaingen.ReplaceStakeVersion(tVersion)(b)
		chaingen.ReplaceVoteVersions(tVersion)(b)
	}

	// ---------------------------------------------------------------------
	// Generate and accept enough blocks with the appropriate vote bits set
	// to reach one block prior to the treasury agenda becoming active.
	// ---------------------------------------------------------------------

	g.AdvanceToStakeValidationHeight()
	g.AdvanceFromSVHToActiveAgenda(tVoteID)

	// Ensure treasury agenda is active.
	gotActive, err := g.chain.IsTreasuryAgendaActive()
	if err != nil {
		t.Fatalf("IsTreasuryAgendaActive: %v", err)
	}
	if !gotActive {
		t.Fatalf("IsTreasuryAgendaActive: expected enabled treasury")
	}

	// ---------------------------------------------------------------------
	// Generate enough blocks to get to TVI.
	//
	//   ... -> bva19 -> bpretvi0 -> bpretvi1
	// ---------------------------------------------------------------------

	nextBlockHeight := g.Tip().Header.Height + 1
	expiry := standalone.CalculateTSpendExpiry(int64(nextBlockHeight), tvi,
		mul)
	start, err := standalone.CalculateTSpendWindowStart(expiry, tvi, mul)
	if err != nil {
		t.Fatal(err)
	}

	// Generate up to TVI blocks.
	outs := g.OldestCoinbaseOuts()
	for i := uint32(0); i < start-nextBlockHeight; i++ {
		name := fmt.Sprintf("bpretvi%v", i)
		g.NextBlock(name, nil, outs[1:], replaceTreasuryVersions,
			replaceCoinbase)
		g.SaveTipCoinbaseOutsWithTreasury()
		g.AcceptTipBlock()
		outs = g.OldestCoinbaseOuts()
	}

	// ---------------------------------------------------------------------
	// Generate 2*Policy*TVI worth of rewards.
	//
	//   ... -> b0 ... -> b63
	//                 \-> btoomuch0
	// ---------------------------------------------------------------------
	for i := uint64(0); i < 2*tvi*mul*params.TreasuryVoteIntervalPolicy; i++ {
		name := fmt.Sprintf("b%v", i)
		g.NextBlock(name, nil, outs[1:], replaceTreasuryVersions,
			replaceCoinbase)
		g.SaveTipCoinbaseOutsWithTreasury()
		g.AcceptTipBlock()
		outs = g.OldestCoinbaseOuts()
	}

	// ---------------------------------------------------------------------
	// Generate a TVI worth of rewards and try to spend more.
	//
	//   ... -> bv0 ... -> bv7
	//                 \-> btoomuch0
	// ---------------------------------------------------------------------

	// Create TSPEND in mempool for 150% of last policy window gain.
	nextBlockHeight = g.Tip().Header.Height + 1 - uint32(tvi) // travel a bit back
	expiry = standalone.CalculateTSpendExpiry(int64(nextBlockHeight), tvi,
		mul)
	start, err = standalone.CalculateTSpendWindowStart(expiry, tvi, mul)
	if err != nil {
		t.Fatal(err)
	}
	end, err := standalone.CalculateTSpendWindowEnd(expiry, tvi)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("nbh %v expiry %v start %v end %v",
		nextBlockHeight, expiry, start, end)

	// This calculation is inprecise due to the blockreward going down.
	x := tvi * mul * params.TreasuryVoteIntervalPolicy * devsub
	tspendAmount := x + x/2 + devsub*2 // 150% including maturity
	tspendFee := uint64(0)
	tspend := g.CreateTreasuryTSpend(privKey, []chaingen.AddressAmountTuple{
		{
			Amount: dcrutil.Amount(tspendAmount - tspendFee),
		},
	},
		dcrutil.Amount(tspendFee), expiry)
	tspendHash := tspend.TxHash()
	t.Logf("tspend %v amount %v fee %v", tspendHash, tspendAmount-tspendFee,
		tspendFee)

	voteCount := params.TicketsPerBlock
	for i := uint64(0); i < tvi*mul; i++ {
		name := fmt.Sprintf("bv%v", i)
		g.NextBlock(name, nil, outs[1:], replaceTreasuryVersions,
			replaceCoinbase,
			addTSpendVotes(t, []*chainhash.Hash{&tspendHash},
				[]stake.TreasuryVoteT{stake.TreasuryVoteYes},
				voteCount, false))
		g.SaveTipCoinbaseOutsWithTreasury()
		g.AcceptTipBlock()
		outs = g.OldestCoinbaseOuts()
	}

	// Try spending > ~150% than treasury gain over policy interval.
	name := "btoomuch0"
	g.NextBlock(name, nil, outs[1:], replaceTreasuryVersions,
		replaceCoinbase,
		func(b *wire.MsgBlock) {
			// Add TSpend
			b.AddSTransaction(tspend)
		})
	g.RejectTipBlock(ErrInvalidExpenditure)
}

func TestTSpendDupVote(t *testing.T) {
	// Use a set of test chain parameters which allow for quicker vote
	// activation as compared to various existing network params.
	params := quickVoteActivationParams()

	// Clone the parameters so they can be mutated, find the correct deployment
	// for the fix sequence locks agenda, and, finally, ensure it is always
	// available to vote by removing the time constraints to prevent test
	// failures when the real expiration time passes.
	const tVoteID = chaincfg.VoteIDTreasury
	params = cloneParams(params)
	tVersion, deployment, err := findDeployment(params, tVoteID)
	if err != nil {
		t.Fatal(err)
	}
	removeDeploymentTimeConstraints(deployment)

	// Dave off tvi and mul.
	tvi := params.TreasuryVoteInterval
	mul := params.TreasuryVoteIntervalMultiplier

	// Create a test harness initialized with the genesis block as the tip.
	g, teardownFunc := newChaingenHarness(t, params, "treasurytest")
	defer teardownFunc()

	// replaceTreasuryVersions is a munge function which modifies the
	// provided block by replacing the block, stake, and vote versions with the
	// fix sequence locks deployment version.
	replaceTreasuryVersions := func(b *wire.MsgBlock) {
		chaingen.ReplaceBlockVersion(int32(tVersion))(b)
		chaingen.ReplaceStakeVersion(tVersion)(b)
		chaingen.ReplaceVoteVersions(tVersion)(b)
	}

	// ---------------------------------------------------------------------
	// Generate and accept enough blocks with the appropriate vote bits set
	// to reach one block prior to the treasury agenda becoming active.
	// ---------------------------------------------------------------------

	g.AdvanceToStakeValidationHeight()
	g.AdvanceFromSVHToActiveAgenda(tVoteID)

	// Ensure treasury agenda is active.
	gotActive, err := g.chain.IsTreasuryAgendaActive()
	if err != nil {
		t.Fatalf("IsTreasuryAgendaActive: %v", err)
	}
	if !gotActive {
		t.Fatalf("IsTreasuryAgendaActive: expected enabled treasury")
	}

	// ---------------------------------------------------------------------
	// Create two TSPEND with invalid bits and duplicate votes.
	// ---------------------------------------------------------------------
	nextBlockHeight := g.Tip().Header.Height + 1
	expiry := standalone.CalculateTSpendExpiry(int64(nextBlockHeight), tvi,
		mul)
	start, err := standalone.CalculateTSpendWindowStart(expiry, tvi, mul)
	if err != nil {
		t.Fatal(err)
	}
	end, err := standalone.CalculateTSpendWindowEnd(expiry, tvi)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("nbh %v expiry %v start %v end %v",
		nextBlockHeight, expiry, start, end)

	tspendAmount := devsub * (tvi*mul - uint64(params.CoinbaseMaturity) +
		uint64(start-nextBlockHeight))
	tspendFee := uint64(0)
	tspend := g.CreateTreasuryTSpend(privKey,
		[]chaingen.AddressAmountTuple{

			{
				Amount: dcrutil.Amount(tspendAmount - tspendFee),
			},
		},
		dcrutil.Amount(tspendFee), expiry)
	tspendHash := tspend.TxHash()
	tspend2 := g.CreateTreasuryTSpend(privKey,
		[]chaingen.AddressAmountTuple{
			{
				Amount: dcrutil.Amount(tspendAmount - tspendFee),
			},
		},
		dcrutil.Amount(tspendFee), expiry)
	tspendHash2 := tspend2.TxHash()
	t.Logf("tspend %v amount %v fee %v",
		tspendHash, tspendAmount-tspendFee, tspendFee)
	t.Logf("tspend2 %v amount %v fee %v",
		tspendHash2, tspendAmount-tspendFee, tspendFee)

	// ---------------------------------------------------------------------
	// Generate enough blocks to get to TVI.
	//
	//   ... -> bva19 -> bpretvi0 -> bpretvi1
	// ---------------------------------------------------------------------

	// Generate votes up to TVI. This is legal however they should NOT be
	// counted in the totals since they are outside of the voting window.
	outs := g.OldestCoinbaseOuts()
	for i := uint32(0); i < start-nextBlockHeight; i++ {
		name := fmt.Sprintf("bpretvi%v", i)
		g.NextBlock(name, nil, outs[1:], replaceTreasuryVersions,
			replaceCoinbase)
		g.SaveTipCoinbaseOutsWithTreasury()
		g.AcceptTipBlock()
		outs = g.OldestCoinbaseOuts()
	}

	// ---------------------------------------------------------------------
	//   ... -> pretvi1
	//       \-> bdv0
	// ---------------------------------------------------------------------

	startTip := g.TipName()
	voteCount := params.TicketsPerBlock
	g.NextBlock("bdv0", nil, outs[1:], replaceTreasuryVersions,
		replaceCoinbase,
		addTSpendVotes(t,
			[]*chainhash.Hash{
				&tspendHash,
				&tspendHash,
			},
			[]stake.TreasuryVoteT{
				stake.TreasuryVoteYes,
				stake.TreasuryVoteYes,
			},
			voteCount, true))
	g.RejectTipBlock(ErrBadTxInput)

	// ---------------------------------------------------------------------
	//   ... -> pretvi1
	//       \-> bdv1
	// ---------------------------------------------------------------------

	g.SetTip(startTip)
	g.NextBlock("bdv1", nil, outs[1:], replaceTreasuryVersions,
		replaceCoinbase,
		addTSpendVotes(t,
			[]*chainhash.Hash{
				&tspendHash2,
			},
			[]stake.TreasuryVoteT{
				0x00, // Invalid bits
			},
			voteCount, true))
	g.RejectTipBlock(ErrBadTxInput)
}

func TestTSpendTooManyTSpend(t *testing.T) {
	// Use a set of test chain parameters which allow for quicker vote
	// activation as compared to various existing network params.
	params := quickVoteActivationParams()

	// Clone the parameters so they can be mutated, find the correct deployment
	// for the fix sequence locks agenda, and, finally, ensure it is always
	// available to vote by removing the time constraints to prevent test
	// failures when the real expiration time passes.
	const tVoteID = chaincfg.VoteIDTreasury
	params = cloneParams(params)
	tVersion, deployment, err := findDeployment(params, tVoteID)
	if err != nil {
		t.Fatal(err)
	}
	removeDeploymentTimeConstraints(deployment)

	// Dave off tvi and mul.
	tvi := params.TreasuryVoteInterval
	mul := params.TreasuryVoteIntervalMultiplier

	// Create a test harness initialized with the genesis block as the tip.
	g, teardownFunc := newChaingenHarness(t, params, "treasurytest")
	defer teardownFunc()

	// replaceTreasuryVersions is a munge function which modifies the
	// provided block by replacing the block, stake, and vote versions with the
	// fix sequence locks deployment version.
	replaceTreasuryVersions := func(b *wire.MsgBlock) {
		chaingen.ReplaceBlockVersion(int32(tVersion))(b)
		chaingen.ReplaceStakeVersion(tVersion)(b)
		chaingen.ReplaceVoteVersions(tVersion)(b)
	}

	// ---------------------------------------------------------------------
	// Generate and accept enough blocks with the appropriate vote bits set
	// to reach one block prior to the treasury agenda becoming active.
	// ---------------------------------------------------------------------

	g.AdvanceToStakeValidationHeight()
	g.AdvanceFromSVHToActiveAgenda(tVoteID)

	// Ensure treasury agenda is active.
	gotActive, err := g.chain.IsTreasuryAgendaActive()
	if err != nil {
		t.Fatalf("IsTreasuryAgendaActive: %v", err)
	}
	if !gotActive {
		t.Fatalf("IsTreasuryAgendaActive: expected enabled treasury")
	}

	// ---------------------------------------------------------------------
	// Create two TSPEND with invalid bits and duplicate votes.
	// ---------------------------------------------------------------------
	nextBlockHeight := g.Tip().Header.Height + 1
	expiry := standalone.CalculateTSpendExpiry(int64(nextBlockHeight), tvi,
		mul)
	start, err := standalone.CalculateTSpendWindowStart(expiry, tvi, mul)
	if err != nil {
		t.Fatal(err)
	}
	end, err := standalone.CalculateTSpendWindowEnd(expiry, tvi)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("nbh %v expiry %v start %v end %v",
		nextBlockHeight, expiry, start, end)

	tspendAmount := devsub * (tvi*mul - uint64(params.CoinbaseMaturity) +
		uint64(start-nextBlockHeight))
	tspendFee := uint64(0)
	maxTspends := 7
	tspends := make([]*wire.MsgTx, maxTspends+1)
	tspendHashes := make([]*chainhash.Hash, maxTspends+1)
	tspendVotes := make([]stake.TreasuryVoteT, maxTspends+1)
	for i := 0; i < maxTspends+1; i++ {
		tspends[i] = g.CreateTreasuryTSpend(privKey,
			[]chaingen.AddressAmountTuple{
				{
					Amount: dcrutil.Amount(tspendAmount - tspendFee),
				},
			},
			dcrutil.Amount(tspendFee), expiry)
		hash := tspends[i].TxHash()
		tspendHashes[i] = &hash
		tspendVotes[i] = stake.TreasuryVoteYes
		t.Logf("%v: tspend %v amount %v fee %v", i, hash,
			tspendAmount-tspendFee, tspendFee)
	}

	// ---------------------------------------------------------------------
	// Generate enough blocks to get to TVI.
	//
	//   ... -> bva19 -> bpretvi0 -> bpretvi1
	// ---------------------------------------------------------------------

	// Generate votes up to TVI. This is legal however they should NOT be
	// counted in the totals since they are outside of the voting window.
	outs := g.OldestCoinbaseOuts()
	for i := uint32(0); i < start-nextBlockHeight; i++ {
		name := fmt.Sprintf("bpretvi%v", i)
		g.NextBlock(name, nil, outs[1:], replaceTreasuryVersions,
			replaceCoinbase)
		g.SaveTipCoinbaseOutsWithTreasury()
		g.AcceptTipBlock()
		outs = g.OldestCoinbaseOuts()
	}

	// ---------------------------------------------------------------------
	// 8 votes is illegal and therefore this SSGEN is NOT recognized as a
	// vote and therefore it falls through the type if/else case in
	// validate.go CheckTransactionSanity.
	//
	//   ... -> pretvi1
	//       \-> bdv0
	// ---------------------------------------------------------------------

	voteCount := params.TicketsPerBlock
	g.NextBlock("bdv0", nil, outs[1:], replaceTreasuryVersions,
		replaceCoinbase,
		addTSpendVotes(t, tspendHashes, tspendVotes, voteCount, true))
	g.RejectTipBlock(ErrBadTxInput)
}

func TestTSpendWindow(t *testing.T) {
	// Use a set of test chain parameters which allow for quicker vote
	// activation as compared to various existing network params.
	params := quickVoteActivationParams()

	// Clone the parameters so they can be mutated, find the correct deployment
	// for the fix sequence locks agenda, and, finally, ensure it is always
	// available to vote by removing the time constraints to prevent test
	// failures when the real expiration time passes.
	const tVoteID = chaincfg.VoteIDTreasury
	params = cloneParams(params)
	tVersion, deployment, err := findDeployment(params, tVoteID)
	if err != nil {
		t.Fatal(err)
	}
	removeDeploymentTimeConstraints(deployment)

	// Dave off tvi and mul.
	tvi := params.TreasuryVoteInterval
	mul := params.TreasuryVoteIntervalMultiplier

	// Create a test harness initialized with the genesis block as the tip.
	g, teardownFunc := newChaingenHarness(t, params, "treasurytest")
	defer teardownFunc()

	// replaceTreasuryVersions is a munge function which modifies the
	// provided block by replacing the block, stake, and vote versions with the
	// fix sequence locks deployment version.
	replaceTreasuryVersions := func(b *wire.MsgBlock) {
		chaingen.ReplaceBlockVersion(int32(tVersion))(b)
		chaingen.ReplaceStakeVersion(tVersion)(b)
		chaingen.ReplaceVoteVersions(tVersion)(b)
	}

	// ---------------------------------------------------------------------
	// Generate and accept enough blocks with the appropriate vote bits set
	// to reach one block prior to the treasury agenda becoming active.
	// ---------------------------------------------------------------------

	g.AdvanceToStakeValidationHeight()
	g.AdvanceFromSVHToActiveAgenda(tVoteID)

	// Ensure treasury agenda is active.
	gotActive, err := g.chain.IsTreasuryAgendaActive()
	if err != nil {
		t.Fatalf("IsTreasuryAgendaActive: %v", err)
	}
	if !gotActive {
		t.Fatalf("IsTreasuryAgendaActive: expected enabled treasury")
	}

	// ---------------------------------------------------------------------
	// Create TSPEND in mempool
	// ---------------------------------------------------------------------
	nextBlockHeight := g.Tip().Header.Height + uint32(tvi*mul*4) + 1
	expiry := standalone.CalculateTSpendExpiry(int64(nextBlockHeight), tvi,
		mul)
	start, err := standalone.CalculateTSpendWindowStart(expiry, tvi, mul)
	if err != nil {
		t.Fatal(err)
	}
	end, err := standalone.CalculateTSpendWindowEnd(expiry, tvi)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("nbh %v expiry %v start %v end %v",
		nextBlockHeight, expiry, start, end)

	tspendAmount := devsub * (tvi*mul - uint64(params.CoinbaseMaturity) +
		uint64(start-nextBlockHeight))
	tspendFee := uint64(0)
	tspend := g.CreateTreasuryTSpend(privKey, []chaingen.AddressAmountTuple{
		{
			Amount: dcrutil.Amount(tspendAmount - tspendFee),
		},
	},
		dcrutil.Amount(tspendFee), expiry)
	tspendHash := tspend.TxHash()
	t.Logf("tspend %v amount %v fee %v", tspendHash, tspendAmount-tspendFee,
		tspendFee)

	// ---------------------------------------------------------------------
	// Generate enough blocks to get to TVI.
	//
	//   ... -> bva19 -> bpretvi0 -> bpretvi1
	// ---------------------------------------------------------------------

	// Generate votes up to TVI. This is legal however they should NOT be
	// counted in the totals since they are outside of the voting window.
	outs := g.OldestCoinbaseOuts()
	for i := uint32(0); i < start-nextBlockHeight; i++ {
		name := fmt.Sprintf("bpretvi%v", i)
		g.NextBlock(name, nil, outs[1:], replaceTreasuryVersions,
			replaceCoinbase)
		g.SaveTipCoinbaseOutsWithTreasury()
		g.AcceptTipBlock()
		outs = g.OldestCoinbaseOuts()
	}

	// ---------------------------------------------------------------------
	// Generate a TVI worth of rewards and try to spend more.
	//
	//   ... -> b0 ... -> b7
	//                 \-> bvidaltoosoon0
	// ---------------------------------------------------------------------

	voteCount := params.TicketsPerBlock
	for i := uint64(0); i < tvi*mul; i++ {
		name := fmt.Sprintf("b%v", i)
		g.NextBlock(name, nil, outs[1:], replaceTreasuryVersions,
			replaceCoinbase,
			addTSpendVotes(t, []*chainhash.Hash{&tspendHash},
				[]stake.TreasuryVoteT{stake.TreasuryVoteYes},
				voteCount, false))
		g.SaveTipCoinbaseOutsWithTreasury()
		g.AcceptTipBlock()
		outs = g.OldestCoinbaseOuts()
	}

	// Try accepting TSpend from the future.
	g.NextBlock("bvidaltoosoon0", nil, outs[1:], replaceTreasuryVersions,
		replaceCoinbase,
		func(b *wire.MsgBlock) {
			// Add TSpend
			b.AddSTransaction(tspend)
		})
	g.RejectTipBlock(ErrInvalidTSpendWindow)
}

type testLogger struct {
	t *testing.T
}

func (l *testLogger) Write(p []byte) (int, error) {
	s := string(p)
	s = strings.TrimRight(s, "\r\n")
	l.t.Log(s)
	return 0, nil
}

func logDcrd(t *testing.T, enable bool) {
	if enable == false {
		return
	}
	tlog := testLogger{t}
	backendLogger := slog.NewBackend(&tlog)
	log := backendLogger.Logger("TRSY")
	log.SetLevel(slog.LevelTrace)
	UseLogger(log)
	UseTreasuryLogger(log)
}

func TestTSpendExists(t *testing.T) {
	logDcrd(t, false) // Make true to log dcrd output.

	// Use a set of test chain parameters which allow for quicker vote
	// activation as compared to various existing network params.
	params := quickVoteActivationParams()

	// Clone the parameters so they can be mutated, find the correct deployment
	// for the fix sequence locks agenda, and, finally, ensure it is always
	// available to vote by removing the time constraints to prevent test
	// failures when the real expiration time passes.
	const tVoteID = chaincfg.VoteIDTreasury
	params = cloneParams(params)
	tVersion, deployment, err := findDeployment(params, tVoteID)
	if err != nil {
		t.Fatal(err)
	}
	removeDeploymentTimeConstraints(deployment)

	// Dave off tvi and mul.
	tvi := params.TreasuryVoteInterval
	mul := params.TreasuryVoteIntervalMultiplier
	cbm := params.CoinbaseMaturity

	// Create a test harness initialized with the genesis block as the tip.
	g, teardownFunc := newChaingenHarness(t, params, "treasurytest")
	defer teardownFunc()

	// replaceTreasuryVersions is a munge function which modifies the
	// provided block by replacing the block, stake, and vote versions with the
	// fix sequence locks deployment version.
	replaceTreasuryVersions := func(b *wire.MsgBlock) {
		chaingen.ReplaceBlockVersion(int32(tVersion))(b)
		chaingen.ReplaceStakeVersion(tVersion)(b)
		chaingen.ReplaceVoteVersions(tVersion)(b)
	}

	// ---------------------------------------------------------------------
	// Generate and accept enough blocks with the appropriate vote bits set
	// to reach one block prior to the treasury agenda becoming active.
	// ---------------------------------------------------------------------

	g.AdvanceToStakeValidationHeight()
	g.AdvanceFromSVHToActiveAgenda(tVoteID)

	// Ensure treasury agenda is active.
	gotActive, err := g.chain.IsTreasuryAgendaActive()
	if err != nil {
		t.Fatalf("IsTreasuryAgendaActive: %v", err)
	}
	if !gotActive {
		t.Fatalf("IsTreasuryAgendaActive: expected enabled treasury")
	}

	// splitSecondRegularTxOutputs is a munge function which modifies the
	// provided block by replacing its second regular transaction with one
	// that creates several utxos.
	const splitTxNumOutputs = 6
	splitSecondRegularTxOutputs := func(b *wire.MsgBlock) {
		// Remove the current outputs of the second transaction while
		// saving the relevant public key script, input amount, and fee
		// for below.
		tx := b.Transactions[1]
		inputAmount := tx.TxIn[0].ValueIn
		pkScript := tx.TxOut[0].PkScript
		fee := inputAmount - tx.TxOut[0].Value
		tx.TxOut = tx.TxOut[:0]

		// Final outputs are the input amount minus the fee split into
		// more than one output.  These are intended to provide
		// additional utxos for testing.
		outputAmount := inputAmount - fee
		splitAmount := outputAmount / splitTxNumOutputs
		for i := 0; i < splitTxNumOutputs; i++ {
			if i == splitTxNumOutputs-1 {
				splitAmount = outputAmount -
					splitAmount*(splitTxNumOutputs-1)
			}
			tx.AddTxOut(wire.NewTxOut(splitAmount, pkScript))
		}
	}

	// Generate spendable outputs to do fork tests with.
	var txOuts [][]chaingen.SpendableOut
	genBlocks := cbm * 8
	for i := uint16(0); i < genBlocks; i++ {
		outs := g.OldestCoinbaseOuts()
		name := fmt.Sprintf("bouts%v", i)
		g.NextBlock(name, &outs[0], outs[1:], replaceTreasuryVersions,
			replaceCoinbase, splitSecondRegularTxOutputs)
		g.SaveTipCoinbaseOutsWithTreasury()
		g.AcceptTipBlock()

		souts := make([]chaingen.SpendableOut, 0, splitTxNumOutputs)
		for j := 0; j < splitTxNumOutputs; j++ {
			spendableOut := chaingen.MakeSpendableOut(g.Tip(), 1,
				uint32(j))
			souts = append(souts, spendableOut)
		}
		txOuts = append(txOuts, souts)
	}

	// ---------------------------------------------------------------------
	// Create TSPEND in mempool
	// ---------------------------------------------------------------------
	nextBlockHeight := g.Tip().Header.Height + 1
	expiry := standalone.CalculateTSpendExpiry(int64(nextBlockHeight), tvi,
		mul)
	start, err := standalone.CalculateTSpendWindowStart(expiry, tvi, mul)
	if err != nil {
		t.Fatal(err)
	}
	end, err := standalone.CalculateTSpendWindowEnd(expiry, tvi)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("nbh %v expiry %v start %v end %v",
		nextBlockHeight, expiry, start, end)

	tspendAmount := uint64(devsub)
	tspendFee := uint64(0)
	tspend := g.CreateTreasuryTSpend(privKey, []chaingen.AddressAmountTuple{
		{
			Amount: dcrutil.Amount(tspendAmount - tspendFee),
		},
	},
		dcrutil.Amount(tspendFee), expiry)
	tspendHash := tspend.TxHash()
	t.Logf("tspend %v amount %v fee %v", tspendHash, tspendAmount-tspendFee,
		tspendFee)

	// ---------------------------------------------------------------------
	// Generate enough blocks to get to TVI.
	//
	//   ... -> bouts -> bpretvi0 -> bpretvi1
	// ---------------------------------------------------------------------

	outs := g.OldestCoinbaseOuts()
	for i := uint32(0); i < start-nextBlockHeight; i++ {
		name := fmt.Sprintf("bpretvi%v", i)
		g.NextBlock(name, nil, outs[1:], replaceTreasuryVersions,
			replaceCoinbase)
		g.SaveTipCoinbaseOutsWithTreasury()
		g.AcceptTipBlock()
		outs = g.OldestCoinbaseOuts()
	}

	// ---------------------------------------------------------------------
	// Generate a TVI with votes.
	//
	//   ... -> b0 ... -> b3
	// ---------------------------------------------------------------------

	voteCount := params.TicketsPerBlock
	for i := uint64(0); i < tvi; i++ {
		name := fmt.Sprintf("b%v", i)
		g.NextBlock(name, nil, outs[1:], replaceTreasuryVersions,
			replaceCoinbase,
			addTSpendVotes(t, []*chainhash.Hash{&tspendHash},
				[]stake.TreasuryVoteT{stake.TreasuryVoteYes},
				voteCount, false))
		g.SaveTipCoinbaseOutsWithTreasury()
		g.AcceptTipBlock()
		outs = g.OldestCoinbaseOuts()
	}

	// ---------------------------------------------------------------------
	// Generate a TVI and mine same TSpend, which has to be rejected.
	//
	//   ... -> be0 ... -> be3 ->
	//                         \-> bexists0
	// ---------------------------------------------------------------------

	startTip := g.TipName()
	for i := uint64(0); i < tvi; i++ {
		name := fmt.Sprintf("be%v", i)
		if i == 0 {
			// Mine tspend.
			g.NextBlock(name, nil, txOuts[i][1:],
				replaceTreasuryVersions,
				replaceCoinbase, func(b *wire.MsgBlock) {
					// Add TSpend
					b.AddSTransaction(tspend)
				})
		} else {
			g.NextBlock(name, nil, txOuts[i][1:],
				replaceTreasuryVersions,
				replaceCoinbase)
		}
		g.AcceptTipBlock()
	}
	oldTip := g.TipName()

	// Mine tspend again.
	g.NextBlock("bexists0", nil, outs[1:], replaceTreasuryVersions,
		replaceCoinbase,
		func(b *wire.MsgBlock) {
			// Add TSpend
			b.AddSTransaction(tspend)
		})
	g.RejectTipBlock(ErrTSpendExists)

	// ---------------------------------------------------------------------
	// Generate a TVI and mine same TSpend, should not exist since it is a
	// fork.
	//
	//      /-> be0 ... -> be3 -> bexists0
	// ... -> b3
	//      \-> bep0 ... -> bep3 -> bexists1
	// ---------------------------------------------------------------------

	g.SetTip(startTip)
	var nextFork uint64
	for i := uint64(0); i < tvi; i++ {
		name := fmt.Sprintf("bep%v", i)
		g.NextBlock(name, nil, txOuts[i][1:], replaceTreasuryVersions,
			replaceCoinbase)
		g.AcceptedToSideChainWithExpectedTip(oldTip)
		nextFork = i + 1
	}

	// Mine tspend again.
	g.NextBlock("bexists1", nil, txOuts[nextFork][1:], replaceTreasuryVersions,
		replaceCoinbase,
		func(b *wire.MsgBlock) {
			// Add TSpend
			b.AddSTransaction(tspend)
		})
	g.AcceptTipBlock()

	// ---------------------------------------------------------------------
	// Generate a TVI and mine same TSpend, should not exist since it is a
	// fork.
	//
	//      /-> be0 ... -> be3 -> bexists0
	// ... -> b3
	//      \-> bep0 ... -> bep3 -> bexists1 ->bep4
	//       \-> bepp0 ... -> bepp3 -> bexists2 -> bxxx0 .. bxxx3
	// ---------------------------------------------------------------------

	// Generate one more block to extend current best chain.
	name := fmt.Sprintf("bep%v", nextFork)
	g.NextBlock(name, nil, txOuts[nextFork+1][1:], replaceTreasuryVersions,
		replaceCoinbase)
	g.AcceptTipBlock()
	oldTip = g.TipName()
	oldHeight := g.Tip().Header.Height

	// Create bepp fork
	g.SetTip(startTip)
	txIdx := uint64(0)
	for i := uint64(0); i < tvi; i++ {
		name := fmt.Sprintf("bepp%v", i)
		g.NextBlock(name, nil, txOuts[i][1:],
			replaceTreasuryVersions, replaceCoinbase)
		g.AcceptedToSideChainWithExpectedTip(oldTip)
		txIdx = i + 1
	}

	// Mine tspend yet again.
	g.NextBlock("bexists2", nil, txOuts[txIdx][1:],
		replaceTreasuryVersions, replaceCoinbase,
		func(b *wire.MsgBlock) {
			// Add TSpend
			b.AddSTransaction(tspend)
		})
	g.AcceptedToSideChainWithExpectedTip(oldTip)
	txIdx++

	// Force reorg
	for i := uint64(0); i < tvi; i++ {
		name := fmt.Sprintf("bxxx%v", i)
		b := g.NextBlock(name, nil, txOuts[txIdx][1:],
			replaceTreasuryVersions, replaceCoinbase)
		if b.Header.Height <= oldHeight {
			g.AcceptedToSideChainWithExpectedTip(oldTip)
		} else {
			g.AcceptTipBlock()
		}
		txIdx++
	}
}
