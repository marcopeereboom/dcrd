// Copyright (c) 2019 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockchain

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/decred/dcrd/blockchain/stake/v2"
	"github.com/decred/dcrd/blockchain/v2/internal/dbnamespace"
	"github.com/decred/dcrd/chaincfg/chainhash"
	"github.com/decred/dcrd/database/v2"
	"github.com/decred/dcrd/dcrutil/v2"
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
//
// XXX this really doesn't belong here but there is no other way to share this
// between blockchain and stake package.
type TreasuryState struct {
	Balance int64   // Treasury balance as of this block
	Values  []int64 // All TADD/TSPEND values in this block (for use when block is mature)
}

// serializeTreasuryState serializes the TreasuryState structure
// for use in the database.
// The format is as follows:
// littleendian.int64(treasury balance as of this block)
// littleendian.int64(length of values arrays)
// []littleendian.int64(all additions and subtractions from treasury in this
//   block)
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

// deserializeTreasuryState desrializes a binary blob into a
// TreasuryState structure.
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

// DbPutTreasury inserts a treasury state record into the database.
func DbPutTreasury(dbTx database.Tx, hash chainhash.Hash, ts TreasuryState) error {
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

// DbFetchTreasury uses an existing database transaction to fetch the treasury
// state.
func DbFetchTreasury(dbTx database.Tx, hash chainhash.Hash) (*TreasuryState, error) {
	meta := dbTx.Metadata()
	bucket := meta.Bucket(dbnamespace.TreasuryBucketName)

	v := bucket.Get(hash[:])
	if v == nil {
		return nil, fmt.Errorf("treasury db missing key: %v", hash)
	}

	return deserializeTreasuryState(v)
}

func (b *BlockChain) calculateTreasuryBalance(block *dcrutil.Block, node *blockNode) (int64, error) {
	wantNode := node.RelativeAncestor(int64(b.chainParams.CoinbaseMaturity))
	if wantNode == nil {
		// Since the node does not exist we can safely assume the
		// balance is 0
		return 0, nil
	}

	return 0, fmt.Errorf("not yet")
}

// WriteTreasury inserts the current balance and the future treasury add/spend
// into the database.
func (b *BlockChain) WriteTreasury(dbTx database.Tx, block *dcrutil.Block, node *blockNode) error {
	msgBlock := block.MsgBlock()
	ts := TreasuryState{
		Balance: 0, // XXX
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
	return DbPutTreasury(dbTx, *hash, ts)
}

// AddTreasuryBucket creates the treasury database if it doesn't exist.
func AddTreasuryBucket(db database.DB) error {
	return db.Update(func(dbTx database.Tx) error {
		_, err := dbTx.Metadata().CreateBucketIfNotExists(dbnamespace.TreasuryBucketName)
		return err
	})
}
