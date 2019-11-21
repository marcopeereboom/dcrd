// Copyright (c) 2019 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package stake

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/decred/dcrd/blockchain/stake/v2/internal/dbnamespace"
	"github.com/decred/dcrd/chaincfg/chainhash"
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

func AddTreasuryBucket(db database.DB) error {
	return db.Update(func(dbTx database.Tx) error {
		_, err := dbTx.Metadata().CreateBucketIfNotExists(dbnamespace.TreasuryBucketName)
		return err
	})
}

// TreasuryState records the treasury balance as of this block and it records
// the yet to mature adds and spends. The TADDS are positive and the TSPENDS
// are negative. Additionally the values are written in the exact same order as
// they appear in the block. This can be used to verify the correctness of the
// record if needed.
type TreasuryState struct {
	Balance int64   // Treasury balance as of this block
	Values  []int64 // All TADD/TSPEND values in this block (for use when block is mature)
}

func deserializeTreasuryState(data []byte) (*TreasuryState, error) {
	var ts TreasuryState
	buf := bytes.NewReader(data)
	err := binary.Read(buf, binary.LittleEndian, &ts.Balance)
	if err != nil {
		return nil, ticketDBError(fmt.Sprintf("balance %v", err))
	}

	var count int
	err := binary.Read(buf, binary.LittleEndian, &count)
	if err != nil {
		return nil, ticketDBError(fmt.Sprintf("count %v", err))
	}

	ts.Values = make([]int64, count)
	for i := 0; i < count; i++ {
		err := binary.Read(buf, binary.LittleEndian, &Values[i])
		if err != nil {
			return nil, ticketDBError(fmt.Sprintf("values %v %v",
				i, err))
		}
	}

	return &ts, nil
}

func serializeTreasuryState(ts TreasuryState) []byte {
	serializedData := new(bytes.Buffer)

	err := binary.Write(serializedData, binary.LittleEndian, ts.Balance)
	if err != nil {
		panic("serializeTreasuryState ts.Balance")
	}
	err = binary.Write(serializedData, binary.LittleEndian, len(ts.Values))
	if err != nil {
		panic("serializeTreasuryState len(ts.Values)")
	}
	for k, v := range ts.Values {
		err := binary.Write(serializedData, binary.LittleEndian, v)
		if err != nil {
			panic(fmt.Sprintf("serializeTreasuryState k=%v v=%v",
				k, v))
		}
	}
	return serializedData.Bytes()
}

// DbPutTreasury inserts a treasury record into the database.
func DbPutTreasury(dbTx database.Tx, ts TreasuryState) error {
	// Serialize the current treasury state.
	serializedData := serializeTreasuryState(ts)

	// Store the current treasury state into the database.
	return dbTx.Metadata().Put(dbnamespace.TreasuryBucketName, serializedData)
}

// dbFetchTreasury uses an existing database transaction to fetch the best
// treasury state.
func dbFetchTreasury(dbTx database.Tx, hash *chainhash.Hash) (*TreasuryState, error) {
	meta := dbTx.Metadata()
	bucket := meta.Bucket(dbnamespace.TreasuryBucketName)

	v := bucket.Get(hash[:])
	if v == nil {
		return nil, ticketDBError(ErrMissingKey,
			fmt.Sprintf("missing key %v for treasury", hash))
	}

	return deserializeTreasuryState(v)
}

// WriteTreasury inserts the current balance and the future treasury add/spend
// into the database.
func WriteTreasury(dbTx database.Tx, block *dcrutil.Block) error {
	return fmt.Errorf("not yet WriteTreasury")
}
