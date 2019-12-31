// Copyright (c) 2019 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockchain

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/decred/dcrd/blockchain/stake/v3"
	"github.com/decred/dcrd/blockchain/v3/internal/dbnamespace"
	"github.com/decred/dcrd/chaincfg/chainhash"
	"github.com/decred/dcrd/database/v2"
	"github.com/decred/dcrd/dcrutil/v3"
)

var ()

const (
	// TreasuryMaxEntriesPerBlock is the maximum number of OP_TADD/OP_SPEND
	// transactions in a given block.
	TreasuryMaxEntriesPerBlock = 256
)

// TreasuryState records the treasury balance as of this block and it records
// the yet to mature adds and spends. The TADDS are positive and the TSPENDS
// are negative. Additionally the values are written in the exact same order as
// they appear in the block. This can be used to verify the correctness of the
// record if needed.
type TreasuryState struct {
	Balance int64   // Treasury balance as of this block
	Values  []int64 // All TADD/TSPEND values in this block (for use when block is mature)
}

// serializeTreasuryState serializes the TreasuryState structure
// for use in the database.
// The format is as follows:
// dbnamespace.ByteOrder.int64(treasury balance as of this block)
// dbnamespace.ByteOrder.int64(length of values arrays)
// []dbnamespace.ByteOrder.int64(all additions and subtractions from treasury
//   in this block)
func serializeTreasuryState(ts TreasuryState) ([]byte, error) {
	// Just a little sanity testing.
	if ts.Balance < 0 {
		return nil, fmt.Errorf("invalid treasury balance: %v",
			ts.Balance)
	}
	if len(ts.Values) > TreasuryMaxEntriesPerBlock {
		return nil, fmt.Errorf("invalid treasury values length: %v",
			len(ts.Values))
	}

	// Serialize TreasuryState.
	serializedData := new(bytes.Buffer)
	err := binary.Write(serializedData, dbnamespace.ByteOrder, ts.Balance)
	if err != nil {
		return nil, err
	}
	err = binary.Write(serializedData, dbnamespace.ByteOrder,
		int64(len(ts.Values)))
	if err != nil {
		return nil, err
	}
	for _, v := range ts.Values {
		err := binary.Write(serializedData, dbnamespace.ByteOrder, v)
		if err != nil {
			return nil, err
		}
	}
	return serializedData.Bytes(), nil
}

// deserializeTreasuryState deserializes a binary blob into a TreasuryState
// structure.
func deserializeTreasuryState(data []byte) (*TreasuryState, error) {
	var ts TreasuryState
	buf := bytes.NewReader(data)
	err := binary.Read(buf, dbnamespace.ByteOrder, &ts.Balance)
	if err != nil {
		return nil, fmt.Errorf("balance %v", err)
	}

	var count int64
	err = binary.Read(buf, dbnamespace.ByteOrder, &count)
	if err != nil {
		return nil, fmt.Errorf("count %v", err)
	}
	if count > TreasuryMaxEntriesPerBlock {
		return nil,
			fmt.Errorf("invalid treasury values length: %v", count)
	}

	ts.Values = make([]int64, count)
	for i := int64(0); i < count; i++ {
		err := binary.Read(buf, dbnamespace.ByteOrder, &ts.Values[i])
		if err != nil {
			return nil,
				fmt.Errorf("values read %v error %v", i, err)
		}
	}

	return &ts, nil
}

// dbPutTreasury inserts a treasury state record into the database.
func dbPutTreasury(dbTx database.Tx, hash chainhash.Hash, ts TreasuryState) error {
	// Serialize the current treasury state.
	serializedData, err := serializeTreasuryState(ts)
	if err != nil {
		return err
	}

	// Store the current treasury state into the database.
	meta := dbTx.Metadata()
	bucket := meta.Bucket(dbnamespace.TreasuryBucketName)
	return bucket.Put(hash[:], serializedData)
}

// dbFetchTreasury uses an existing database transaction to fetch the treasury
// state.
func dbFetchTreasury(dbTx database.Tx, hash chainhash.Hash) (*TreasuryState, error) {
	meta := dbTx.Metadata()
	bucket := meta.Bucket(dbnamespace.TreasuryBucketName)

	v := bucket.Get(hash[:])
	if v == nil {
		return nil, fmt.Errorf("treasury db missing key: %v", hash)
	}

	return deserializeTreasuryState(v)
}

// calculateTreasuryBalance calculates the treasury balance as of the provided
// node. It does that by moving back CoinbaseMaturity blocks and
// adding/subtracting the treasury updates to/from the *parent node*.
func (b *BlockChain) calculateTreasuryBalance(dbTx database.Tx, node *blockNode) (int64, error) {
	wantNode := node.RelativeAncestor(int64(b.chainParams.CoinbaseMaturity))
	if wantNode == nil {
		// Since the node does not exist we can safely assume the
		// balance is 0
		return 0, nil
	}

	// Current balance is in the parent node
	ts, err := dbFetchTreasury(dbTx, node.parent.hash)
	if err != nil {
		// Since the node.parent.hash does not exist in the treasury db
		// we can safely assume the balance is 0
		return 0, nil
	}

	// Add values to current balance
	valuesBlock, err := dbFetchBlockByNode(dbTx, wantNode)
	if err != nil {
		return 0, err
	}

	// XXX fetch ts from wantNode instead of doing this
	var netValue int64
	for _, v := range valuesBlock.MsgBlock().STransactions {
		if stake.IsTAdd(v) {
			// This is a TAdd, pull values out of block.
			for _, vv := range v.TxOut {
				netValue += vv.Value
			}
			continue
		}
		if stake.IsTSpend(v) {
			// This is a TSpend, pull values out of block.
			for _, vv := range v.TxIn {
				netValue -= vv.ValueIn
			}
			continue
		}
	}

	return ts.Balance + netValue, nil
}

// WriteTreasury inserts the current balance and the future treasury add/spend
// into the database.
func (b *BlockChain) writeTreasury(dbTx database.Tx, block *dcrutil.Block, node *blockNode) error {
	// Calculate balance as of this node
	balance, err := b.calculateTreasuryBalance(dbTx, node)
	if err != nil {
		return err
	}
	msgBlock := block.MsgBlock()
	ts := TreasuryState{
		Balance: balance,
		Values:  make([]int64, 0, len(msgBlock.Transactions)*2),
	}
	for _, v := range msgBlock.STransactions {
		if stake.IsTAdd(v) {
			// This is a TAdd, pull values out of block.
			for _, vv := range v.TxOut {
				ts.Values = append(ts.Values, vv.Value)
			}
			continue
		}
		if stake.IsTSpend(v) {
			// This is a TSpend, pull values out of block.
			for _, vv := range v.TxIn {
				ts.Values = append(ts.Values, -vv.ValueIn)
			}
			continue
		}
	}

	hash := block.Hash()
	return dbPutTreasury(dbTx, *hash, ts)
}

// AddTreasuryBucket creates the treasury database if it doesn't exist.
func addTreasuryBucket(db database.DB) error {
	return db.Update(func(dbTx database.Tx) error {
		_, err := dbTx.Metadata().CreateBucketIfNotExists(dbnamespace.TreasuryBucketName)
		return err
	})
}

// treasuryBalanceFailure returns a failure for the TreasuryBalance function.
// It exists because the return values should not be copy pasted.
func treasuryBalanceFailure(err error) (string, int64, int64, []int64, error) {
	return "", 0, 0, []int64{}, err
}

// TreasuryBalance returns the hash, height, treasury balance and the updates
// for the block that is CoinbaseMaturity from now.  If there is no hash
// provided it'll return the values for bestblock.
func (b *BlockChain) TreasuryBalance(hash *string) (string, int64, int64, []int64, error) {
	// Use best block if a hash is not provided.
	if hash == nil {
		best := b.BestSnapshot()
		h := best.Hash.String()
		hash = &h
	}

	// Retrieve block node.
	ch, err := chainhash.NewHashFromStr(*hash)
	if err != nil {
		return treasuryBalanceFailure(err)
	}
	node := b.index.LookupNode(ch)
	if node == nil || !b.index.NodeStatus(node).HaveData() {
		return treasuryBalanceFailure(
			fmt.Errorf("block %s is not known", *hash))
	}
	if ok, _ := b.isTreasuryAgendaActive(node); !ok {
		return treasuryBalanceFailure(fmt.Errorf("treasury not active"))
	}

	// Retrieve treasury bits.
	var ts *TreasuryState
	err = b.db.View(func(dbTx database.Tx) error {
		ts, err = dbFetchTreasury(dbTx, *ch)
		return err
	})
	if err != nil {
		return treasuryBalanceFailure(err)
	}

	return *hash, node.height, ts.Balance, ts.Values, nil
}
