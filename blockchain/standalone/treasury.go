// Copyright (c) 2020 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package standalone

import "fmt"

// CalculateTSpendExpiry returns the only valid value relative to the next
// block height where the transaction will expire. We add two blocks in the end
// because transaction expiry is inclusive (>=) relative to blockheight.
// We try very hard to use the "natural" types but this is a giant mess.
func CalculateTSpendExpiry(nextBlockHeight int64, tvi, multiplier uint64) uint32 {
	nbh := uint64(nextBlockHeight)
	nextTVI := nbh + ((tvi - (nbh % tvi)) % tvi) // Round up to next TVI
	if nextTVI == nbh {
		nextTVI += tvi // NextBlockHeight cannot be itself.
	}
	maxTVI := nextTVI + tvi*multiplier // Max TVI allowed at this time.
	return uint32(maxTVI + 2)
}

// IsTreasuryVoteInterval returns true if the passed height is on a Treasury
// Vote Interval.
func IsTreasuryVoteInterval(height, tvi uint64) bool {
	return height%tvi == 0
}

// CalculateTSpendWindowStart calculates the start of a treasury voting window
// based on the parameters that are passed. Great care must be taken to ensure
// this function is only called with an expiry that *IS* on a TVI.
func CalculateTSpendWindowStart(expiry uint32, tvi, multiplier uint64) (uint32, error) {
	if !IsTreasuryVoteInterval(uint64(expiry-2), tvi) {
		return 0, fmt.Errorf("CalculateTSpendWindowStart invalid "+
			"expiry: %v", expiry)
	}
	return expiry - uint32(tvi*multiplier) - 2, nil
}

// CalculateTSpendWindowEnd calculates the end of a treasury voting window
// based on the parameters that are passed. Great care must be taken to ensure
// this function is only called with an expiry that *IS* on a TVI.
func CalculateTSpendWindowEnd(expiry uint32, tvi uint64) (uint32, error) {
	if !IsTreasuryVoteInterval(uint64(expiry-2), tvi) {
		return 0, fmt.Errorf("CalculateTSpendWindowEnd invalid "+
			"expiry: %v", expiry)
	}
	return expiry - 2, nil
}

// InsideTSpendWindow returns true if the provided block height is inside the
// treasury vote window of the provided expiry.
// This function should only be called with an expiry that is on a TVI.
func InsideTSpendWindow(blockHeight int64, expiry uint32, tvi, multiplier uint64) bool {
	s, err := CalculateTSpendWindowStart(expiry, tvi, multiplier)
	if err != nil {
		return false
	}
	e, err := CalculateTSpendWindowEnd(expiry, tvi)
	if err != nil {
		return false
	}
	return uint32(blockHeight) >= s && uint32(blockHeight) <= e
}
