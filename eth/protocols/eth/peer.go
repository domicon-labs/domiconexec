// Copyright 2020 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package eth

import (
	"math/big"
	"math/rand"
	"sync"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rlp"
)

const (
	// maxKnownTxs is the maximum transactions hashes to keep in the known list
	// before starting to randomly evict them.
	maxKnownTxs = 32768

	// maxKnownFds is the maximum fileData hashes to keep in the known list
	// before starting to randomly evict them.
	maxKnownFds = 32768

	// maxKnownBlocks is the maximum block hashes to keep in the known list
	// before starting to randomly evict them.
	maxKnownBlocks = 1024

	// maxQueuedTxs is the maximum number of transactions to queue up before dropping
	// older broadcasts.
	maxQueuedTxs = 4096

	// maxQueuedFileData is the maximum number of fileData to queue up before dropping
	// older broadcasts.
	maxQueuedFileData = 4096

	// maxQueuedTxAnns is the maximum number of transaction announcements to queue up
	// before dropping older announcements.
	maxQueuedTxAnns = 4096

	// maxQueuedFdAnns is the maximum number of fileData announcements to queue up
	// before dropping older announcements.
	maxQueuedFdAnns = 4096

	// maxQueuedBlocks is the maximum number of block propagations to queue up before
	// dropping broadcasts. There's not much point in queueing stale blocks, so a few
	// that might cover uncles should be enough.
	maxQueuedBlocks = 4

	// maxQueuedBlockAnns is the maximum number of block announcements to queue up before
	// dropping broadcasts. Similarly to block propagations, there's no point to queue
	// above some healthy uncle limit, so use that.
	maxQueuedBlockAnns = 4
)

// max is a helper function which returns the larger of the two given integers.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Peer is a collection of relevant information we have about a `eth` peer.
type Peer struct {
	id string // Unique ID for the peer, cached

	*p2p.Peer                   // The embedded P2P package peer
	rw        p2p.MsgReadWriter // Input/output streams for snap
	version   uint              // Protocol version negotiated

	head common.Hash // Latest advertised head block hash
	td   *big.Int    // Latest advertised head block total difficulty
	//
	//knownBlocks     *knownCache            // Set of block hashes known to be known by this peer
	//queuedBlocks    chan *blockPropagation // Queue of blocks to broadcast to the peer
	//queuedBlockAnns chan *types.Block      // Queue of blocks to announce to the peer

	//txpool      TxPool             // Transaction pool used by the broadcasters for liveness checks
	//knownTxs    *knownCache        // Set of transaction hashes known to be known by this peer
	//txBroadcast chan []common.Hash // Channel used to queue transaction propagation requests
	//txAnnounce  chan []common.Hash // Channel used to queue transaction announcement requests

	fdpool      FileDataPool       // fileData pool used by the broadcasters for liveness checks
	knownFds    *knownCache        // Set of fileData hashes known to be known by this peer
	fdBroadcast chan []common.Hash // Channel used to queue fileData propagation requests
	fdAnnounce  chan []common.Hash // Channel used to queue fileData announcement requests

	reqDispatch chan *request  // Dispatch channel to send requests and track then until fulfilment
	reqCancel   chan *cancel   // Dispatch channel to cancel pending requests and untrack them
	resDispatch chan *response // Dispatch channel to fulfil pending requests and untrack them

	term chan struct{} // Termination channel to stop the broadcasters
	lock sync.RWMutex  // Mutex protecting the internal fields
}

// NewPeer create a wrapper for a network connection and negotiated  protocol
// version.
func NewPeer(version uint, p *p2p.Peer, rw p2p.MsgReadWriter, fdpool FileDataPool) *Peer {
	peer := &Peer{
		id:              p.ID().String(),
		Peer:            p,
		rw:              rw,
		version:         version,
		//knownTxs:        newKnownCache(maxKnownTxs),
		knownFds:        newKnownCache(maxKnownFds),
		//knownBlocks:     newKnownCache(maxKnownBlocks),
		//queuedBlocks:    make(chan *blockPropagation, maxQueuedBlocks),
		//queuedBlockAnns: make(chan *types.Block, maxQueuedBlockAnns),
		//txBroadcast:     make(chan []common.Hash),
		//txAnnounce:      make(chan []common.Hash),
		fdBroadcast:     make(chan []common.Hash),
		fdAnnounce:      make(chan []common.Hash),
		reqDispatch:     make(chan *request),
		reqCancel:       make(chan *cancel),
		resDispatch:     make(chan *response),
		//txpool:          txpool,
		fdpool:          fdpool,
		term:            make(chan struct{}),
	}
	// Start up all the broadcasters
	//go peer.broadcastBlocks()
	//go peer.broadcastTransactions()
	//go peer.announceTransactions()
	go peer.broadcastFileData()
	go peer.announceFileDatas()
	go peer.dispatcher()

	return peer
}

// Close signals the broadcast goroutine to terminate. Only ever call this if
// you created the peer yourself via NewPeer. Otherwise let whoever created it
// clean it up!
func (p *Peer) Close() {
	close(p.term)
}

// ID retrieves the peer's unique identifier.
func (p *Peer) ID() string {
	return p.id
}

// Version retrieves the peer's negotiated `eth` protocol version.
func (p *Peer) Version() uint {
	return p.version
}

// Head retrieves the current head hash and total difficulty of the peer.
func (p *Peer) Head() (hash common.Hash, td *big.Int) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	copy(hash[:], p.head[:])
	return hash, new(big.Int).Set(p.td)
}

// SetHead updates the head hash and total difficulty of the peer.
func (p *Peer) SetHead(hash common.Hash, td *big.Int) {
	p.lock.Lock()
	defer p.lock.Unlock()

	copy(p.head[:], hash[:])
	p.td.Set(td)
}


// KnownFileData returns whether peer is known to already have a fileData.
func (p *Peer) KnownFileData(hash common.Hash) bool {
	return p.knownFds.Contains(hash)
}


// markFileData marks a fileData as known for the peer, ensuring that it
// will never be propagated to this particular peer.
func (p *Peer) markFileData(hash common.Hash) {
	p.knownFds.Add(hash)
}


// SendFileDatas sends fileData to the peer and includes the hashes
// in its fileData hash set for future reference.
func (p *Peer) SendFileDatas(fds []*types.DA) error {
	var txHash common.Hash
	var commitIsEmpty bool
	for _, fd := range fds {
		txHash = fd.TxHash
		if fd.Commitment.X.IsZero() && fd.Commitment.Y.IsZero() {
			commitIsEmpty = true
		}
		switch  {
		case txHash.Cmp(common.Hash{}) == 0 && commitIsEmpty:
			return errDADataIllegal
		case txHash.Cmp(common.Hash{}) == 0 && !commitIsEmpty:
			cmHash := common.BytesToHash(fd.Commitment.Marshal())
			p.knownFds.Add(cmHash)
		case txHash.Cmp(common.Hash{}) != 0 && !commitIsEmpty:
			cmHash := common.BytesToHash(fd.Commitment.Marshal())
			p.knownFds.Add(cmHash)
			p.knownFds.Add(fd.TxHash)
		case txHash.Cmp(common.Hash{}) != 0 && commitIsEmpty:
			p.knownFds.Add(fd.TxHash)
		}
	}
	log.Info("SendFileDatas----", "FileDataMsg", FileDataMsg, "fds length", len(fds), "peer id", p.ID(), "txHash", txHash.String())
	return p2p.Send(p.rw, FileDataMsg, fds)
}

// AsyncSendFileData queues a list of fileData (by txHash hash) to eventually
// propagate to a remote peer. The number of pending sends are capped (new ones
// will force old sends to be dropped)
func (p *Peer) AsyncSendFileData(hashes []common.Hash) {
	select {
	case p.fdBroadcast <- hashes:
		// Mark all the fileData as known, but ensure we don't overflow our limits
		p.knownFds.Add(hashes...)
	case <-p.term:
		p.Log().Debug("Dropping fileData propagation", "count", len(hashes))
	}
}

// sendPooledFileDataHashes66 sends fileData hashes to the peer and includes
// them in its transaction hash set for future reference.
//
// This method is a helper used by the async fileData announcer. Don't call it
// directly as the queueing (memory) and transmission (bandwidth) costs should
// not be managed directly.
func (p *Peer) sendPooledFileDataHashes66(hashes []common.Hash) error {
	// Mark all the fileDatas as known, but ensure we don't overflow our limits
	p.knownFds.Add(hashes...)
	log.Info("sendPooledFileDataHashes66---广播交易哈希", "txHash", hashes[0].String())
	return p2p.Send(p.rw, NewPooledFileDataHashesMsg, NewPooledFileDataHashesPacket67(hashes))
}

// sendPooledFileDataHashes68 sends fileData hashes (tagged with their type
// and size) to the peer and includes them in its fileData hash set for future
// reference.
//
// This method is a helper used by the async fileData announcer. Don't call it
// directly as the queueing (memory) and fileDatamission (bandwidth) costs should
// not be managed directly.
func (p *Peer) sendPooledFileDataHashes68(hashes []common.Hash, sizes []uint32) error {
	// Mark all the fileDatas as known, but ensure we don't overflow our limits
	p.knownFds.Add(hashes...)
	log.Info("sendPooledFileDataHashes68---广播交易哈希", "txHash", hashes[0].String())
	return p2p.Send(p.rw, NewPooledFileDataHashesMsg, NewPooledFileDataHashesPacket68{Sizes: sizes, Hashes: hashes})
}

// AsyncSendPooledFileDataHashes queues a list of fileDatas hashes to eventually
// announce to a remote peer.  The number of pending sends are capped (new ones
// will force old sends to be dropped)
func (p *Peer) AsyncSendPooledFileDataHashes(hashes []common.Hash) {
	select {
	case p.fdAnnounce <- hashes:
		// Mark all the fileDatas as known, but ensure we don't overflow our limits
		p.knownFds.Add(hashes...)
	case <-p.term:
		p.Log().Debug("Dropping transaction announcement", "count", len(hashes))
	}
}

// ReplyPooledFileDatasRLP is the response to RequestTxs.
func (p *Peer) ReplyPooledFileDatasRLP(id uint64, hashes []common.Hash, fds []rlp.RawValue) error {
	// Mark all the fileData as known, but ensure we don't overflow our limits
	p.knownFds.Add(hashes...)
	// Not packed into PooledFileDataResponse to avoid RLP decoding
	return p2p.Send(p.rw, PooledFileDatasMsg, &PooledFileDataRLPPacket{
		RequestId:                 id,
		PooledFileDataRLPResponse: fds,
		//PooledFileDataStatusResponse: status,
	})
}

// ReplyBlockHeadersRLP is the response to GetBlockHeaders.
func (p *Peer) ReplyBlockHeadersRLP(id uint64, headers []rlp.RawValue) error {
	return p2p.Send(p.rw, BlockHeadersMsg, &BlockHeadersRLPPacket{
		RequestId:               id,
		BlockHeadersRLPResponse: headers,
	})
}

// ReplyBlockBodiesRLP is the response to GetBlockBodies.
func (p *Peer) ReplyBlockBodiesRLP(id uint64, bodies []rlp.RawValue) error {
	// Not packed into BlockBodiesResponse to avoid RLP decoding
	return p2p.Send(p.rw, BlockBodiesMsg, &BlockBodiesRLPPacket{
		RequestId:              id,
		BlockBodiesRLPResponse: bodies,
	})
}

// ReplyReceiptsRLP is the response to GetReceipts.
func (p *Peer) ReplyReceiptsRLP(id uint64, receipts []rlp.RawValue) error {
	return p2p.Send(p.rw, ReceiptsMsg, &ReceiptsRLPPacket{
		RequestId:           id,
		ReceiptsRLPResponse: receipts,
	})
}

func (p *Peer) ReplyFileDatasMarshal(id uint64, fileDatas []*BantchFileData) []error {
	errs := make([]error, 0)	
	for _,bfd := range fileDatas {
			data,err := rlp.EncodeToBytes(bfd)
			if err != nil {
					log.Error("ReplyFileDatasMarshal---encode","err",err.Error())
			}
			err = p2p.Send(p.rw, ResFileDatasMsg, &FileDatasResponseRLPPacket{
				RequestId:           id,
				FileDatasResponse:   data,
			}) 
			errs = append(errs, err)
	}
	return errs
}


// RequestOneHeader is a wrapper around the header query functions to fetch a
// single header. It is used solely by the fetcher.
func (p *Peer) RequestOneHeader(hash common.Hash, sink chan *Response) (*Request, error) {
	p.Log().Debug("Fetching single header", "hash", hash)
	id := rand.Uint64()

	req := &Request{
		id:   id,
		sink: sink,
		code: GetBlockHeadersMsg,
		want: BlockHeadersMsg,
		data: &GetBlockHeadersPacket{
			RequestId: id,
			GetBlockHeadersRequest: &GetBlockHeadersRequest{
				Origin:  HashOrNumber{Hash: hash},
				Amount:  uint64(1),
				Skip:    uint64(0),
				Reverse: false,
			},
		},
	}
	if err := p.dispatchRequest(req); err != nil {
		return nil, err
	}
	return req, nil
}

// RequestHeadersByHash fetches a batch of blocks' headers corresponding to the
// specified header query, based on the hash of an origin block.
func (p *Peer) RequestHeadersByHash(origin common.Hash, amount int, skip int, reverse bool, sink chan *Response) (*Request, error) {
	p.Log().Debug("Fetching batch of headers", "count", amount, "fromhash", origin, "skip", skip, "reverse", reverse)
	id := rand.Uint64()

	req := &Request{
		id:   id,
		sink: sink,
		code: GetBlockHeadersMsg,
		want: BlockHeadersMsg,
		data: &GetBlockHeadersPacket{
			RequestId: id,
			GetBlockHeadersRequest: &GetBlockHeadersRequest{
				Origin:  HashOrNumber{Hash: origin},
				Amount:  uint64(amount),
				Skip:    uint64(skip),
				Reverse: reverse,
			},
		},
	}
	if err := p.dispatchRequest(req); err != nil {
		return nil, err
	}
	return req, nil
}

// RequestHeadersByNumber fetches a batch of blocks' headers corresponding to the
// specified header query, based on the number of an origin block.
func (p *Peer) RequestHeadersByNumber(origin uint64, amount int, skip int, reverse bool, sink chan *Response) (*Request, error) {
	p.Log().Debug("Fetching batch of headers", "count", amount, "fromnum", origin, "skip", skip, "reverse", reverse)
	id := rand.Uint64()

	req := &Request{
		id:   id,
		sink: sink,
		code: GetBlockHeadersMsg,
		want: BlockHeadersMsg,
		data: &GetBlockHeadersPacket{
			RequestId: id,
			GetBlockHeadersRequest: &GetBlockHeadersRequest{
				Origin:  HashOrNumber{Number: origin},
				Amount:  uint64(amount),
				Skip:    uint64(skip),
				Reverse: reverse,
			},
		},
	}
	if err := p.dispatchRequest(req); err != nil {
		return nil, err
	}
	return req, nil
}

// RequestBodies fetches a batch of blocks' bodies corresponding to the hashes
// specified.
func (p *Peer) RequestBodies(hashes []common.Hash, sink chan *Response) (*Request, error) {
	p.Log().Debug("Fetching batch of block bodies", "count", len(hashes))
	id := rand.Uint64()

	req := &Request{
		id:   id,
		sink: sink,
		code: GetBlockBodiesMsg,
		want: BlockBodiesMsg,
		data: &GetBlockBodiesPacket{
			RequestId:             id,
			GetBlockBodiesRequest: hashes,
		},
	}
	if err := p.dispatchRequest(req); err != nil {
		return nil, err
	}
	return req, nil
}

// RequestReceipts fetches a batch of transaction receipts from a remote node.
func (p *Peer) RequestReceipts(hashes []common.Hash, sink chan *Response) (*Request, error) {
	p.Log().Debug("Fetching batch of receipts", "count", len(hashes))
	id := rand.Uint64()

	req := &Request{
		id:   id,
		sink: sink,
		code: GetReceiptsMsg,
		want: ReceiptsMsg,
		data: &GetReceiptsPacket{
			RequestId:          id,
			GetReceiptsRequest: hashes,
		},
	}
	if err := p.dispatchRequest(req); err != nil {
		return nil, err
	}
	return req, nil
}

// RequestTxs fetches a batch of transactions from a remote node.
func (p *Peer) RequestTxs(hashes []common.Hash) error {
	p.Log().Debug("Fetching batch of transactions", "count", len(hashes))
	id := rand.Uint64()

	requestTracker.Track(p.id, p.version, GetPooledTransactionsMsg, PooledTransactionsMsg, id)
	return p2p.Send(p.rw, GetPooledTransactionsMsg, &GetPooledTransactionsPacket{
		RequestId:                    id,
		GetPooledTransactionsRequest: hashes,
	})
}

// RequestFileDatas fetches a batch of fileDatas from a remote node.
func (p *Peer) RequestFileDatas(hashes []common.Hash) error {
	p.Log().Debug("Fetching batch of fileDatas", "count", len(hashes))
	id := rand.Uint64()
	log.Info("RequestFileDatas----", "hash", hashes[0].String())
	requestTracker.Track(p.id, p.version, GetPooledFileDatasMsg, PooledFileDatasMsg, id)
	return p2p.Send(p.rw, GetPooledFileDatasMsg, &GetPooledFileDataPacket{
		RequestId:                 id,
		GetPooledFileDatasRequest: hashes,
	})
}

// StartRequestFileDatas implements downloader.Peer.
func (p *Peer) StartRequestFileDatas(hashes []common.Hash,sink chan *Response) (*Request, error) {
	p.Log().Debug("Fetching batch of fileData", "count", len(hashes))
	id := rand.Uint64()

	req := &Request{
		id:   id,
		sink: sink,
		code: ReqFileDatasMsg,
		want: ResFileDatasMsg,
		data: &GetFileDatasPacket{
			RequestId:          id,
			GetFileDatasRequest: hashes,
		},
	}
	if err := p.dispatchRequest(req); err != nil {
		return nil, err
	}
	return req, nil
}

// knownCache is a cache for known hashes.
type knownCache struct {
	hashes mapset.Set[common.Hash]
	max    int
}

// newKnownCache creates a new knownCache with a max capacity.
func newKnownCache(max int) *knownCache {
	return &knownCache{
		max:    max,
		hashes: mapset.NewSet[common.Hash](),
	}
}

// Add adds a list of elements to the set.
func (k *knownCache) Add(hashes ...common.Hash) {
	for k.hashes.Cardinality() > max(0, k.max-len(hashes)) {
		k.hashes.Pop()
	}
	for _, hash := range hashes {
		k.hashes.Add(hash)
	}
}

// Contains returns whether the given item is in the set.
func (k *knownCache) Contains(hash common.Hash) bool {
	return k.hashes.Contains(hash)
}

// Cardinality returns the number of elements in the set.
func (k *knownCache) Cardinality() int {
	return k.hashes.Cardinality()
}
