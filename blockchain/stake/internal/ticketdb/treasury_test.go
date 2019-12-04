// Copyright (c) 2016-2019 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package ticketdb

import (
	"reflect"
	"testing"

	"github.com/decred/dcrd/blockchain/stake/v2/internal/dbnamespace"
)

func TestSerializeTreasuryState(t *testing.T) {
	ts := dbnamespace.TreasuryState{
		Balance: 100,
		Values:  []int64{1, 2, 3, -3, -2},
	}

	b, err := serializeTreasuryState(ts)
	if err != nil {
		t.Fatal(err)
	}
	tso, err := deserializeTreasuryState(b)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ts, *tso) {
		t.Fatalf("got %v expected %v", *tso, ts)
	}
}
