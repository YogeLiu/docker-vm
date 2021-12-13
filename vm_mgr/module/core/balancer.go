package core

import (
	"sync"
	"sync/atomic"

	"chainmaker.org/chainmaker/vm-docker-go/vm_mgr/protocol"
)

// balancer strategy
const (
	// SRoundRobin every time, launch max num backend,
	// and then using round-robin to feed txs
	SRoundRobin = iota

	// SLeast first launch one backend, when reaching the 2/3 capacity, launch the second one
	// using the least connections' strategy to feed txs
	SLeast
)

var (
	maxPeer = 10
)

type PeerGroup struct {
	peers  []protocol.Peer
	curIdx uint64
	size   int
}

type BalancerImpl struct {
	peerMap  map[string]*PeerGroup
	strategy int
	mutex    sync.Mutex
}

func NewBalancerImpl() *BalancerImpl {
	balancer := &BalancerImpl{
		peerMap:  make(map[string]*PeerGroup),
		strategy: SLeast,
	}
	return balancer
}

// SetStrategy set balancer strategy
// two types of strategy available now: RoundRobin and Least
func (b *BalancerImpl) SetStrategy(_strategy int) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.strategy = _strategy
}

// AddPeer add new peer into balancer
// the store map is key(string) <-> peerGroup
// add new peer into slice of peerGroup
func (b *BalancerImpl) AddPeer(key string, peer protocol.Peer) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	group, ok := b.peerMap[key]

	if ok {
		prevSize := group.size
		group.peers[prevSize] = peer

	} else {
		group = &PeerGroup{
			peers:  make([]protocol.Peer, maxPeer),
			curIdx: 0,
			size:   0,
		}
		group.peers[0] = peer

		b.peerMap[key] = group
	}

	group.size++

	return nil
}

func (b *BalancerImpl) RemovePeer(key string, peerName string) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	//group, ok := b.peerMap[key]
	//
	//if !ok {
	//	return errors.New("peer doesn't exist")
	//}

	return nil

}

// GetPeer get next peer based on key
// based on different strategy, using different method to get next
// if peer list just have one peer, always return it
// when this peer reach limit, scheduler will generate new peer, limit May 2/3 of capacity or 4/5 of capacity
// then function will return this new peer always, because we should feed this new process -- using lease size function
// when return peer reach limit, then generate new peer, do above process
// eventually, peer list reach limit, and return peer also reach limit, using round-robin algorithm to return peer
func (b *BalancerImpl) GetPeer(key string) protocol.Peer {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	// peer list just have one peer, always return it
	// until returned peer reach limit, will generate new peer into balancer
	group := b.peerMap[key]
	if group.size == 1 {
		return group.peers[0]
	}

	// peer list contains 1 ~ maxPeer peers, return lease size peer
	// when returned peer reach limit, will generate new peer into balancer
	// until returned peer also reach limit, set strategy as SRoundRobin
	if group.size <= maxPeer && b.strategy == SLeast {
		return b.getNextPeerLeastSize(group)
	}

	// peer list is full, using Round-Robin to return peer
	// at this time, strategy == SRoundRobin
	// when one peer timeout, release this peer, reset strategy as SLeast
	// so using lease size to return peer
	return b.getNextPeerRoundRobin(group)

}

// getNextPeerRoundRobin get peer with Round Robin algorithm, which is equally get next peer
func (b *BalancerImpl) getNextPeerRoundRobin(group *PeerGroup) protocol.Peer {

	if group.size == 0 {
		return nil
	}

	if group.size == 1 {
		return group.peers[0]
	}

	// loop entire backends to find out an Alive backend
	next := b.nextIndex(group)
	l := len(group.peers) + next // start from next and move a full cycle
	for i := next; i < l; i++ {
		idx := i % len(group.peers) // take an index by modding with length
		// if we have an alive backend, use it and store if its not the original one
		if group.peers[idx].IsAlive() {
			if i != next {
				atomic.StoreUint64(&group.curIdx, uint64(idx)) // mark the current one
			}
			return group.peers[idx]
		}
	}
	return nil
}

// getNextPeerLeastSize get next peer with the smallest size
func (b *BalancerImpl) getNextPeerLeastSize(group *PeerGroup) protocol.Peer {

	nextIdx := 0
	minSize := processWaitingQueueSize + 1

	for i := 0; i < group.size; i++ {
		curPeer := group.peers[i]
		if curPeer.Size() < minSize {
			if group.peers[i].IsAlive() {
				minSize = curPeer.Size()
				nextIdx = i
			}
		}
	}
	return group.peers[nextIdx]
}

func (b *BalancerImpl) nextIndex(group *PeerGroup) int {
	return int(atomic.AddUint64(&group.curIdx, uint64(1)) % uint64(group.size))
}
