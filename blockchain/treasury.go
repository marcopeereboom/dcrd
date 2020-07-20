// Copyright (c) 2019 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockchain

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/decred/dcrd/blockchain/stake/v3"
	"github.com/decred/dcrd/blockchain/standalone/v2"
	"github.com/decred/dcrd/blockchain/v3/internal/dbnamespace"
	"github.com/decred/dcrd/chaincfg/chainhash"
	"github.com/decred/dcrd/database/v2"
	"github.com/decred/dcrd/dcrec/secp256k1/v3/schnorr"
	"github.com/decred/dcrd/dcrutil/v3"
	"github.com/decred/dcrd/txscript/v3"
	"github.com/decred/dcrd/wire"
)

const (
	// yesTreasury signifies the treasury agenda should be treated as
	// though it is active.  It is used to increase the readability of the
	// code.
	yesTreasury = true

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

// dbPutTreasuryBalanceWriter inserts a treasury state record into the database.
func dbPutTreasuryBalanceWriter(dbTx database.Tx, hash chainhash.Hash, ts TreasuryState) error {
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

// errDbTreasury wraps a dbFetchTreasuryBalance error.
type errDbTreasury struct {
	err error
}

func (e errDbTreasury) Error() string {
	return e.err.Error()
}

// dbFetchTreasuryBalance uses an existing database transaction to fetch the treasury
// state.
func dbFetchTreasuryBalance(dbTx database.Tx, hash chainhash.Hash) (*TreasuryState, error) {
	meta := dbTx.Metadata()
	bucket := meta.Bucket(dbnamespace.TreasuryBucketName)

	v := bucket.Get(hash[:])
	if v == nil {
		return nil, errDbTreasury{
			err: fmt.Errorf("treasury db missing key: %v", hash),
		}
	}

	return deserializeTreasuryState(v)
}

// dbFetchTreasurySingle wraps dbFetchTreasuryBalance in a view.
func (b *BlockChain) dbFetchTreasurySingle(hash chainhash.Hash) (*TreasuryState, error) {
	var (
		ts  *TreasuryState
		err error
	)
	err = b.db.View(func(dbTx database.Tx) error {
		ts, err = dbFetchTreasuryBalance(dbTx, hash)
		return err
	})
	return ts, err
}

// serializeTSpend serializes the TSpend data for use in the database.
// The format is as follows:
// Block chainhash.Hash (block where TSpend was mined).
func serializeTSpend(blocks []chainhash.Hash) ([]byte, error) {
	serializedData := new(bytes.Buffer)
	err := binary.Write(serializedData, dbnamespace.ByteOrder,
		int64(len(blocks)))
	if err != nil {
		return nil, err
	}
	for _, v := range blocks {
		err := binary.Write(serializedData, dbnamespace.ByteOrder, v[:])
		if err != nil {
			return nil, err
		}
	}
	return serializedData.Bytes(), nil
}

// deserializeTSpend deserializes a binary blob into a chainhash.Hash.
func deserializeTSpend(data []byte) ([]chainhash.Hash, error) {
	buf := bytes.NewReader(data)
	var count int64
	err := binary.Read(buf, dbnamespace.ByteOrder, &count)
	if err != nil {
		return nil, fmt.Errorf("count %v", err)
	}
	hashes := make([]chainhash.Hash, count)
	for i := int64(0); i < count; i++ {
		err := binary.Read(buf, dbnamespace.ByteOrder, &hashes[i])
		if err != nil {
			return nil,
				fmt.Errorf("values read %v error %v", i, err)
		}
	}

	return hashes, nil
}

// dbPutTSpend inserts a treasury tspend record into the database. Note that this call is the low level write to the database. Use dbUpdateTSpend instead.
func dbPutTSpend(dbTx database.Tx, tx chainhash.Hash, blocks []chainhash.Hash) error {
	// Serialize the current treasury state.
	serializedData, err := serializeTSpend(blocks)
	if err != nil {
		return err
	}

	// Store the current treasury state into the database.
	meta := dbTx.Metadata()
	bucket := meta.Bucket(dbnamespace.TreasuryTSpendName)
	return bucket.Put(tx[:], serializedData)
}

type errDbTSpend struct {
	err error
}

func (e errDbTSpend) Error() string {
	return e.err.Error()
}

// dbFetchTSpend uses an existing database transaction to fetch the block hash
// that contains the provided transaction.
func dbFetchTSpend(dbTx database.Tx, tx chainhash.Hash) ([]chainhash.Hash, error) {
	meta := dbTx.Metadata()
	bucket := meta.Bucket(dbnamespace.TreasuryTSpendName)

	v := bucket.Get(tx[:])
	if v == nil {
		return nil, errDbTSpend{
			err: fmt.Errorf("tspend db missing key: %v", tx),
		}
	}

	return deserializeTSpend(v)
}

// dbUpdateTSpend performs a read/modify/write operation on the provided
// transaction hash. It read the record and appends the block hash and then
// writes it back to the database. Note that the append is dumb and does not
// deduplicate. This is ok because in practice a TX cannot appear in the same
// block.
func dbUpdateTSpend(dbTx database.Tx, tx, block chainhash.Hash) error {
	hashes, err := dbFetchTSpend(dbTx, tx)
	if _, ok := err.(errDbTSpend); ok {
		// Record doesn't exist.
	} else if err != nil {
		return err
	}
	hashes = append(hashes, block)
	return dbPutTSpend(dbTx, tx, hashes)
}

// DbFetchTSpend returns the blocks a TSpend was included in.
func (b *BlockChain) DbFetchTSpend(tspend chainhash.Hash) ([]chainhash.Hash, error) {
	var (
		hashes []chainhash.Hash
		err    error
	)
	err = b.db.View(func(dbTx database.Tx) error {
		hashes, err = dbFetchTSpend(dbTx, tspend)
		return err
	})
	return hashes, err
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
	ts, err := dbFetchTreasuryBalance(dbTx, node.parent.hash)
	if err != nil {
		// Since the node.parent.hash does not exist in the treasury db
		// we can safely assume the balance is 0
		return 0, nil
	}

	// Fetch values that need to be added to the treasury balance.
	wts, err := dbFetchTreasuryBalance(dbTx, wantNode.hash)
	if err != nil {
		// Since wantNode does not exist in the treasury db we can
		// safely assume the balance is 0
		return 0, nil
	}

	// Add all TAdd values to the balance. Note that negative Values are
	// TSpend.
	var netValue int64
	for _, v := range wts.Values {
		netValue += v
	}

	return ts.Balance + netValue, nil
}

// WriteTreasury inserts the current balance and the future treasury add/spend
// into the database.
func (b *BlockChain) dbPutTreasuryBalance(dbTx database.Tx, block *dcrutil.Block, node *blockNode) error {
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
	trsyLog.Tracef("dbPutTreasuryBalance: %v start balance %v",
		node.hash.String(), balance)
	for _, v := range msgBlock.STransactions {
		if stake.IsTAdd(v) {
			// This is a TAdd, pull amount out of TxOut[0].  Note
			// that TxOut[1], if it exists, contains the change
			// output. We have to ignore change.
			ts.Values = append(ts.Values, v.TxOut[0].Value)
			trsyLog.Tracef("  dbPutTreasuryBalance: balance TADD "+
				"%v", v.TxOut[0].Value)
		} else if stake.IsTreasuryBase(v) {
			ts.Values = append(ts.Values, v.TxOut[0].Value)
			trsyLog.Tracef("  dbPutTreasuryBalance: balance "+
				"treasury base %v", v.TxOut[0].Value)
		} else if stake.IsTSpend(v) {
			// This is a TSpend, pull values out of block. Skip
			// first TxOut since it is an OP_RETURN.
			for _, vv := range v.TxOut[1:] {
				trsyLog.Tracef("  dbPutTreasuryBalance: "+
					"balance TSPEND %v", -vv.Value)
				ts.Values = append(ts.Values, -vv.Value)
			}
		}
	}

	hash := block.Hash()
	return dbPutTreasuryBalanceWriter(dbTx, *hash, ts)
}

// addTreasuryBucket creates the treasury database if it doesn't exist.
func addTreasuryBucket(db database.DB) error {
	return db.Update(func(dbTx database.Tx) error {
		_, err := dbTx.Metadata().CreateBucketIfNotExists(dbnamespace.TreasuryBucketName)
		return err
	})
}

// addTSpendBucket creates the tspend database if it doesn't exist.
func addTSpendBucket(db database.DB) error {
	return db.Update(func(dbTx database.Tx) error {
		_, err := dbTx.Metadata().CreateBucketIfNotExists(dbnamespace.TreasuryTSpendName)
		return err
	})
}

// treasuryBalanceFailure returns a failure for the TreasuryBalance function.
// It exists because the return values should not be copy pasted.
func treasuryBalanceFailure(err error) (string, int64, int64, []int64, error) {
	return "", 0, 0, []int64{}, err
}

// dbPutTSpend inserts the TSpends that are included in this block to the
// database.
func (b *BlockChain) dbPutTSpend(dbTx database.Tx, block *dcrutil.Block, node *blockNode) error {
	hash := block.Hash()
	msgBlock := block.MsgBlock()
	trsyLog.Tracef("dbPutTSpend: processing block %v", hash)
	for _, v := range msgBlock.STransactions {
		if !stake.IsTSpend(v) {
			continue
		}

		// Store TSpend and the block it was included in.
		txHash := v.TxHash()
		trsyLog.Tracef("  dbPutTSpend: tspend %v", txHash)
		err := dbUpdateTSpend(dbTx, txHash, *hash)
		if err != nil {
			return err
		}
	}

	return nil
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
		ts, err = dbFetchTreasuryBalance(dbTx, *ch)
		return err
	})
	if err != nil {
		return treasuryBalanceFailure(err)
	}

	return *hash, node.height, ts.Balance, ts.Values, nil
}

// verifyTSpendSignature verifies that the provided signature and public key
// were the ones that signed the provided message transactin.
func verifyTSpendSignature(msgTx *wire.MsgTx, signature, pubKey []byte) error {
	// Calculate signature hash.
	sigHash, err := txscript.CalcSignatureHash(nil,
		txscript.SigHashAll, msgTx, 0, nil)
	if err != nil {
		return fmt.Errorf("CalcSignatureHash: %v", err)
	}

	// Lift Signature from bytes.
	sig, err := schnorr.ParseSignature(signature)
	if err != nil {
		return fmt.Errorf("ParseDERSignature: %v", err)
	}

	// Lift public PI key from bytes.
	pk, err := schnorr.ParsePubKey(pubKey)
	if err != nil {
		return fmt.Errorf("ParsePubKey: %v", err)
	}

	// Verify transaction was properly signed.
	if !sig.Verify(sigHash, pk) {
		return fmt.Errorf("Verify failed")
	}

	return nil
}

// checkTSpendExpenditure verifies that the TSpend transaction expenditure is
// within range. This function does not consider TAdds in this block since
// those are not mature. There is a hard requirement that the tspend parameter
// is a proper TSPEND and that this function is only called on a TVI.
//
// This function must be called with the read lock held.
//
// XXX this function needs more thought since the 150% guard rail is not going
// to age well.
func (b *BlockChain) checkTSpendExpenditure(block *dcrutil.Block, prevNode *blockNode, tspends []*dcrutil.Tx) error {
	trsyLog.Tracef("checkTSpendExpenditure: processing block %v (%v)",
		block.Hash(), len(tspends))
	if len(tspends) == 0 {
		// Nothing to do.
		return nil
	}

	// Determine how much this block is spending.
	var wantSpend int64
	for _, v := range tspends {
		// A valid TSPEND always stores the entire amount that the
		// treasury is spending in the first TxIn.
		wantSpend += v.MsgTx().TxIn[0].ValueIn
	}

	// Ensure that we are not depleting treasury.
	var (
		treasuryBalance int64
		err             error
	)
	err = b.db.View(func(dbTx database.Tx) error {
		treasuryBalance, err = b.calculateTreasuryBalance(dbTx, prevNode)
		return err
	})
	if err != nil {
		return err
	}
	// wantSpend is a negative number therefor we use +.
	if treasuryBalance-wantSpend < 0 {
		return fmt.Errorf("treasury balance may not become negative: "+
			"balance %v spend %v", treasuryBalance, wantSpend)
	}
	trsyLog.Tracef("  checkTSpendExpenditure: balance %v spend %v",
		treasuryBalance, wantSpend)

	// This function is pretty naive. It simply iterates through prior
	// blocks one at a time. This is very expensive and may need to be
	// rethought.

	// Calculate the net in and out of the treasury and verify those values
	// are within parameter per policy requirements.
	policyWindow := b.chainParams.TreasuryVoteInterval *
		b.chainParams.TreasuryVoteIntervalMultiplier *
		b.chainParams.TreasuryVoteIntervalPolicy
	var add, spend int64
	node := prevNode
	// XXX We are ignoring CoinbaseMaturity which is incorrect.
	// XXX this may be way too complicated too; we may be able to get away
	// with calculating blockreward*policyWindow. The thing to note is that
	// over time the value will decay and be too restrictive; we need to
	// think about this.
	// The policy window is inclusive.
	for i := uint64(0); i <= policyWindow; i++ {
		ts, err := b.dbFetchTreasurySingle(node.hash)
		if _, ok := err.(errDbTreasury); ok {
			// Record doesn't exist.
			continue
		} else if err != nil {
			return err
		}

		// Range over values.
		for _, v := range ts.Values {
			if v < 0 {
				spend += v
			} else {
				add += v
			}
		}

		node = b.index.lookupNode(&node.parent.hash)
		if node == nil {
			break
		}
	}

	// XXX subtract spend?

	allowedToSpend := add + add/2 // ~150%
	trsyLog.Tracef("  checkTSpendExpenditure: add %v spend %v allowed %v",
		add, spend, allowedToSpend)

	if wantSpend > allowedToSpend {
		return fmt.Errorf("treasury spend greater than allowed %v > %v",
			wantSpend, allowedToSpend)
	}

	return nil
}

// checkTSpendExists verifies that the provided TSpend has not been mined in a
// block on the chain of prevNode.
func (b *BlockChain) checkTSpendExists(block *dcrutil.Block, prevNode *blockNode, tspend *dcrutil.Tx) error {
	hash := tspend.Hash()
	trsyLog.Tracef(" checkTSpendExists: tspend %v", hash)
	blocks, err := b.DbFetchTSpend(*hash)
	if _, ok := err.(errDbTSpend); ok {
		// Record does not exist.
		return nil
	} else if err != nil {
		return err
	}

	// Do fork detection on all blocks.
	for _, v := range blocks {
		// Lookup blockNode.
		// XXX is it ok to use the index here instead of fetching the
		// block?
		node := b.index.LookupNode(&v)
		if node == nil {
			// This should not happen.
			trsyLog.Errorf("  checkTSpendExists: block not found "+
				"%v tspend %v", v, hash)
			continue
		}

		if prevNode.Ancestor(node.height) != node {
			trsyLog.Errorf("  checkTSpendExists: not ancestor "+
				"block %v tspend %v", v, hash)
			continue
		}
		trsyLog.Errorf("  checkTSpendExists: is ancestor "+
			"block %v tspend %v", v, hash)
		return fmt.Errorf("tspend has already been mined on this "+
			"chain %v", hash)
	}

	return nil
}

// getVotes returns yes and no votes for the provided hash.
func getVotes(votes []stake.TreasuryVoteTuple, hash *chainhash.Hash) (yes int, no int) {
	if votes == nil {
		return
	}

	for _, v := range votes {
		if !hash.IsEqual(&v.Hash) {
			continue
		}

		switch v.Vote {
		case stake.TreasuryVoteYes:
			yes++
		case stake.TreasuryVoteNo:
			no++
		default:
			// Can't happen.
			trsyLog.Criticalf("getVotes: invalid vote 0x%v", v.Vote)
		}
	}

	return
}

// TSpendCountVotes returns the number the start block, end block, yes and no
// votes.  This function must only be called on a TVI.  Note that block can be
// incomplete; it only must contain the right height.
func (b *BlockChain) TSpendCountVotes(block *dcrutil.Block, prevNode *blockNode, tspend *dcrutil.Tx) (startBlock, endBlock uint32, yesVotes, noVotes int, err error) {
	trsyLog.Tracef("TSpendCountVotes: processing block %v tspend %v ",
		block.Hash(), tspend.Hash())

	expiry := tspend.MsgTx().Expiry
	startBlock, err = standalone.CalculateTSpendWindowStart(expiry,
		b.chainParams.TreasuryVoteInterval,
		b.chainParams.TreasuryVoteIntervalMultiplier)
	if err != nil {
		return
	}
	endBlock, err = standalone.CalculateTSpendWindowEnd(expiry,
		b.chainParams.TreasuryVoteInterval)
	if err != nil {
		return
	}

	trsyLog.Tracef("  TSpendCountVotes: height %v start %v expiry %v",
		block.Height(), startBlock, endBlock)

	// Ensure tspend is within the window.
	if !standalone.InsideTSpendWindow(block.Height(),
		expiry, b.chainParams.TreasuryVoteInterval,
		b.chainParams.TreasuryVoteIntervalMultiplier) {
		err = fmt.Errorf("tspend outside of window: height %v "+
			"start %v expiry %v", block.Height(), startBlock, expiry)
		return
	}

	// Walk prevNode back to the start of the window and count votes.
	node := prevNode
	for {
		trsyLog.Tracef("  TSpendCountVotes height %v start %v",
			node.height, startBlock)
		if node.height < int64(startBlock) {
			break
		}

		trsyLog.Tracef("  TSpendCountVotes count votes: %v",
			node.hash)

		// Find SSGen and peel out votes.
		var xblock *dcrutil.Block
		xblock, err = b.fetchBlockByNode(node)
		if err != nil {
			// Should not happen.
			return
		}
		for _, v := range xblock.STransactions() {
			votes, err := stake.CheckSSGenVotes(v.MsgTx(),
				yesTreasury)
			if err != nil {
				// Not an SSGEN
				continue
			}

			// Find our vote bits.
			yes, no := getVotes(votes, tspend.Hash())
			yesVotes += yes
			noVotes += no
		}

		node = b.index.lookupNode(&node.parent.hash)
		if node == nil {
			break
		}
	}

	return startBlock, endBlock, yesVotes, noVotes, nil
}

// checkTSpendHasVotes verifies that the provided TSpend has enough votes to be
// included in the provided block. This function must only be called on a TVI.
// Note that block can be incomplete; it only must contain the right height.
// This is needed in the mining path.
func (b *BlockChain) checkTSpendHasVotes(block *dcrutil.Block, prevNode *blockNode, tspend *dcrutil.Tx) error {
	startBlock, endBlock, yesVotes, noVotes, err := b.TSpendCountVotes(block,
		prevNode, tspend)
	if err != nil {
		return err
	}

	// Passing citeria are 20% quorum and 60% yes.
	maxVotes := uint32(b.chainParams.TicketsPerBlock) *
		(endBlock - startBlock)
	quorum := uint64(maxVotes) * b.chainParams.TreasuryVoteQuorumMultiplier /
		b.chainParams.TreasuryVoteQuorumDivisor
	if uint64(yesVotes+noVotes) < quorum {
		return fmt.Errorf("quorum not met: yes %v no %v quorum %v "+
			"max %v", yesVotes, noVotes, quorum, maxVotes)
	}

	requiredVotes := uint64(yesVotes+noVotes) *
		b.chainParams.TreasuryVoteRequiredMultiplier /
		b.chainParams.TreasuryVoteRequiredDivisor
	if uint64(yesVotes) < requiredVotes {
		return fmt.Errorf("not enough yes votes: yes %v no %v "+
			"quorum %v max %v required %v", yesVotes, noVotes,
			quorum, maxVotes, requiredVotes)
	}

	trsyLog.Infof("TSpend %v passed with: yes %v no %v quorum %v "+
		"required %v", tspend.Hash(), yesVotes, noVotes, quorum,
		requiredVotes)

	return nil
}

// CheckTSpendHasVotes exports checkTSpendHasVotes for mining purposes.
func (b *BlockChain) CheckTSpendHasVotes(block *dcrutil.Block, prevNode *blockNode, tspend *dcrutil.Tx) error {
	return b.checkTSpendHasVotes(block, prevNode, tspend)
}
