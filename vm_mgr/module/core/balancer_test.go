package core

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"chainmaker.org/chainmaker/vm-docker-go/vm_mgr/protocol"
)

func TestBalancerImpl_AddPeer(t *testing.T) {
	balancer := NewBalancerImpl()

	key := "test"
	peer0 := &PeerImplement{}

	_ = balancer.AddPeer(key, peer0)
	assert.Equal(t, 1, balancer.peerMap[key].size)
	assert.Equal(t, uint64(0), balancer.peerMap[key].curIdx)

	peer1 := &PeerImplement{}
	_ = balancer.AddPeer(key, peer1)
	assert.Equal(t, 2, balancer.peerMap[key].size)
	assert.Equal(t, uint64(0), balancer.peerMap[key].curIdx)

	key2 := "test2"
	peer2 := &PeerImplement{}
	_ = balancer.AddPeer(key2, peer2)
	assert.Equal(t, 1, balancer.peerMap[key2].size)
	assert.Equal(t, uint64(0), balancer.peerMap[key2].curIdx)

}

func TestBalancerImpl_GetPeer(t *testing.T) {
	maxPeer = 3
	balancer := NewBalancerImpl()

	key := "test"

	peer0 := &PeerImplement{}
	_ = balancer.AddPeer(key, peer0)

	result := balancer.GetPeer(key)
	assert.Equal(t, peer0, result)

	peer1 := &PeerImplement{}
	_ = balancer.AddPeer(key, peer1)
	result = balancer.GetPeer(key)
	assert.Equal(t, peer0, result)

	peer2 := &PeerImplement{}
	_ = balancer.AddPeer(key, peer2)
	result = balancer.GetPeer(key)
	assert.Equal(t, peer0, result)

	peer0.setSize(1)
	result = balancer.GetPeer(key)
	assert.Equal(t, peer1, result)

	peer1.setSize(1)
	result = balancer.GetPeer(key)
	assert.Equal(t, peer2, result)

	balancer.SetStrategy(SRoundRobin)
	result = balancer.GetPeer(key)
	assert.Equal(t, peer1, result)

	result = balancer.GetPeer(key)
	assert.Equal(t, peer2, result)

	result = balancer.GetPeer(key)
	assert.Equal(t, peer0, result)

}

func TestBalancerImpl_getNextPeerLeastSize(t *testing.T) {

}

func TestBalancerImpl_getNextPeerRoundRobin(t *testing.T) {
	balancer := NewBalancerImpl()

	peerGroup := &PeerGroup{
		peers:  make([]protocol.Peer, maxPeer),
		curIdx: 0,
		size:   0,
	}

	result := balancer.getNextPeerRoundRobin(peerGroup)
	assert.Nil(t, result)

	peer1 := &PeerImplement{}
	peerGroup.peers[0] = peer1
	peerGroup.size++
	result = balancer.getNextPeerRoundRobin(peerGroup)
	assert.Equal(t, peer1, result)

	peer2 := &PeerImplement{}
	peerGroup.peers[1] = peer2
	peerGroup.size++
	result = balancer.getNextPeerRoundRobin(peerGroup)
	assert.Equal(t, peer2, result)

	peer3 := &PeerImplement{}
	peerGroup.peers[2] = peer3
	peerGroup.size++

	peer4 := &PeerImplement{}
	peerGroup.peers[3] = peer4
	peerGroup.size++

	peer5 := &PeerImplement{}
	peerGroup.peers[4] = peer5
	peerGroup.size++

	result = balancer.getNextPeerRoundRobin(peerGroup)
	assert.Equal(t, peer3, result)

	result = balancer.getNextPeerRoundRobin(peerGroup)
	assert.Equal(t, peer4, result)

	result = balancer.getNextPeerRoundRobin(peerGroup)
	assert.Equal(t, peer5, result)

	result = balancer.getNextPeerRoundRobin(peerGroup)
	assert.Equal(t, peer1, result)

	result = balancer.getNextPeerRoundRobin(peerGroup)
	assert.Equal(t, peer1, result)

	result = balancer.getNextPeerRoundRobin(peerGroup)
	assert.Equal(t, peer2, result)

	result = balancer.getNextPeerRoundRobin(peerGroup)
	assert.Equal(t, peer3, result)

}

func TestBalancerImpl_nextIndex(t *testing.T) {

	balancer := NewBalancerImpl()

	peerGroup := &PeerGroup{
		peers:  make([]protocol.Peer, maxPeer),
		curIdx: 0,
		size:   0,
	}

	peer0 := &PeerImplement{}
	peerGroup.peers[0] = peer0
	peerGroup.size++

	result := balancer.nextIndex(peerGroup)
	assert.Equal(t, 0, result)

	peer1 := &PeerImplement{}
	peerGroup.peers[1] = peer1
	peerGroup.size++

	result = balancer.nextIndex(peerGroup)
	assert.Equal(t, 0, result)

	result = balancer.nextIndex(peerGroup)
	assert.Equal(t, 1, result)

}

type PeerImplement struct {
	size int
}

func (p *PeerImplement) IsAlive() bool {
	return true
}

func (p *PeerImplement) Size() int {
	return p.size
}

func (p *PeerImplement) AddTx(tx int) {

}

func (p *PeerImplement) setSize(_size int) {
	p.size = _size
}
