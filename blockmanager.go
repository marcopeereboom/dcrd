// Copyright (c) 2013-2016 The btcsuite developers
// Copyright (c) 2015-2020 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"container/list"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/decred/dcrd/blockchain/standalone/v2"
	"github.com/decred/dcrd/blockchain/v4"
	"github.com/decred/dcrd/chaincfg/chainhash"
	"github.com/decred/dcrd/chaincfg/v3"
	"github.com/decred/dcrd/database/v2"
	"github.com/decred/dcrd/dcrutil/v3"
	"github.com/decred/dcrd/internal/mempool"
	"github.com/decred/dcrd/internal/progresslog"
	"github.com/decred/dcrd/internal/rpcserver"
	peerpkg "github.com/decred/dcrd/peer/v2"
	"github.com/decred/dcrd/txscript/v3"
	"github.com/decred/dcrd/wire"
)

const (
	// minInFlightBlocks is the minimum number of blocks that should be
	// in the request queue for headers-first mode before requesting
	// more.
	minInFlightBlocks = 10

	// maxOrphanBlocks is the maximum number of orphan blocks that can be
	// queued.
	maxOrphanBlocks = 500

	// maxRejectedTxns is the maximum number of rejected transactions
	// hashes to store in memory.
	maxRejectedTxns = 1000

	// maxRequestedBlocks is the maximum number of requested block
	// hashes to store in memory.
	maxRequestedBlocks = wire.MaxInvPerMsg

	// maxRequestedTxns is the maximum number of requested transactions
	// hashes to store in memory.
	maxRequestedTxns = wire.MaxInvPerMsg
)

// zeroHash is the zero value hash (all zeros).  It is defined as a convenience.
var zeroHash chainhash.Hash

// newPeerMsg signifies a newly connected peer to the block handler.
type newPeerMsg struct {
	peer *peerpkg.Peer
}

// blockMsg packages a Decred block message and the peer it came from together
// so the block handler has access to that information.
type blockMsg struct {
	block *dcrutil.Block
	peer  *peerpkg.Peer
	reply chan struct{}
}

// invMsg packages a Decred inv message and the peer it came from together
// so the block handler has access to that information.
type invMsg struct {
	inv  *wire.MsgInv
	peer *peerpkg.Peer
}

// headersMsg packages a Decred headers message and the peer it came from
// together so the block handler has access to that information.
type headersMsg struct {
	headers *wire.MsgHeaders
	peer    *peerpkg.Peer
}

// notFoundMsg packages a Decred notfound message and the peer it came from
// together so the block handler has access to that information.
type notFoundMsg struct {
	notFound *wire.MsgNotFound
	peer     *peerpkg.Peer
}

// donePeerMsg signifies a newly disconnected peer to the block handler.
type donePeerMsg struct {
	peer *peerpkg.Peer
}

// txMsg packages a Decred tx message and the peer it came from together
// so the block handler has access to that information.
type txMsg struct {
	tx    *dcrutil.Tx
	peer  *peerpkg.Peer
	reply chan struct{}
}

// getSyncPeerMsg is a message type to be sent across the message channel for
// retrieving the current sync peer.
type getSyncPeerMsg struct {
	reply chan int32
}

// requestFromPeerMsg is a message type to be sent across the message channel
// for requesting either blocks or transactions from a given peer. It routes
// this through the block manager so the block manager doesn't ban the peer
// when it sends this information back.
type requestFromPeerMsg struct {
	peer   *peerpkg.Peer
	blocks []*chainhash.Hash
	txs    []*chainhash.Hash
	reply  chan requestFromPeerResponse
}

// requestFromPeerResponse is a response sent to the reply channel of a
// requestFromPeerMsg query.
type requestFromPeerResponse struct {
	err error
}

// tipGenerationResponse is a response sent to the reply channel of a
// tipGenerationMsg query.
type tipGenerationResponse struct {
	hashes []chainhash.Hash
	err    error
}

// tipGenerationMsg is a message type to be sent across the message
// channel for requesting the required the entire generation of a
// block node.
type tipGenerationMsg struct {
	reply chan tipGenerationResponse
}

// forceReorganizationResponse is a response sent to the reply channel of a
// forceReorganizationMsg query.
type forceReorganizationResponse struct {
	err error
}

// forceReorganizationMsg is a message type to be sent across the message
// channel for requesting that the block on head be reorganized to one of its
// adjacent orphans.
type forceReorganizationMsg struct {
	formerBest chainhash.Hash
	newBest    chainhash.Hash
	reply      chan forceReorganizationResponse
}

// processBlockResponse is a response sent to the reply channel of a
// processBlockMsg.
type processBlockResponse struct {
	forkLen  int64
	isOrphan bool
	err      error
}

// processBlockMsg is a message type to be sent across the message channel
// for requested a block is processed.  Note this call differs from blockMsg
// above in that blockMsg is intended for blocks that came from peers and have
// extra handling whereas this message essentially is just a concurrent safe
// way to call ProcessBlock on the internal block chain instance.
type processBlockMsg struct {
	block *dcrutil.Block
	flags blockchain.BehaviorFlags
	reply chan processBlockResponse
}

// processTransactionResponse is a response sent to the reply channel of a
// processTransactionMsg.
type processTransactionResponse struct {
	acceptedTxs []*dcrutil.Tx
	err         error
}

// processTransactionMsg is a message type to be sent across the message
// channel for requesting a transaction to be processed through the block
// manager.
type processTransactionMsg struct {
	tx            *dcrutil.Tx
	allowOrphans  bool
	rateLimit     bool
	allowHighFees bool
	tag           mempool.Tag
	reply         chan processTransactionResponse
}

// headerNode is used as a node in a list of headers that are linked together
// between checkpoints.
type headerNode struct {
	height int64
	hash   *chainhash.Hash
}

// peerNotifier provides an interface for server peer notifications.
type peerNotifier interface {
	// AnnounceNewTransactions generates and relays inventory vectors and
	// notifies websocket clients of the passed transactions.
	AnnounceNewTransactions(txns []*dcrutil.Tx)

	// UpdatePeerHeights updates the heights of all peers who have
	// announced the latest connected main chain block, or a recognized orphan.
	UpdatePeerHeights(latestBlkHash *chainhash.Hash, latestHeight int64, updateSource *peerpkg.Peer)
}

// blockManangerConfig is a configuration struct for a blockManager.
type blockManagerConfig struct {
	PeerNotifier peerNotifier

	// The following fields are for accessing the chain and its configuration.
	Chain        *blockchain.BlockChain
	ChainParams  *chaincfg.Params
	SubsidyCache *standalone.SubsidyCache

	// SigCache defines the signature cache to use.
	SigCache *txscript.SigCache

	// The following field provides access to the mempool.
	TxMemPool *mempool.TxPool

	// RpcServer returns an instance of an RPC server to use for notifications.
	// It may return nil if there is no active RPC server.
	RpcServer func() *rpcserver.Server

	// DisableCheckpoints indicates whether or not the block manager should make
	// use of checkpoints.
	DisableCheckpoints bool

	// NoMiningStateSync indicates whether or not the block manager should
	// perform an initial mining state synchronization with peers once they are
	// believed to be fully synced.
	NoMiningStateSync bool

	// MaxPeers specifies the maximum number of peers the server is expected to
	// be connected with.  It is primarily used as a hint for more efficient
	// synchronization.
	MaxPeers int

	// MaxOrphanTxs specifies the maximum number of orphan transactions the
	// transaction pool associated with the server supports.
	MaxOrphanTxs int
}

// peerSyncState stores additional information that the blockManager tracks
// about a peer.
type peerSyncState struct {
	syncCandidate   bool
	requestedTxns   map[chainhash.Hash]struct{}
	requestedBlocks map[chainhash.Hash]struct{}
}

// orphanBlock represents a block for which the parent is not yet available.  It
// is a normal block plus an expiration time to prevent caching the orphan
// forever.
type orphanBlock struct {
	block      *dcrutil.Block
	expiration time.Time
}

// blockManager provides a concurrency safe block manager for handling all
// incoming blocks.
type blockManager struct {
	cfg             *blockManagerConfig
	started         int32
	shutdown        int32
	rejectedTxns    map[chainhash.Hash]struct{}
	requestedTxns   map[chainhash.Hash]struct{}
	requestedBlocks map[chainhash.Hash]struct{}
	progressLogger  *progresslog.BlockLogger
	syncPeer        *peerpkg.Peer
	msgChan         chan interface{}
	wg              sync.WaitGroup
	quit            chan struct{}
	peerStates      map[*peerpkg.Peer]*peerSyncState

	// The following fields are used for headers-first mode.
	headersFirstMode bool
	headerList       *list.List
	startHeader      *list.Element
	nextCheckpoint   *chaincfg.Checkpoint

	// These fields are related to handling of orphan blocks.  They are
	// protected by the orphan lock.
	orphanLock   sync.RWMutex
	orphans      map[chainhash.Hash]*orphanBlock
	prevOrphans  map[chainhash.Hash][]*orphanBlock
	oldestOrphan *orphanBlock

	// The following fields are used to track the height being synced to from
	// peers.
	syncHeightMtx sync.Mutex
	syncHeight    int64

	// The following fields are used to track whether or not the manager
	// believes it is fully synced to the network.
	isCurrentMtx sync.RWMutex
	isCurrent    bool
}

// resetHeaderState sets the headers-first mode state to values appropriate for
// syncing from a new peer.
func (b *blockManager) resetHeaderState(newestHash *chainhash.Hash, newestHeight int64) {
	b.headersFirstMode = false
	b.headerList.Init()
	b.startHeader = nil

	// When there is a next checkpoint, add an entry for the latest known
	// block into the header pool.  This allows the next downloaded header
	// to prove it links to the chain properly.
	if b.nextCheckpoint != nil {
		node := headerNode{height: newestHeight, hash: newestHash}
		b.headerList.PushBack(&node)
	}
}

// SyncHeight returns latest known block being synced to.
func (b *blockManager) SyncHeight() int64 {
	b.syncHeightMtx.Lock()
	defer b.syncHeightMtx.Unlock()
	return b.syncHeight
}

// findNextHeaderCheckpoint returns the next checkpoint after the passed height.
// It returns nil when there is not one either because the height is already
// later than the final checkpoint or some other reason such as disabled
// checkpoints.
func (b *blockManager) findNextHeaderCheckpoint(height int64) *chaincfg.Checkpoint {
	checkpoints := b.cfg.Chain.Checkpoints()
	if len(checkpoints) == 0 {
		return nil
	}

	// There is no next checkpoint if the height is already after the final
	// checkpoint.
	finalCheckpoint := &checkpoints[len(checkpoints)-1]
	if height >= finalCheckpoint.Height {
		return nil
	}

	// Find the next checkpoint.
	nextCheckpoint := finalCheckpoint
	for i := len(checkpoints) - 2; i >= 0; i-- {
		if height >= checkpoints[i].Height {
			break
		}
		nextCheckpoint = &checkpoints[i]
	}
	return nextCheckpoint
}

// chainBlockLocatorToHashes converts a block locator from chain to a slice
// of hashes.
func chainBlockLocatorToHashes(locator blockchain.BlockLocator) []chainhash.Hash {
	if len(locator) == 0 {
		return nil
	}

	result := make([]chainhash.Hash, 0, len(locator))
	for _, hash := range locator {
		result = append(result, *hash)
	}
	return result
}

// startSync will choose the best peer among the available candidate peers to
// download/sync the blockchain from.  When syncing is already running, it
// simply returns.  It also examines the candidates for any which are no longer
// candidates and removes them as needed.
func (b *blockManager) startSync() {
	// Return now if we're already syncing.
	if b.syncPeer != nil {
		return
	}

	best := b.cfg.Chain.BestSnapshot()
	var bestPeer *peerpkg.Peer
	for peer, state := range b.peerStates {
		if !state.syncCandidate {
			continue
		}

		// Remove sync candidate peers that are no longer candidates due
		// to passing their latest known block.  NOTE: The < is
		// intentional as opposed to <=.  While technically the peer
		// doesn't have a later block when it's equal, it will likely
		// have one soon so it is a reasonable choice.  It also allows
		// the case where both are at 0 such as during regression test.
		if peer.LastBlock() < best.Height {
			state.syncCandidate = false
			continue
		}

		// the best sync candidate is the most updated peer
		if bestPeer == nil {
			bestPeer = peer
		}
		if bestPeer.LastBlock() < peer.LastBlock() {
			bestPeer = peer
		}
	}

	// Update the state of whether or not the manager believes the chain is
	// fully synced to whatever the chain believes when there is no candidate
	// for a sync peer.
	if bestPeer == nil {
		b.isCurrentMtx.Lock()
		b.isCurrent = b.cfg.Chain.IsCurrent()
		b.isCurrentMtx.Unlock()
	}

	// Start syncing from the best peer if one was selected.
	if bestPeer != nil {
		// Clear the requestedBlocks if the sync peer changes, otherwise
		// we may ignore blocks we need that the last sync peer failed
		// to send.
		b.requestedBlocks = make(map[chainhash.Hash]struct{})

		blkLocator, err := b.cfg.Chain.LatestBlockLocator()
		if err != nil {
			bmgrLog.Errorf("Failed to get block locator for the "+
				"latest block: %v", err)
			return
		}
		locator := chainBlockLocatorToHashes(blkLocator)

		bmgrLog.Infof("Syncing to block height %d from peer %v",
			bestPeer.LastBlock(), bestPeer.Addr())

		// The chain is not synced whenever the current best height is less than
		// the height to sync to.
		if best.Height < bestPeer.LastBlock() {
			b.isCurrentMtx.Lock()
			b.isCurrent = false
			b.isCurrentMtx.Unlock()
		}

		// When the current height is less than a known checkpoint we
		// can use block headers to learn about which blocks comprise
		// the chain up to the checkpoint and perform less validation
		// for them.  This is possible since each header contains the
		// hash of the previous header and a merkle root.  Therefore if
		// we validate all of the received headers link together
		// properly and the checkpoint hashes match, we can be sure the
		// hashes for the blocks in between are accurate.  Further, once
		// the full blocks are downloaded, the merkle root is computed
		// and compared against the value in the header which proves the
		// full block hasn't been tampered with.
		//
		// Once we have passed the final checkpoint, or checkpoints are
		// disabled, use standard inv messages learn about the blocks
		// and fully validate them.  Finally, regression test mode does
		// not support the headers-first approach so do normal block
		// downloads when in regression test mode.
		if b.nextCheckpoint != nil &&
			best.Height < b.nextCheckpoint.Height &&
			!b.cfg.DisableCheckpoints {

			err := bestPeer.PushGetHeadersMsg(locator, b.nextCheckpoint.Hash)
			if err != nil {
				bmgrLog.Errorf("Failed to push getheadermsg for the "+
					"latest blocks: %v", err)
				return
			}
			b.headersFirstMode = true
			bmgrLog.Infof("Downloading headers for blocks %d to "+
				"%d from peer %s", best.Height+1,
				b.nextCheckpoint.Height, bestPeer.Addr())
		} else {
			err := bestPeer.PushGetBlocksMsg(locator, &zeroHash)
			if err != nil {
				bmgrLog.Errorf("Failed to push getblocksmsg for the "+
					"latest blocks: %v", err)
				return
			}
		}
		b.syncPeer = bestPeer
		b.syncHeightMtx.Lock()
		b.syncHeight = bestPeer.LastBlock()
		b.syncHeightMtx.Unlock()
	} else {
		bmgrLog.Warnf("No sync peer candidates available")
	}
}

// isSyncCandidate returns whether or not the peer is a candidate to consider
// syncing from.
func (b *blockManager) isSyncCandidate(peer *peerpkg.Peer) bool {
	// The peer is not a candidate for sync if it's not a full node.
	return peer.Services()&wire.SFNodeNetwork == wire.SFNodeNetwork
}

// syncMiningStateAfterSync polls the blockManager for the current sync
// state; if the manager is synced, it executes a call to the peer to
// sync the mining state to the network.
func (b *blockManager) syncMiningStateAfterSync(peer *peerpkg.Peer) {
	go func() {
		for {
			select {
			case <-time.After(3 * time.Second):
			case <-b.quit:
				return
			}

			if !peer.Connected() {
				return
			}
			if !b.IsCurrent() {
				continue
			}

			pver := peer.ProtocolVersion()
			var msg wire.Message

			switch {
			case pver < wire.InitStateVersion:
				// Before protocol version InitStateVersion
				// nodes use the GetMiningState/MiningState
				// pair of p2p messages.
				msg = wire.NewMsgGetMiningState()

			default:
				// On the most recent protocol version, nodes
				// use the GetInitState/InitState pair of p2p
				// messages.
				m := wire.NewMsgGetInitState()
				err := m.AddTypes(wire.InitStateHeadBlocks,
					wire.InitStateHeadBlockVotes,
					wire.InitStateTSpends)
				if err != nil {
					bmgrLog.Errorf("Unexpected error while "+
						"building getinitstate msg: %v",
						err)
					return
				}
				msg = m
			}

			peer.QueueMessage(msg, nil)
			return
		}
	}()
}

// handleNewPeerMsg deals with new peers that have signalled they may
// be considered as a sync peer (they have already successfully negotiated).  It
// also starts syncing if needed.  It is invoked from the syncHandler goroutine.
func (b *blockManager) handleNewPeerMsg(peer *peerpkg.Peer) {
	// Ignore if in the process of shutting down.
	if atomic.LoadInt32(&b.shutdown) != 0 {
		return
	}

	bmgrLog.Infof("New valid peer %s (%s)", peer, peer.UserAgent())

	// Initialize the peer state
	isSyncCandidate := b.isSyncCandidate(peer)
	b.peerStates[peer] = &peerSyncState{
		syncCandidate:   isSyncCandidate,
		requestedTxns:   make(map[chainhash.Hash]struct{}),
		requestedBlocks: make(map[chainhash.Hash]struct{}),
	}

	// Start syncing by choosing the best candidate if needed.
	if isSyncCandidate && b.syncPeer == nil {
		b.startSync()
	}

	// Grab the mining state from this peer once synced when enabled.
	if !b.cfg.NoMiningStateSync {
		b.syncMiningStateAfterSync(peer)
	}
}

// handleDonePeerMsg deals with peers that have signalled they are done.  It
// removes the peer as a candidate for syncing and in the case where it was
// the current sync peer, attempts to select a new best peer to sync from.  It
// is invoked from the syncHandler goroutine.
func (b *blockManager) handleDonePeerMsg(peer *peerpkg.Peer) {
	state, exists := b.peerStates[peer]
	if !exists {
		bmgrLog.Warnf("Received done peer message for unknown peer %s", peer)
		return
	}

	// Remove the peer from the list of candidate peers.
	delete(b.peerStates, peer)

	// Remove requested transactions from the global map so that they will
	// be fetched from elsewhere next time we get an inv.
	for txHash := range state.requestedTxns {
		delete(b.requestedTxns, txHash)
	}

	// Remove requested blocks from the global map so that they will be
	// fetched from elsewhere next time we get an inv.
	// TODO(oga) we could possibly here check which peers have these blocks
	// and request them now to speed things up a little.
	for blockHash := range state.requestedBlocks {
		delete(b.requestedBlocks, blockHash)
	}

	// Attempt to find a new peer to sync from if the quitting peer is the
	// sync peer.  Also, reset the headers-first state if in headers-first
	// mode so
	if b.syncPeer == peer {
		b.syncPeer = nil
		if b.headersFirstMode {
			best := b.cfg.Chain.BestSnapshot()
			b.resetHeaderState(&best.Hash, best.Height)
		}
		b.startSync()
	}
}

// errToWireRejectCode determines the wire rejection code and description for a
// given error. This function can convert some select blockchain and mempool
// error types to the historical rejection codes used on the p2p wire protocol.
func errToWireRejectCode(err error) (wire.RejectCode, string) {
	// The default reason to reject a transaction/block is due to it being
	// invalid somehow.
	code := wire.RejectInvalid
	var reason string

	// Convert recognized errors to a reject code.
	switch {
	// Rejected due to duplicate.
	case errors.Is(err, blockchain.ErrDuplicateBlock):
		code = wire.RejectDuplicate
		reason = err.Error()

	// Rejected due to obsolete version.
	case errors.Is(err, blockchain.ErrBlockVersionTooOld):
		code = wire.RejectObsolete
		reason = err.Error()

	// Rejected due to checkpoint.
	case errors.Is(err, blockchain.ErrCheckpointTimeTooOld),
		errors.Is(err, blockchain.ErrDifficultyTooLow),
		errors.Is(err, blockchain.ErrBadCheckpoint),
		errors.Is(err, blockchain.ErrForkTooOld):
		code = wire.RejectCheckpoint
		reason = err.Error()

	// Error codes which map to a duplicate transaction already
	// mined or in the mempool.
	case errors.Is(err, mempool.ErrMempoolDoubleSpend),
		errors.Is(err, mempool.ErrAlreadyVoted),
		errors.Is(err, mempool.ErrDuplicate),
		errors.Is(err, mempool.ErrTooManyVotes),
		errors.Is(err, mempool.ErrDuplicateRevocation),
		errors.Is(err, mempool.ErrAlreadyExists),
		errors.Is(err, mempool.ErrOrphan):
		code = wire.RejectDuplicate
		reason = err.Error()

	// Error codes which map to a non-standard transaction being
	// relayed.
	case errors.Is(err, mempool.ErrOrphanPolicyViolation),
		errors.Is(err, mempool.ErrOldVote),
		errors.Is(err, mempool.ErrSeqLockUnmet),
		errors.Is(err, mempool.ErrNonStandard):
		code = wire.RejectNonstandard
		reason = err.Error()

	// Error codes which map to an insufficient fee being paid.
	case errors.Is(err, mempool.ErrInsufficientFee),
		errors.Is(err, mempool.ErrInsufficientPriority):
		code = wire.RejectInsufficientFee
		reason = err.Error()

	// Error codes which map to an attempt to create dust outputs.
	case errors.Is(err, mempool.ErrDustOutput):
		code = wire.RejectDust
		reason = err.Error()

	default:
		reason = fmt.Sprintf("rejected: %v", err)
	}
	return code, reason
}

// handleTxMsg handles transaction messages from all peers.
func (b *blockManager) handleTxMsg(tmsg *txMsg) {
	peer := tmsg.peer
	state, exists := b.peerStates[peer]
	if !exists {
		bmgrLog.Warnf("Received tx message from unknown peer %s", peer)
		return
	}

	// NOTE:  BitcoinJ, and possibly other wallets, don't follow the spec of
	// sending an inventory message and allowing the remote peer to decide
	// whether or not they want to request the transaction via a getdata
	// message.  Unfortunately, the reference implementation permits
	// unrequested data, so it has allowed wallets that don't follow the
	// spec to proliferate.  While this is not ideal, there is no check here
	// to disconnect peers for sending unsolicited transactions to provide
	// interoperability.
	txHash := tmsg.tx.Hash()

	// Ignore transactions that we have already rejected.  Do not
	// send a reject message here because if the transaction was already
	// rejected, the transaction was unsolicited.
	if _, exists = b.rejectedTxns[*txHash]; exists {
		bmgrLog.Debugf("Ignoring unsolicited previously rejected "+
			"transaction %v from %s", txHash, peer)
		return
	}

	// Process the transaction to include validation, insertion in the
	// memory pool, orphan handling, etc.
	allowOrphans := b.cfg.MaxOrphanTxs > 0
	acceptedTxs, err := b.cfg.TxMemPool.ProcessTransaction(tmsg.tx,
		allowOrphans, true, true, mempool.Tag(tmsg.peer.ID()))

	// Remove transaction from request maps. Either the mempool/chain
	// already knows about it and as such we shouldn't have any more
	// instances of trying to fetch it, or we failed to insert and thus
	// we'll retry next time we get an inv.
	delete(state.requestedTxns, *txHash)
	delete(b.requestedTxns, *txHash)

	if err != nil {
		// Do not request this transaction again until a new block
		// has been processed.
		limitAdd(b.rejectedTxns, *txHash, maxRejectedTxns)

		// When the error is a rule error, it means the transaction was
		// simply rejected as opposed to something actually going wrong,
		// so log it as such.  Otherwise, something really did go wrong,
		// so log it as an actual error.
		var rErr mempool.RuleError
		if errors.As(err, &rErr) {
			bmgrLog.Debugf("Rejected transaction %v from %s: %v",
				txHash, peer, err)
		} else {
			bmgrLog.Errorf("Failed to process transaction %v: %v",
				txHash, err)
		}

		// Convert the error into an appropriate reject message and
		// send it.
		code, reason := errToWireRejectCode(err)
		peer.PushRejectMsg(wire.CmdTx, code, reason, txHash, false)
		return
	}

	b.cfg.PeerNotifier.AnnounceNewTransactions(acceptedTxs)
}

// isKnownOrphan returns whether the passed hash is currently a known orphan.
// Keep in mind that only a limited number of orphans are held onto for a
// limited amount of time, so this function must not be used as an absolute way
// to test if a block is an orphan block.  A full block (as opposed to just its
// hash) must be passed to ProcessBlock for that purpose.  This function
// provides a mechanism for a caller to intelligently detect *recent* duplicate
// orphans and react accordingly.
//
// This function is safe for concurrent access.
func (b *blockManager) isKnownOrphan(hash *chainhash.Hash) bool {
	// Protect concurrent access.  Using a read lock only so multiple readers
	// can query without blocking each other.
	b.orphanLock.RLock()
	_, exists := b.orphans[*hash]
	b.orphanLock.RUnlock()
	return exists
}

// orphanRoot returns the head of the chain for the provided hash from the map
// of orphan blocks.
//
// This function is safe for concurrent access.
func (b *blockManager) orphanRoot(hash *chainhash.Hash) *chainhash.Hash {
	// Protect concurrent access.  Using a read lock only so multiple
	// readers can query without blocking each other.
	b.orphanLock.RLock()
	defer b.orphanLock.RUnlock()

	// Keep looping while the parent of each orphaned block is known and is an
	// orphan itself.
	orphanRoot := hash
	prevHash := hash
	for {
		orphan, exists := b.orphans[*prevHash]
		if !exists {
			break
		}
		orphanRoot = prevHash
		prevHash = &orphan.block.MsgBlock().Header.PrevBlock
	}

	return orphanRoot
}

// removeOrphanBlock removes the passed orphan block from the orphan pool and
// previous orphan index.
func (b *blockManager) removeOrphanBlock(orphan *orphanBlock) {
	// Protect concurrent access.
	b.orphanLock.Lock()
	defer b.orphanLock.Unlock()

	// Remove the orphan block from the orphan pool.
	orphanHash := orphan.block.Hash()
	delete(b.orphans, *orphanHash)

	// Remove the reference from the previous orphan index too.  An indexing
	// for loop is intentionally used over a range here as range does not
	// reevaluate the slice on each iteration nor does it adjust the index
	// for the modified slice.
	prevHash := &orphan.block.MsgBlock().Header.PrevBlock
	orphans := b.prevOrphans[*prevHash]
	for i := 0; i < len(orphans); i++ {
		hash := orphans[i].block.Hash()
		if hash.IsEqual(orphanHash) {
			copy(orphans[i:], orphans[i+1:])
			orphans[len(orphans)-1] = nil
			orphans = orphans[:len(orphans)-1]
			i--
		}
	}
	b.prevOrphans[*prevHash] = orphans

	// Remove the map entry altogether if there are no longer any orphans
	// which depend on the parent hash.
	if len(b.prevOrphans[*prevHash]) == 0 {
		delete(b.prevOrphans, *prevHash)
	}
}

// addOrphanBlock adds the passed block (which is already determined to be an
// orphan prior calling this function) to the orphan pool.  It lazily cleans up
// any expired blocks so a separate cleanup poller doesn't need to be run.  It
// also imposes a maximum limit on the number of outstanding orphan blocks and
// will remove the oldest received orphan block if the limit is exceeded.
func (b *blockManager) addOrphanBlock(block *dcrutil.Block) {
	// Remove expired orphan blocks.
	for _, oBlock := range b.orphans {
		if time.Now().After(oBlock.expiration) {
			b.removeOrphanBlock(oBlock)
			continue
		}

		// Update the oldest orphan block pointer so it can be discarded
		// in case the orphan pool fills up.
		if b.oldestOrphan == nil ||
			oBlock.expiration.Before(b.oldestOrphan.expiration) {
			b.oldestOrphan = oBlock
		}
	}

	// Limit orphan blocks to prevent memory exhaustion.
	if len(b.orphans)+1 > maxOrphanBlocks {
		// Remove the oldest orphan to make room for the new one.
		b.removeOrphanBlock(b.oldestOrphan)
		b.oldestOrphan = nil
	}

	// Protect concurrent access.  This is intentionally done here instead
	// of near the top since removeOrphanBlock does its own locking and
	// the range iterator is not invalidated by removing map entries.
	b.orphanLock.Lock()
	defer b.orphanLock.Unlock()

	// Insert the block into the orphan map with an expiration time
	// 1 hour from now.
	expiration := time.Now().Add(time.Hour)
	oBlock := &orphanBlock{
		block:      block,
		expiration: expiration,
	}
	b.orphans[*block.Hash()] = oBlock

	// Add to previous hash lookup index for faster dependency lookups.
	prevHash := &block.MsgBlock().Header.PrevBlock
	b.prevOrphans[*prevHash] = append(b.prevOrphans[*prevHash], oBlock)
}

// processOrphans determines if there are any orphans which depend on the passed
// block hash (they are no longer orphans if true) and potentially accepts them.
// It repeats the process for the newly accepted blocks (to detect further
// orphans which may no longer be orphans) until there are no more.
//
// The flags do not modify the behavior of this function directly, however they
// are needed to pass along to maybeAcceptBlock.
func (b *blockManager) processOrphans(hash *chainhash.Hash, flags blockchain.BehaviorFlags) error {
	// Start with processing at least the passed hash.  Leave a little room for
	// additional orphan blocks that need to be processed without needing to
	// grow the array in the common case.
	processHashes := make([]*chainhash.Hash, 0, 10)
	processHashes = append(processHashes, hash)
	for len(processHashes) > 0 {
		// Pop the first hash to process from the slice.
		processHash := processHashes[0]
		processHashes[0] = nil // Prevent GC leak.
		processHashes = processHashes[1:]

		// Look up all orphans that are parented by the block we just accepted.
		// This will typically only be one, but it could be multiple if multiple
		// blocks are mined and broadcast around the same time.  The one with
		// the most proof of work will eventually win out.  An indexing for loop
		// is intentionally used over a range here as range does not reevaluate
		// the slice on each iteration nor does it adjust the index for the
		// modified slice.
		for i := 0; i < len(b.prevOrphans[*processHash]); i++ {
			orphan := b.prevOrphans[*processHash][i]
			if orphan == nil {
				bmgrLog.Warnf("Found a nil entry at index %d in the orphan "+
					"dependency list for block %v", i, processHash)
				continue
			}

			// Remove the orphan from the orphan pool.
			orphanHash := orphan.block.Hash()
			b.removeOrphanBlock(orphan)
			i--

			// Potentially accept the block into the block chain.
			_, err := b.cfg.Chain.ProcessBlock(orphan.block, flags)
			if err != nil {
				return err
			}

			// Add this block to the list of blocks to process so any orphan
			// blocks that depend on this block are handled too.
			processHashes = append(processHashes, orphanHash)
		}
	}
	return nil
}

// processBlockAndOrphans processes the provided block using the internal chain
// instance while keeping track of orphan blocks and also processing any orphans
// that depend on the passed block to potentially accept as well.
//
// When no errors occurred during processing, the first return value indicates
// the length of the fork the block extended.  In the case it either extended
// the best chain or is now the tip of the best chain due to causing a
// reorganize, the fork length will be 0.  The second return value indicates
// whether or not the block is an orphan, in which case the fork length will
// also be zero as expected, because it, by definition, does not connect to the
// best chain.
func (b *blockManager) processBlockAndOrphans(block *dcrutil.Block, flags blockchain.BehaviorFlags) (int64, bool, error) {
	// Process the block to include validation, best chain selection, etc.
	//
	// Also, keep track of orphan blocks in the block manager when the error
	// returned indicates the block is an orphan.
	blockHash := block.Hash()
	forkLen, err := b.cfg.Chain.ProcessBlock(block, flags)
	if errors.Is(err, blockchain.ErrMissingParent) {
		bmgrLog.Infof("Adding orphan block %v with parent %v", blockHash,
			block.MsgBlock().Header.PrevBlock)
		b.addOrphanBlock(block)

		// The fork length of orphans is unknown since they, by definition, do
		// not connect to the best chain.
		return 0, true, nil
	}
	if err != nil {
		return 0, false, err
	}

	// Accept any orphan blocks that depend on this block (they are no longer
	// orphans) and repeat for those accepted blocks until there are no more.
	if err := b.processOrphans(blockHash, flags); err != nil {
		return 0, false, err
	}

	// The chain is considered synced once both the blockchain believes it is
	// current and the sync height is reached or exceeded.
	best := b.cfg.Chain.BestSnapshot()
	syncHeight := b.SyncHeight()
	if best.Height >= syncHeight && b.cfg.Chain.IsCurrent() {
		b.isCurrentMtx.Lock()
		b.isCurrent = true
		b.isCurrentMtx.Unlock()
	}

	return forkLen, false, nil
}

// handleBlockMsg handles block messages from all peers.
func (b *blockManager) handleBlockMsg(bmsg *blockMsg) {
	peer := bmsg.peer
	state, exists := b.peerStates[peer]
	if !exists {
		bmgrLog.Warnf("Received block message from unknown peer %s", peer)
		return
	}

	// If we didn't ask for this block then the peer is misbehaving.
	blockHash := bmsg.block.Hash()
	if _, exists := state.requestedBlocks[*blockHash]; !exists {
		bmgrLog.Warnf("Got unrequested block %v from %s -- "+
			"disconnecting", blockHash, bmsg.peer.Addr())
		bmsg.peer.Disconnect()
		return
	}

	// When in headers-first mode, if the block matches the hash of the
	// first header in the list of headers that are being fetched, it's
	// eligible for less validation since the headers have already been
	// verified to link together and are valid up to the next checkpoint.
	// Also, remove the list entry for all blocks except the checkpoint
	// since it is needed to verify the next round of headers links
	// properly.
	isCheckpointBlock := false
	behaviorFlags := blockchain.BFNone
	if b.headersFirstMode {
		firstNodeEl := b.headerList.Front()
		if firstNodeEl != nil {
			firstNode := firstNodeEl.Value.(*headerNode)
			if blockHash.IsEqual(firstNode.hash) {
				behaviorFlags |= blockchain.BFFastAdd
				if firstNode.hash.IsEqual(b.nextCheckpoint.Hash) {
					isCheckpointBlock = true
				} else {
					b.headerList.Remove(firstNodeEl)
				}
			}
		}
	}

	// Remove block from request maps. Either chain will know about it and
	// so we shouldn't have any more instances of trying to fetch it, or we
	// will fail the insert and thus we'll retry next time we get an inv.
	delete(state.requestedBlocks, *blockHash)
	delete(b.requestedBlocks, *blockHash)

	// Process the block to include validation, best chain selection, orphan
	// handling, etc.
	forkLen, isOrphan, err := b.processBlockAndOrphans(bmsg.block, behaviorFlags)
	if err != nil {
		// When the error is a rule error, it means the block was simply
		// rejected as opposed to something actually going wrong, so log
		// it as such.  Otherwise, something really did go wrong, so log
		// it as an actual error.
		var rErr blockchain.RuleError
		if errors.As(err, &rErr) {
			bmgrLog.Infof("Rejected block %v from %s: %v", blockHash,
				peer, err)
		} else {
			bmgrLog.Errorf("Failed to process block %v: %v",
				blockHash, err)
		}
		var dbErr database.Error
		if errors.As(err, &dbErr) && dbErr.ErrorCode ==
			database.ErrCorruption {
			bmgrLog.Errorf("Critical failure: %v", dbErr.Error())
		}

		// Convert the error into an appropriate reject message and
		// send it.
		code, reason := errToWireRejectCode(err)
		peer.PushRejectMsg(wire.CmdBlock, code, reason, blockHash, false)
		return
	}

	// Request the parents for the orphan block from the peer that sent it.
	onMainChain := !isOrphan && forkLen == 0
	if isOrphan {
		orphanRoot := b.orphanRoot(blockHash)
		blkLocator, err := b.cfg.Chain.LatestBlockLocator()
		if err != nil {
			bmgrLog.Warnf("Failed to get block locator for the "+
				"latest block: %v", err)
		} else {
			locator := chainBlockLocatorToHashes(blkLocator)
			err = peer.PushGetBlocksMsg(locator, orphanRoot)
			if err != nil {
				bmgrLog.Warnf("Failed to push getblocksmsg for the "+
					"latest block: %v", err)
			}
		}
	} else {
		// When the block is not an orphan, log information about it and
		// update the chain state.
		b.progressLogger.LogBlockHeight(bmsg.block.MsgBlock(), b.SyncHeight())

		if onMainChain {
			// Notify stake difficulty subscribers and prune invalidated
			// transactions.
			best := b.cfg.Chain.BestSnapshot()
			if r := b.cfg.RpcServer(); r != nil {
				// Update registered websocket clients on the
				// current stake difficulty.
				r.NotifyStakeDifficulty(
					&rpcserver.StakeDifficultyNtfnData{
						BlockHash:       best.Hash,
						BlockHeight:     best.Height,
						StakeDifficulty: best.NextStakeDiff,
					})
			}
			b.cfg.TxMemPool.PruneStakeTx(best.NextStakeDiff, best.Height)
			b.cfg.TxMemPool.PruneExpiredTx()

			// Clear the rejected transactions.
			b.rejectedTxns = make(map[chainhash.Hash]struct{})

			// Proactively evict SigCache entries.
			b.proactivelyEvictSigCacheEntries(best.Height)
		}
	}

	// Update the latest block height for the peer to avoid stale heights when
	// looking for future potential sync node candidacy.
	//
	// Also, when the block is an orphan or the chain is considered current and
	// the block was accepted to the main chain, update the heights of other
	// peers whose invs may have been ignored when actively syncing while the
	// chain was not yet current or lost the lock announcement race.
	blockHeight := int64(bmsg.block.MsgBlock().Header.Height)
	peer.UpdateLastBlockHeight(blockHeight)
	if isOrphan || (onMainChain && b.IsCurrent()) {
		go b.cfg.PeerNotifier.UpdatePeerHeights(blockHash, blockHeight,
			bmsg.peer)
	}

	// Nothing more to do if we aren't in headers-first mode.
	if !b.headersFirstMode {
		return
	}

	// This is headers-first mode, so if the block is not a checkpoint
	// request more blocks using the header list when the request queue is
	// getting short.
	if !isCheckpointBlock {
		if b.startHeader != nil &&
			len(state.requestedBlocks) < minInFlightBlocks {
			b.fetchHeaderBlocks()
		}
		return
	}

	// This is headers-first mode and the block is a checkpoint.  When
	// there is a next checkpoint, get the next round of headers by asking
	// for headers starting from the block after this one up to the next
	// checkpoint.
	prevHeight := b.nextCheckpoint.Height
	prevHash := b.nextCheckpoint.Hash
	b.nextCheckpoint = b.findNextHeaderCheckpoint(prevHeight)
	if b.nextCheckpoint != nil {
		locator := []chainhash.Hash{*prevHash}
		err := peer.PushGetHeadersMsg(locator, b.nextCheckpoint.Hash)
		if err != nil {
			bmgrLog.Warnf("Failed to send getheaders message to "+
				"peer %s: %v", peer.Addr(), err)
			return
		}
		bmgrLog.Infof("Downloading headers for blocks %d to %d from peer %s",
			prevHeight+1, b.nextCheckpoint.Height, b.syncPeer.Addr())
		return
	}

	// This is headers-first mode, the block is a checkpoint, and there are
	// no more checkpoints, so switch to normal mode by requesting blocks
	// from the block after this one up to the end of the chain (zero hash).
	b.headersFirstMode = false
	b.headerList.Init()
	bmgrLog.Infof("Reached the final checkpoint -- switching to normal mode")
	locator := []chainhash.Hash{*blockHash}
	err = bmsg.peer.PushGetBlocksMsg(locator, &zeroHash)
	if err != nil {
		bmgrLog.Warnf("Failed to send getblocks message to peer %s: %v",
			peer.Addr(), err)
		return
	}
}

// proactivelyEvictSigCacheEntries fetches the block that is
// txscript.ProactiveEvictionDepth levels deep from bestHeight and passes it to
// SigCache to evict the entries associated with the transactions in that block.
func (b *blockManager) proactivelyEvictSigCacheEntries(bestHeight int64) {
	// Nothing to do before the eviction depth is reached.
	if bestHeight <= txscript.ProactiveEvictionDepth {
		return
	}

	evictHeight := bestHeight - txscript.ProactiveEvictionDepth
	block, err := b.cfg.Chain.BlockByHeight(evictHeight)
	if err != nil {
		bmgrLog.Warnf("Failed to retrieve the block at height %d: %v",
			evictHeight, err)
		return
	}

	b.cfg.SigCache.EvictEntries(block.MsgBlock())
}

// fetchHeaderBlocks creates and sends a request to the syncPeer for the next
// list of blocks to be downloaded based on the current list of headers.
func (b *blockManager) fetchHeaderBlocks() {
	// Nothing to do if there is no start header.
	if b.startHeader == nil {
		bmgrLog.Warnf("fetchHeaderBlocks called with no start header")
		return
	}

	// Build up a getdata request for the list of blocks the headers
	// describe.  The size hint will be limited to wire.MaxInvPerMsg by
	// the function, so no need to double check it here.
	gdmsg := wire.NewMsgGetDataSizeHint(uint(b.headerList.Len()))
	numRequested := 0
	for e := b.startHeader; e != nil; e = e.Next() {
		node, ok := e.Value.(*headerNode)
		if !ok {
			bmgrLog.Warn("Header list node type is not a headerNode")
			continue
		}

		iv := wire.NewInvVect(wire.InvTypeBlock, node.hash)
		haveInv, err := b.haveInventory(iv)
		if err != nil {
			bmgrLog.Warnf("Unexpected failure when checking for "+
				"existing inventory during header block "+
				"fetch: %v", err)
			continue
		}
		if !haveInv {
			b.requestedBlocks[*node.hash] = struct{}{}
			syncPeerState := b.peerStates[b.syncPeer]
			syncPeerState.requestedBlocks[*node.hash] = struct{}{}
			err = gdmsg.AddInvVect(iv)
			if err != nil {
				bmgrLog.Warnf("Failed to add invvect while fetching "+
					"block headers: %v", err)
			}
			numRequested++
		}
		b.startHeader = e.Next()
		if numRequested >= wire.MaxInvPerMsg {
			break
		}
	}
	if len(gdmsg.InvList) > 0 {
		b.syncPeer.QueueMessage(gdmsg, nil)
	}
}

// handleHeadersMsg handles headers messages from all peers.
func (b *blockManager) handleHeadersMsg(hmsg *headersMsg) {
	peer := hmsg.peer
	_, exists := b.peerStates[peer]
	if !exists {
		bmgrLog.Warnf("Received headers message from unknown peer %s", peer)
		return
	}

	// The remote peer is misbehaving if we didn't request headers.
	msg := hmsg.headers
	numHeaders := len(msg.Headers)
	if !b.headersFirstMode {
		bmgrLog.Warnf("Got %d unrequested headers from %s -- "+
			"disconnecting", numHeaders, peer.Addr())
		peer.Disconnect()
		return
	}

	// Nothing to do for an empty headers message.
	if numHeaders == 0 {
		return
	}

	// Process all of the received headers ensuring each one connects to the
	// previous and that checkpoints match.
	receivedCheckpoint := false
	var finalHash *chainhash.Hash
	for _, blockHeader := range msg.Headers {
		blockHash := blockHeader.BlockHash()
		finalHash = &blockHash

		// Ensure there is a previous header to compare against.
		prevNodeEl := b.headerList.Back()
		if prevNodeEl == nil {
			bmgrLog.Warnf("Header list does not contain a previous " +
				"element as expected -- disconnecting peer")
			peer.Disconnect()
			return
		}

		// Ensure the header properly connects to the previous one and
		// add it to the list of headers.
		node := headerNode{hash: &blockHash}
		prevNode := prevNodeEl.Value.(*headerNode)
		if prevNode.hash.IsEqual(&blockHeader.PrevBlock) {
			node.height = prevNode.height + 1
			e := b.headerList.PushBack(&node)
			if b.startHeader == nil {
				b.startHeader = e
			}
		} else {
			bmgrLog.Warnf("Received block header that does not "+
				"properly connect to the chain from peer %s "+
				"-- disconnecting", peer.Addr())
			peer.Disconnect()
			return
		}

		// Verify the header at the next checkpoint height matches.
		if node.height == b.nextCheckpoint.Height {
			if node.hash.IsEqual(b.nextCheckpoint.Hash) {
				receivedCheckpoint = true
				bmgrLog.Infof("Verified downloaded block "+
					"header against checkpoint at height "+
					"%d/hash %s", node.height, node.hash)
			} else {
				bmgrLog.Warnf("Block header at height %d/hash "+
					"%s from peer %s does NOT match "+
					"expected checkpoint hash of %s -- "+
					"disconnecting", node.height, node.hash,
					peer.Addr(), b.nextCheckpoint.Hash)
				peer.Disconnect()
				return
			}
			break
		}
	}

	// When this header is a checkpoint, switch to fetching the blocks for
	// all of the headers since the last checkpoint.
	if receivedCheckpoint {
		// Since the first entry of the list is always the final block
		// that is already in the database and is only used to ensure
		// the next header links properly, it must be removed before
		// fetching the blocks.
		b.headerList.Remove(b.headerList.Front())
		bmgrLog.Infof("Received %v block headers: Fetching blocks",
			b.headerList.Len())
		b.progressLogger.SetLastLogTime(time.Now())
		b.fetchHeaderBlocks()
		return
	}

	// This header is not a checkpoint, so request the next batch of
	// headers starting from the latest known header and ending with the
	// next checkpoint.
	locator := []chainhash.Hash{*finalHash}
	err := peer.PushGetHeadersMsg(locator, b.nextCheckpoint.Hash)
	if err != nil {
		bmgrLog.Warnf("Failed to send getheaders message to "+
			"peer %s: %v", peer.Addr(), err)
		return
	}
}

// handleNotFoundMsg handles notfound messages from all peers.
func (b *blockManager) handleNotFoundMsg(nfmsg *notFoundMsg) {
	peer := nfmsg.peer
	state, exists := b.peerStates[peer]
	if !exists {
		bmgrLog.Warnf("Received notfound message from unknown peer %s", peer)
		return
	}
	for _, inv := range nfmsg.notFound.InvList {
		// verify the hash was actually announced by the peer
		// before deleting from the global requested maps.
		switch inv.Type {
		case wire.InvTypeBlock:
			if _, exists := state.requestedBlocks[inv.Hash]; exists {
				delete(state.requestedBlocks, inv.Hash)
				delete(b.requestedBlocks, inv.Hash)
			}
		case wire.InvTypeTx:
			if _, exists := state.requestedTxns[inv.Hash]; exists {
				delete(state.requestedTxns, inv.Hash)
				delete(b.requestedTxns, inv.Hash)
			}
		}
	}
}

// haveInventory returns whether or not the inventory represented by the passed
// inventory vector is known.  This includes checking all of the various places
// inventory can be when it is in different states such as blocks that are part
// of the main chain, on a side chain, in the orphan pool, and transactions that
// are in the memory pool (either the main pool or orphan pool).
func (b *blockManager) haveInventory(invVect *wire.InvVect) (bool, error) {
	switch invVect.Type {
	case wire.InvTypeBlock:
		// Determine if the block is known in any form (main chain, side
		// chain, or orphan).
		hash := &invVect.Hash
		return b.isKnownOrphan(hash) || b.cfg.Chain.HaveBlock(hash), nil

	case wire.InvTypeTx:
		// Ask the transaction memory pool if the transaction is known
		// to it in any form (main pool or orphan).
		if b.cfg.TxMemPool.HaveTransaction(&invVect.Hash) {
			return true, nil
		}

		// Check if the transaction exists from the point of view of the
		// end of the main chain.
		entry, err := b.cfg.Chain.FetchUtxoEntry(&invVect.Hash)
		if err != nil {
			return false, err
		}
		return entry != nil && !entry.IsFullySpent(), nil
	}

	// The requested inventory is an unsupported type, so just claim
	// it is known to avoid requesting it.
	return true, nil
}

// handleInvMsg handles inv messages from all peers.
// We examine the inventory advertised by the remote peer and act accordingly.
func (b *blockManager) handleInvMsg(imsg *invMsg) {
	peer := imsg.peer
	state, exists := b.peerStates[peer]
	if !exists {
		bmgrLog.Warnf("Received inv message from unknown peer %s", peer)
		return
	}

	// Attempt to find the final block in the inventory list.  There may
	// not be one.
	lastBlock := -1
	invVects := imsg.inv.InvList
	for i := len(invVects) - 1; i >= 0; i-- {
		if invVects[i].Type == wire.InvTypeBlock {
			lastBlock = i
			break
		}
	}

	fromSyncPeer := peer == b.syncPeer
	isCurrent := b.IsCurrent()

	// If this inv contains a block announcement, and this isn't coming from
	// our current sync peer or we're current, then update the last
	// announced block for this peer. We'll use this information later to
	// update the heights of peers based on blocks we've accepted that they
	// previously announced.
	if lastBlock != -1 && (!fromSyncPeer || isCurrent) {
		peer.UpdateLastAnnouncedBlock(&invVects[lastBlock].Hash)
	}

	// Ignore invs from peers that aren't the sync if we are not current.
	// Helps prevent fetching a mass of orphans.
	if !fromSyncPeer && !isCurrent {
		return
	}

	// If our chain is current and a peer announces a block we already
	// know of, then update their current block height.
	if lastBlock != -1 && isCurrent {
		blkHeight, err := b.cfg.Chain.BlockHeightByHash(&invVects[lastBlock].Hash)
		if err == nil {
			imsg.peer.UpdateLastBlockHeight(blkHeight)
		}
	}

	// Request the advertised inventory if we don't already have it.  Also,
	// request parent blocks of orphans if we receive one we already have.
	// Finally, attempt to detect potential stalls due to long side chains
	// we already have and request more blocks to prevent them.
	var requestQueue []*wire.InvVect
	for i, iv := range invVects {
		// Ignore unsupported inventory types.
		if iv.Type != wire.InvTypeBlock && iv.Type != wire.InvTypeTx {
			continue
		}

		// Add the inventory to the cache of known inventory
		// for the peer.
		peer.AddKnownInventory(iv)

		// Ignore inventory when we're in headers-first mode.
		if b.headersFirstMode {
			continue
		}

		// Request the inventory if we don't already have it.
		haveInv, err := b.haveInventory(iv)
		if err != nil {
			bmgrLog.Warnf("Unexpected failure when checking for "+
				"existing inventory during inv message "+
				"processing: %v", err)
			continue
		}
		if !haveInv {
			if iv.Type == wire.InvTypeTx {
				// Skip the transaction if it has already been
				// rejected.
				if _, exists := b.rejectedTxns[iv.Hash]; exists {
					continue
				}
			}

			// Add it to the request queue.
			requestQueue = append(requestQueue, iv)
			continue
		}

		if iv.Type == wire.InvTypeBlock {
			// The block is an orphan block that we already have.
			// When the existing orphan was processed, it requested
			// the missing parent blocks.  When this scenario
			// happens, it means there were more blocks missing
			// than are allowed into a single inventory message.  As
			// a result, once this peer requested the final
			// advertised block, the remote peer noticed and is now
			// resending the orphan block as an available block
			// to signal there are more missing blocks that need to
			// be requested.
			if b.isKnownOrphan(&iv.Hash) {
				// Request blocks starting at the latest known
				// up to the root of the orphan that just came
				// in.
				orphanRoot := b.orphanRoot(&iv.Hash)
				blkLocator, err := b.cfg.Chain.LatestBlockLocator()
				if err != nil {
					bmgrLog.Errorf("PEER: Failed to get block "+
						"locator for the latest block: "+
						"%v", err)
					continue
				}
				locator := chainBlockLocatorToHashes(blkLocator)
				err = peer.PushGetBlocksMsg(locator, orphanRoot)
				if err != nil {
					bmgrLog.Errorf("PEER: Failed to push getblocksmsg "+
						"for orphan chain: %v", err)
				}
				continue
			}

			// We already have the final block advertised by this
			// inventory message, so force a request for more.  This
			// should only happen if we're on a really long side
			// chain.
			if i == lastBlock {
				// Request blocks after this one up to the
				// final one the remote peer knows about (zero
				// stop hash).
				blkLocator := b.cfg.Chain.BlockLocatorFromHash(&iv.Hash)
				locator := chainBlockLocatorToHashes(blkLocator)
				err = imsg.peer.PushGetBlocksMsg(locator, &zeroHash)
				if err != nil {
					bmgrLog.Errorf("PEER: Failed to push getblocksmsg: "+
						"%v", err)
				}
			}
		}
	}

	// Request as much as possible at once.
	numRequested := 0
	gdmsg := wire.NewMsgGetData()
	for _, iv := range requestQueue {
		switch iv.Type {
		case wire.InvTypeBlock:
			// Request the block if there is not already a pending
			// request.
			if _, exists := b.requestedBlocks[iv.Hash]; !exists {
				limitAdd(b.requestedBlocks, iv.Hash, maxRequestedBlocks)
				limitAdd(state.requestedBlocks, iv.Hash, maxRequestedBlocks)
				gdmsg.AddInvVect(iv)
				numRequested++
			}

		case wire.InvTypeTx:
			// Request the transaction if there is not already a
			// pending request.
			if _, exists := b.requestedTxns[iv.Hash]; !exists {
				limitAdd(b.requestedTxns, iv.Hash, maxRequestedTxns)
				limitAdd(state.requestedTxns, iv.Hash, maxRequestedTxns)
				gdmsg.AddInvVect(iv)
				numRequested++
			}
		}

		if numRequested == wire.MaxInvPerMsg {
			// Send full getdata message and reset.
			//
			// NOTE: There should never be more than wire.MaxInvPerMsg
			// in the inv request, so we could return after the
			// QueueMessage, but this is more safe.
			imsg.peer.QueueMessage(gdmsg, nil)
			gdmsg = wire.NewMsgGetData()
			numRequested = 0
		}
	}

	if len(gdmsg.InvList) > 0 {
		peer.QueueMessage(gdmsg, nil)
	}
}

// limitAdd is a helper function for maps that require a maximum limit by
// evicting a random value if adding the new value would cause it to
// overflow the maximum allowed.
func limitAdd(m map[chainhash.Hash]struct{}, hash chainhash.Hash, limit int) {
	if len(m)+1 > limit {
		// Remove a random entry from the map.  For most compilers, Go's
		// range statement iterates starting at a random item although
		// that is not 100% guaranteed by the spec.  The iteration order
		// is not important here because an adversary would have to be
		// able to pull off preimage attacks on the hashing function in
		// order to target eviction of specific entries anyways.
		for txHash := range m {
			delete(m, txHash)
			break
		}
	}
	m[hash] = struct{}{}
}

// blockHandler is the main handler for the block manager.  It must be run
// as a goroutine.  It processes block and inv messages in a separate goroutine
// from the peer handlers so the block (MsgBlock) messages are handled by a
// single thread without needing to lock memory data structures.  This is
// important because the block manager controls which blocks are needed and how
// the fetching should proceed.
func (b *blockManager) blockHandler() {
out:
	for {
		select {
		case m := <-b.msgChan:
			switch msg := m.(type) {
			case *newPeerMsg:
				b.handleNewPeerMsg(msg.peer)

			case *txMsg:
				b.handleTxMsg(msg)
				msg.reply <- struct{}{}

			case *blockMsg:
				b.handleBlockMsg(msg)
				msg.reply <- struct{}{}

			case *invMsg:
				b.handleInvMsg(msg)

			case *headersMsg:
				b.handleHeadersMsg(msg)

			case *notFoundMsg:
				b.handleNotFoundMsg(msg)

			case *donePeerMsg:
				b.handleDonePeerMsg(msg.peer)

			case getSyncPeerMsg:
				var peerID int32
				if b.syncPeer != nil {
					peerID = b.syncPeer.ID()
				}
				msg.reply <- peerID

			case requestFromPeerMsg:
				err := b.requestFromPeer(msg.peer, msg.blocks, msg.txs)
				msg.reply <- requestFromPeerResponse{
					err: err,
				}

			case forceReorganizationMsg:
				err := b.cfg.Chain.ForceHeadReorganization(
					msg.formerBest, msg.newBest)

				if err == nil {
					// Notify stake difficulty subscribers and prune
					// invalidated transactions.
					best := b.cfg.Chain.BestSnapshot()
					if r := b.cfg.RpcServer(); r != nil {
						r.NotifyStakeDifficulty(
							&rpcserver.StakeDifficultyNtfnData{
								BlockHash:       best.Hash,
								BlockHeight:     best.Height,
								StakeDifficulty: best.NextStakeDiff,
							})
					}
					b.cfg.TxMemPool.PruneStakeTx(best.NextStakeDiff,
						best.Height)
					b.cfg.TxMemPool.PruneExpiredTx()
				}

				msg.reply <- forceReorganizationResponse{
					err: err,
				}

			case tipGenerationMsg:
				g, err := b.cfg.Chain.TipGeneration()
				msg.reply <- tipGenerationResponse{
					hashes: g,
					err:    err,
				}

			case processBlockMsg:
				forkLen, isOrphan, err := b.processBlockAndOrphans(msg.block,
					msg.flags)
				if err != nil {
					msg.reply <- processBlockResponse{
						forkLen:  forkLen,
						isOrphan: isOrphan,
						err:      err,
					}
					continue
				}

				onMainChain := !isOrphan && forkLen == 0
				if onMainChain {
					// Notify stake difficulty subscribers and prune
					// invalidated transactions.
					best := b.cfg.Chain.BestSnapshot()
					if r := b.cfg.RpcServer(); r != nil {
						r.NotifyStakeDifficulty(
							&rpcserver.StakeDifficultyNtfnData{
								BlockHash:       best.Hash,
								BlockHeight:     best.Height,
								StakeDifficulty: best.NextStakeDiff,
							})
					}
					b.cfg.TxMemPool.PruneStakeTx(best.NextStakeDiff,
						best.Height)
					b.cfg.TxMemPool.PruneExpiredTx()
				}

				msg.reply <- processBlockResponse{
					isOrphan: isOrphan,
					err:      nil,
				}

			case processTransactionMsg:
				acceptedTxs, err := b.cfg.TxMemPool.ProcessTransaction(msg.tx,
					msg.allowOrphans, msg.rateLimit, msg.allowHighFees, msg.tag)
				msg.reply <- processTransactionResponse{
					acceptedTxs: acceptedTxs,
					err:         err,
				}

			default:
				bmgrLog.Warnf("Invalid message type in block handler: %T", msg)
			}

		case <-b.quit:
			break out
		}
	}

	b.wg.Done()
	bmgrLog.Trace("Block handler done")
}

// NewPeer informs the block manager of a newly active peer.
func (b *blockManager) NewPeer(peer *peerpkg.Peer) {
	// Ignore if we are shutting down.
	if atomic.LoadInt32(&b.shutdown) != 0 {
		return
	}
	b.msgChan <- &newPeerMsg{peer: peer}
}

// QueueTx adds the passed transaction message and peer to the block handling
// queue.
func (b *blockManager) QueueTx(tx *dcrutil.Tx, peer *peerpkg.Peer, done chan struct{}) {
	// Don't accept more transactions if we're shutting down.
	if atomic.LoadInt32(&b.shutdown) != 0 {
		done <- struct{}{}
		return
	}

	b.msgChan <- &txMsg{tx: tx, peer: peer, reply: done}
}

// QueueBlock adds the passed block message and peer to the block handling queue.
func (b *blockManager) QueueBlock(block *dcrutil.Block, peer *peerpkg.Peer, done chan struct{}) {
	// Don't accept more blocks if we're shutting down.
	if atomic.LoadInt32(&b.shutdown) != 0 {
		done <- struct{}{}
		return
	}

	b.msgChan <- &blockMsg{block: block, peer: peer, reply: done}
}

// QueueInv adds the passed inv message and peer to the block handling queue.
func (b *blockManager) QueueInv(inv *wire.MsgInv, peer *peerpkg.Peer) {
	// No channel handling here because peers do not need to block on inv
	// messages.
	if atomic.LoadInt32(&b.shutdown) != 0 {
		return
	}

	b.msgChan <- &invMsg{inv: inv, peer: peer}
}

// QueueHeaders adds the passed headers message and peer to the block handling
// queue.
func (b *blockManager) QueueHeaders(headers *wire.MsgHeaders, peer *peerpkg.Peer) {
	// No channel handling here because peers do not need to block on
	// headers messages.
	if atomic.LoadInt32(&b.shutdown) != 0 {
		return
	}

	b.msgChan <- &headersMsg{headers: headers, peer: peer}
}

// QueueNotFound adds the passed notfound message and peer to the block handling
// queue.
func (b *blockManager) QueueNotFound(notFound *wire.MsgNotFound, peer *peerpkg.Peer) {
	// No channel handling here because peers do not need to block on
	// reject messages.
	if atomic.LoadInt32(&b.shutdown) != 0 {
		return
	}

	b.msgChan <- &notFoundMsg{notFound: notFound, peer: peer}
}

// DonePeer informs the blockmanager that a peer has disconnected.
func (b *blockManager) DonePeer(peer *peerpkg.Peer) {
	// Ignore if we are shutting down.
	if atomic.LoadInt32(&b.shutdown) != 0 {
		return
	}

	b.msgChan <- &donePeerMsg{peer: peer}
}

// Start begins the core block handler which processes block and inv messages.
func (b *blockManager) Start() {
	// Already started?
	if atomic.AddInt32(&b.started, 1) != 1 {
		return
	}

	bmgrLog.Trace("Starting block manager")
	b.wg.Add(1)
	go b.blockHandler()
}

// Stop gracefully shuts down the block manager by stopping all asynchronous
// handlers and waiting for them to finish.
func (b *blockManager) Stop() error {
	if atomic.AddInt32(&b.shutdown, 1) != 1 {
		bmgrLog.Warnf("Block manager is already in the process of " +
			"shutting down")
		return nil
	}

	bmgrLog.Infof("Block manager shutting down")
	close(b.quit)
	b.wg.Wait()
	return nil
}

// SyncPeerID returns the ID of the current sync peer, or 0 if there is none.
func (b *blockManager) SyncPeerID() int32 {
	reply := make(chan int32)
	b.msgChan <- getSyncPeerMsg{reply: reply}
	return <-reply
}

// RequestFromPeer allows an outside caller to request blocks or transactions
// from a peer. The requests are logged in the blockmanager's internal map of
// requests so they do not later ban the peer for sending the respective data.
func (b *blockManager) RequestFromPeer(p *peerpkg.Peer, blocks, txs []*chainhash.Hash) error {
	reply := make(chan requestFromPeerResponse)
	b.msgChan <- requestFromPeerMsg{peer: p, blocks: blocks, txs: txs,
		reply: reply}
	response := <-reply

	return response.err
}

func (b *blockManager) requestFromPeer(p *peerpkg.Peer, blocks, txs []*chainhash.Hash) error {
	msgResp := wire.NewMsgGetData()

	state, exists := b.peerStates[p]
	if !exists {
		return fmt.Errorf("unknown peer %s", p)
	}

	// Add the blocks to the request.
	for _, bh := range blocks {
		// If we've already requested this block, skip it.
		_, alreadyReqP := state.requestedBlocks[*bh]
		_, alreadyReqB := b.requestedBlocks[*bh]

		if alreadyReqP || alreadyReqB {
			continue
		}

		// Skip the block when it is already known.
		if b.isKnownOrphan(bh) || b.cfg.Chain.HaveBlock(bh) {
			continue
		}

		err := msgResp.AddInvVect(wire.NewInvVect(wire.InvTypeBlock, bh))
		if err != nil {
			return fmt.Errorf("unexpected error encountered building request "+
				"for mining state block %v: %v",
				bh, err.Error())
		}

		state.requestedBlocks[*bh] = struct{}{}
		b.requestedBlocks[*bh] = struct{}{}
	}

	// Add the vote transactions to the request.
	for _, vh := range txs {
		// If we've already requested this transaction, skip it.
		_, alreadyReqP := state.requestedTxns[*vh]
		_, alreadyReqB := b.requestedTxns[*vh]

		if alreadyReqP || alreadyReqB {
			continue
		}

		// Ask the transaction memory pool if the transaction is known
		// to it in any form (main pool or orphan).
		if b.cfg.TxMemPool.HaveTransaction(vh) {
			continue
		}

		// Check if the transaction exists from the point of view of the
		// end of the main chain.
		entry, err := b.cfg.Chain.FetchUtxoEntry(vh)
		if err != nil {
			return err
		}
		if entry != nil {
			continue
		}

		err = msgResp.AddInvVect(wire.NewInvVect(wire.InvTypeTx, vh))
		if err != nil {
			return fmt.Errorf("unexpected error encountered building request "+
				"for mining state vote %v: %v",
				vh, err.Error())
		}

		state.requestedTxns[*vh] = struct{}{}
		b.requestedTxns[*vh] = struct{}{}
	}

	if len(msgResp.InvList) > 0 {
		p.QueueMessage(msgResp, nil)
	}

	return nil
}

// ForceReorganization forces a reorganization of the block chain to the block
// hash requested, so long as it matches up with the current organization of the
// best chain.  It is funneled through the block manager since blockchain is not
// safe for concurrent access.
func (b *blockManager) ForceReorganization(formerBest, newBest chainhash.Hash) error {
	reply := make(chan forceReorganizationResponse)
	b.msgChan <- forceReorganizationMsg{
		formerBest: formerBest,
		newBest:    newBest,
		reply:      reply}
	response := <-reply
	return response.err
}

// TipGeneration returns the hashes of all the children of the current best
// chain tip.  It is funneled through the block manager since blockchain is not
// safe for concurrent access.
func (b *blockManager) TipGeneration() ([]chainhash.Hash, error) {
	reply := make(chan tipGenerationResponse)
	b.msgChan <- tipGenerationMsg{reply: reply}
	response := <-reply
	return response.hashes, response.err
}

// ProcessBlock makes use of ProcessBlock on an internal instance of a block
// chain.  It is funneled through the block manager since blockchain is not safe
// for concurrent access.
func (b *blockManager) ProcessBlock(block *dcrutil.Block, flags blockchain.BehaviorFlags) (bool, error) {
	reply := make(chan processBlockResponse, 1)
	b.msgChan <- processBlockMsg{block: block, flags: flags, reply: reply}
	response := <-reply
	return response.isOrphan, response.err
}

// ProcessTransaction makes use of ProcessTransaction on an internal instance of
// a block chain.  It is funneled through the block manager since blockchain is
// not safe for concurrent access.
func (b *blockManager) ProcessTransaction(tx *dcrutil.Tx, allowOrphans bool,
	rateLimit bool, allowHighFees bool, tag mempool.Tag) ([]*dcrutil.Tx, error) {
	reply := make(chan processTransactionResponse, 1)
	b.msgChan <- processTransactionMsg{tx, allowOrphans, rateLimit,
		allowHighFees, tag, reply}
	response := <-reply
	return response.acceptedTxs, response.err
}

// IsCurrent returns whether or not the block manager believes it is synced with
// the connected peers.
//
// This function is safe for concurrent access.
func (b *blockManager) IsCurrent() bool {
	b.isCurrentMtx.RLock()
	isCurrent := b.isCurrent
	b.isCurrentMtx.RUnlock()
	return isCurrent
}

// TicketPoolValue returns the current value of the total stake in the ticket
// pool.
func (b *blockManager) TicketPoolValue() (dcrutil.Amount, error) {
	return b.cfg.Chain.TicketPoolValue()
}

// newBlockManager returns a new Decred block manager.
// Use Start to begin processing asynchronous block and inv updates.
func newBlockManager(config *blockManagerConfig) (*blockManager, error) {
	bm := blockManager{
		cfg:             config,
		rejectedTxns:    make(map[chainhash.Hash]struct{}),
		requestedTxns:   make(map[chainhash.Hash]struct{}),
		requestedBlocks: make(map[chainhash.Hash]struct{}),
		peerStates:      make(map[*peerpkg.Peer]*peerSyncState),
		progressLogger:  progresslog.New("Processed", bmgrLog),
		msgChan:         make(chan interface{}, config.MaxPeers*3),
		headerList:      list.New(),
		quit:            make(chan struct{}),
		orphans:         make(map[chainhash.Hash]*orphanBlock),
		prevOrphans:     make(map[chainhash.Hash][]*orphanBlock),
		isCurrent:       config.Chain.IsCurrent(),
	}

	best := bm.cfg.Chain.BestSnapshot()
	if !bm.cfg.DisableCheckpoints {
		// Initialize the next checkpoint based on the current height.
		bm.nextCheckpoint = bm.findNextHeaderCheckpoint(best.Height)
		if bm.nextCheckpoint != nil {
			bm.resetHeaderState(&best.Hash, best.Height)
		}
	} else {
		bmgrLog.Info("Checkpoints are disabled")
	}

	bm.syncHeightMtx.Lock()
	bm.syncHeight = best.Height
	bm.syncHeightMtx.Unlock()

	return &bm, nil
}
