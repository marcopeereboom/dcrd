// Copyright (c) 2015-2016 The btcsuite developers
// Copyright (c) 2015-2019 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package txscript

import (
	"fmt"
	"math/big"
)

const (
	// LockTimeThreshold is the number below which a lock time is
	// interpreted to be a block number.  Since an average of one block
	// is generated per 10 minutes, this allows blocks for about 9,512
	// years.
	LockTimeThreshold = 5e8 // Tue Nov 5 00:53:20 1985 UTC

	// maxUniqueCoinbaseNullDataSize is the maximum number of bytes allowed
	// in the pushed data output of the coinbase output that is used to
	// ensure the coinbase has a unique hash.
	//
	// Deprecated: This will be removed in the next major version bump.
	maxUniqueCoinbaseNullDataSize = 256
)

// ExtractCoinbaseNullData ensures the passed script is a nulldata script as
// required by the consensus rules for the coinbase output that is used to
// ensure the coinbase has a unique hash and returns the data it pushes.
//
// NOTE: This function is only valid for version 0 scripts.  Since the function
// does not accept a script version, the results are undefined for other script
// versions.
//
// Deprecated: This will be removed in the next major version bump.
func ExtractCoinbaseNullData(pkScript []byte) ([]byte, error) {
	// The nulldata in the coinbase must be a single OP_RETURN followed by a
	// data push up to maxUniqueCoinbaseNullDataSize bytes.
	//
	// NOTE: This is intentionally not using GetScriptClass and the related
	// functions because those are specifically for standardness checks which
	// can change over time and this function is specifically intended to be
	// used by the consensus rules.
	//
	// Also of note is that technically normal nulldata scripts support encoding
	// numbers via small opcodes, however the consensus rules require the block
	// height to be encoded as a 4-byte little-endian uint32 pushed via a normal
	// data push, as opposed to using the normal number handling semantics of
	// scripts, so this is specialized to accommodate that.
	const scriptVersion = 0
	if len(pkScript) == 1 && pkScript[0] == OP_RETURN {
		return nil, nil
	}
	if len(pkScript) > 1 && pkScript[0] == OP_RETURN {
		tokenizer := MakeScriptTokenizer(scriptVersion, pkScript[1:])
		if tokenizer.Next() && tokenizer.Done() &&
			tokenizer.Opcode() <= OP_PUSHDATA4 &&
			len(tokenizer.Data()) <= maxUniqueCoinbaseNullDataSize {

			return tokenizer.Data(), nil
		}
	}

	str := fmt.Sprintf("script %x is not well-formed coinbase nulldata",
		pkScript)
	return nil, scriptError(ErrMalformedCoinbaseNullData, str)
}

// CheckSignatureEncoding returns whether or not the passed signature adheres to
// the strict encoding requirements.
func CheckSignatureEncoding(sig []byte) error {
	// The format of a DER encoded signature is as follows:
	//
	// 0x30 <total length> 0x02 <length of R> <R> 0x02 <length of S> <S>
	//   - 0x30 is the ASN.1 identifier for a sequence
	//   - Total length is 1 byte and specifies length of all remaining data
	//   - 0x02 is the ASN.1 identifier that specifies an integer follows
	//   - Length of R is 1 byte and specifies how many bytes R occupies
	//   - R is the arbitrary length big-endian encoded number which
	//     represents the R value of the signature.  DER encoding dictates
	//     that the value must be encoded using the minimum possible number
	//     of bytes.  This implies the first byte can only be null if the
	//     highest bit of the next byte is set in order to prevent it from
	//     being interpreted as a negative number.
	//   - 0x02 is once again the ASN.1 integer identifier
	//   - Length of S is 1 byte and specifies how many bytes S occupies
	//   - S is the arbitrary length big-endian encoded number which
	//     represents the S value of the signature.  The encoding rules are
	//     identical as those for R.
	const (
		asn1SequenceID = 0x30
		asn1IntegerID  = 0x02

		// minSigLen is the minimum length of a DER encoded signature and is
		// when both R and S are 1 byte each.
		//
		// 0x30 + <1-byte> + 0x02 + 0x01 + <byte> + 0x2 + 0x01 + <byte>
		minSigLen = 8

		// maxSigLen is the maximum length of a DER encoded signature and is
		// when both R and S are 33 bytes each.  It is 33 bytes because a
		// 256-bit integer requires 32 bytes and an additional leading null byte
		// might required if the high bit is set in the value.
		//
		// 0x30 + <1-byte> + 0x02 + 0x21 + <33 bytes> + 0x2 + 0x21 + <33 bytes>
		maxSigLen = 72

		// sequenceOffset is the byte offset within the signature of the
		// expected ASN.1 sequence identifier.
		sequenceOffset = 0

		// dataLenOffset is the byte offset within the signature of the expected
		// total length of all remaining data in the signature.
		dataLenOffset = 1

		// rTypeOffset is the byte offset within the signature of the ASN.1
		// identifier for R and is expected to indicate an ASN.1 integer.
		rTypeOffset = 2

		// rLenOffset is the byte offset within the signature of the length of
		// R.
		rLenOffset = 3

		// rOffset is the byte offset within the signature of R.
		rOffset = 4
	)

	// The signature must adhere to the minimum and maximum allowed length.
	sigLen := len(sig)
	if sigLen < minSigLen {
		str := fmt.Sprintf("malformed signature: too short: %d < %d", sigLen,
			minSigLen)
		return scriptError(ErrSigTooShort, str)
	}
	if sigLen > maxSigLen {
		str := fmt.Sprintf("malformed signature: too long: %d > %d", sigLen,
			maxSigLen)
		return scriptError(ErrSigTooLong, str)
	}

	// The signature must start with the ASN.1 sequence identifier.
	if sig[sequenceOffset] != asn1SequenceID {
		str := fmt.Sprintf("malformed signature: format has wrong type: %#x",
			sig[sequenceOffset])
		return scriptError(ErrSigInvalidSeqID, str)
	}

	// The signature must indicate the correct amount of data for all elements
	// related to R and S.
	if int(sig[dataLenOffset]) != sigLen-2 {
		str := fmt.Sprintf("malformed signature: bad length: %d != %d",
			sig[dataLenOffset], sigLen-2)
		return scriptError(ErrSigInvalidDataLen, str)
	}

	// Calculate the offsets of the elements related to S and ensure S is inside
	// the signature.
	//
	// rLen specifies the length of the big-endian encoded number which
	// represents the R value of the signature.
	//
	// sTypeOffset is the offset of the ASN.1 identifier for S and, like its R
	// counterpart, is expected to indicate an ASN.1 integer.
	//
	// sLenOffset and sOffset are the byte offsets within the signature of the
	// length of S and S itself, respectively.
	rLen := int(sig[rLenOffset])
	sTypeOffset := rOffset + rLen
	sLenOffset := sTypeOffset + 1
	if sTypeOffset >= sigLen {
		str := "malformed signature: S type indicator missing"
		return scriptError(ErrSigMissingSTypeID, str)
	}
	if sLenOffset >= sigLen {
		str := "malformed signature: S length missing"
		return scriptError(ErrSigMissingSLen, str)
	}

	// The lengths of R and S must match the overall length of the signature.
	//
	// sLen specifies the length of the big-endian encoded number which
	// represents the S value of the signature.
	sOffset := sLenOffset + 1
	sLen := int(sig[sLenOffset])
	if sOffset+sLen != sigLen {
		str := "malformed signature: invalid S length"
		return scriptError(ErrSigInvalidSLen, str)
	}

	// R elements must be ASN.1 integers.
	if sig[rTypeOffset] != asn1IntegerID {
		str := fmt.Sprintf("malformed signature: R integer marker: %#x != %#x",
			sig[rTypeOffset], asn1IntegerID)
		return scriptError(ErrSigInvalidRIntID, str)
	}

	// Zero-length integers are not allowed for R.
	if rLen == 0 {
		str := "malformed signature: R length is zero"
		return scriptError(ErrSigZeroRLen, str)
	}

	// R must not be negative.
	if sig[rOffset]&0x80 != 0 {
		str := "malformed signature: R is negative"
		return scriptError(ErrSigNegativeR, str)
	}

	// Null bytes at the start of R are not allowed, unless R would otherwise be
	// interpreted as a negative number.
	if rLen > 1 && sig[rOffset] == 0x00 && sig[rOffset+1]&0x80 == 0 {
		str := "malformed signature: R value has too much padding"
		return scriptError(ErrSigTooMuchRPadding, str)
	}

	// S elements must be ASN.1 integers.
	if sig[sTypeOffset] != asn1IntegerID {
		str := fmt.Sprintf("malformed signature: S integer marker: %#x != %#x",
			sig[sTypeOffset], asn1IntegerID)
		return scriptError(ErrSigInvalidSIntID, str)
	}

	// Zero-length integers are not allowed for S.
	if sLen == 0 {
		str := "malformed signature: S length is zero"
		return scriptError(ErrSigZeroSLen, str)
	}

	// S must not be negative.
	if sig[sOffset]&0x80 != 0 {
		str := "malformed signature: S is negative"
		return scriptError(ErrSigNegativeS, str)
	}

	// Null bytes at the start of S are not allowed, unless S would otherwise be
	// interpreted as a negative number.
	if sLen > 1 && sig[sOffset] == 0x00 && sig[sOffset+1]&0x80 == 0 {
		str := "malformed signature: S value has too much padding"
		return scriptError(ErrSigTooMuchSPadding, str)
	}

	// Verify the S value is <= half the order of the curve.  This check is done
	// because when it is higher, the complement modulo the order can be used
	// instead which is a shorter encoding by 1 byte.
	sValue := new(big.Int).SetBytes(sig[sOffset : sOffset+sLen])
	if sValue.Cmp(halfOrder) > 0 {
		return scriptError(ErrSigHighS, "signature is not canonical due to "+
			"unnecessarily high S value")
	}

	return nil
}

// IsStrictSignatureEncoding returns false if the passed signature does not
// adhere to the strict encoding requirements.
func IsStrictSignatureEncoding(signature []byte) bool {
	return CheckSignatureEncoding(signature) == nil
}

// CheckPubKeyEncoding returns an error if the passed public key does not
// adhere to the strict encoding requirements.
func CheckPubKeyEncoding(pubKey []byte) error {
	if !isStrictPubKeyEncoding(pubKey) {
		return scriptError(ErrPubKeyType, "unsupported public key type")
	}
	return nil
}

// CheckHashTypeEncoding returns whether or not the passed hashtype adheres to
// the strict encoding requirements.
func CheckHashTypeEncoding(hashType SigHashType) error {
	sigHashType := hashType & ^SigHashAnyOneCanPay
	if sigHashType < SigHashAll || sigHashType > SigHashSingle {
		str := fmt.Sprintf("invalid hash type 0x%x", hashType)
		return scriptError(ErrInvalidSigHashType, str)
	}
	return nil
}

// IsStrictCompressedPubKeyEncoding returns whether or not the passed public
// key adheres to the strict compressed encoding requirements.
func IsStrictCompressedPubKeyEncoding(pubKey []byte) bool {
	if len(pubKey) == 33 && (pubKey[0] == 0x02 || pubKey[0] == 0x03) {
		// Compressed
		return true
	}
	return false
}

// IsStrictNullData returns wether or not the passed data is an OP_RETURN
// followed by specified length data push. It explicitely verifies that the
// opcode is identical to the expected length. This function does not support
// checks for data > 75 bytes.
func IsStrictNullData(scriptVersion uint16, script []byte, expectedLength int) bool {
	// The only currently supported script version is 0.
	if scriptVersion != 0 {
		return false
	}

	// The script can't possibly be a null data script if it doesn't start
	// with OP_RETURN.  Fail fast to avoid more work below.
	if len(script) < 2 || script[0] != OP_RETURN {
		return false
	}

	// OP_RETURN followed by data push of the expect size.
	tokenizer := MakeScriptTokenizer(scriptVersion, script[1:])
	return tokenizer.Next() && tokenizer.Done() &&
		(isSmallInt(tokenizer.Opcode()) || tokenizer.Opcode() <= OP_DATA_75) &&
		len(tokenizer.Data()) == expectedLength &&
		expectedLength == int(tokenizer.Opcode())
}

// IsScriptHashScript returns whether or not the passed script is a standard
// pay-to-script-hash script.
func IsScriptHashScript(script []byte) bool {
	return extractScriptHash(script) != nil
}

// IsPubKeyHashScript returns whether or not the passed script is a standard
// pay-to-pubkey-hash script.
func IsPubKeyHashScript(script []byte) bool {
	return extractPubKeyHash(script) != nil
}

// IsStakeChangeScript returns whether or not the passed script is a supported
// stake change script.
//
// NOTE: This function is only valid for version 0 scripts.  It will always
// return false for other script versions.
func IsStakeChangeScript(scriptVersion uint16, script []byte) bool {
	// The only currently supported script version is 0.
	if scriptVersion != 0 {
		return false
	}

	// The only supported stake change scripts are pay-to-pubkey-hash and
	// pay-to-script-hash tagged with the stake submission opcode.
	const stakeOpcode = OP_SSTXCHANGE
	return extractStakePubKeyHash(script, stakeOpcode) != nil ||
		extractStakeScriptHash(script, stakeOpcode) != nil
}
