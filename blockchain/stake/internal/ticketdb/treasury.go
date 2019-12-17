// Copyright (c) 2019 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package ticketdb

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/decred/dcrd/blockchain/stake/v2/internal/dbnamespace"
	"github.com/decred/dcrd/chaincfg/chainhash"
	"github.com/decred/dcrd/database/v2"
)

// serializeTreasuryState serializes the dbnamespace.TreasuryState structure
// for use in the database.
// The format is as follows:
// littleendian.int64(treasury balance as of this block)
// littleendian.int64(length of values arrays)
// []littleendian.int64(all additions and subtractions from treasury in this
//   block)
func serializeTreasuryState(ts dbnamespace.TreasuryState) ([]byte, error) {
	// Just a little sanity testing.
	if ts.Balance < 0 {
		return nil, ticketDBError(ErrTreasurySerialization,
			fmt.Sprintf("invalid treasury balance: %v", ts.Balance))
	}
	if len(ts.Values) > dbnamespace.TreasuryMaxEntriesPerBlock {
		return nil, ticketDBError(ErrTreasurySerialization,
			fmt.Sprintf("invalid treasury values length: %v",
				len(ts.Values)))
	}

	// Serialize TreasuryState.
	serializedData := new(bytes.Buffer)
	err := binary.Write(serializedData, binary.LittleEndian, ts.Balance)
	if err != nil {
		return nil, ticketDBError(ErrTreasurySerialization,
			err.Error())
	}
	err = binary.Write(serializedData, binary.LittleEndian,
		int64(len(ts.Values)))
	if err != nil {
		return nil, ticketDBError(ErrTreasurySerialization,
			err.Error())
	}
	for _, v := range ts.Values {
		err := binary.Write(serializedData, binary.LittleEndian, v)
		if err != nil {
			return nil, ticketDBError(ErrTreasurySerialization,
				err.Error())
		}
	}
	return serializedData.Bytes(), nil
}

// deserializeTreasuryState desrializes a binary blob into a
// dbnamespace.TreasuryState structure.
func deserializeTreasuryState(data []byte) (*dbnamespace.TreasuryState, error) {
	var ts dbnamespace.TreasuryState
	buf := bytes.NewReader(data)
	err := binary.Read(buf, binary.LittleEndian, &ts.Balance)
	if err != nil {
		return nil, ticketDBError(ErrTreasuryDeserialization,
			fmt.Sprintf("balance %v", err))
	}

	var count int64
	err = binary.Read(buf, binary.LittleEndian, &count)
	if err != nil {
		return nil, ticketDBError(ErrTreasuryDeserialization,
			fmt.Sprintf("count %v", err))
	}
	if count > dbnamespace.TreasuryMaxEntriesPerBlock {
		return nil, ticketDBError(ErrTreasuryDeserialization,
			fmt.Sprintf("invalid treasury values length: %v", count))
	}

	ts.Values = make([]int64, count)
	for i := int64(0); i < count; i++ {
		err := binary.Read(buf, binary.LittleEndian, &ts.Values[i])
		if err != nil {
			return nil, ticketDBError(ErrTreasuryDeserialization,
				fmt.Sprintf("values read %v error %v", i, err))
		}
	}

	return &ts, nil
}

// DbPutTreasury inserts a treasury state record into the database.
func DbPutTreasury(dbTx database.Tx, hash chainhash.Hash, ts dbnamespace.TreasuryState) error {
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
func DbFetchTreasury(dbTx database.Tx, hash chainhash.Hash) (*dbnamespace.TreasuryState, error) {
	meta := dbTx.Metadata()
	bucket := meta.Bucket(dbnamespace.TreasuryBucketName)

	v := bucket.Get(hash[:])
	if v == nil {
		return nil, ticketDBError(ErrMissingKey,
			fmt.Sprintf("missing key %v for treasury", hash))
	}

	return deserializeTreasuryState(v)
}
