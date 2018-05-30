package les

import (
	"crypto/rand"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

func TestULCSyncWithOnePeer(t *testing.T) {
	f := newFullPeerPair(t, 1, 4, testChainGen)
	ulcConfig := &eth.ULCConfig{
		MinTrustedFraction: 100,
		TrustedNodes:       []string{f.ID.String()},
	}

	l := newLightPeer(t, ulcConfig)

	if reflect.DeepEqual(f.PM.blockchain.CurrentHeader().Hash(), l.PM.blockchain.CurrentHeader().Hash()) {
		t.Fatal("blocks are equal")
	}

	fPeer, _, err := connectPeers(f, l, 2)
	if err != nil {
		t.Fatal(err)
	}

	l.PM.synchronise(fPeer)

	if !reflect.DeepEqual(f.PM.blockchain.CurrentHeader().Hash(), l.PM.blockchain.CurrentHeader().Hash()) {
		t.Fatal("sync doesn't work")
	}
}

func TestULCReceiveAnnounce(t *testing.T) {
	f := newFullPeerPair(t, 1, 4, testChainGen)
	ulcConfig := &eth.ULCConfig{
		MinTrustedFraction: 100,
		TrustedNodes:       []string{f.ID.String()},
	}

	key, err := crypto.GenerateKey()
	ID := discover.PubkeyID(&key.PublicKey)
	l := newLightPeer(t, ulcConfig)
	l.ID = ID

	fPeer, lPeer, err := connectPeers(f, l, 2)
	if err != nil {
		t.Fatal(err)
	}

	l.PM.synchronise(fPeer)

	//check that the sync is finished correctly
	if !reflect.DeepEqual(f.PM.blockchain.CurrentHeader().Hash(), l.PM.blockchain.CurrentHeader().Hash()) {
		t.Fatal("sync doesn't work")
	}

	l.PM.peers.lock.Lock()
	if len(l.PM.peers.peers) == 0 {
		t.Fatal("peer list should not be empty")
	}
	l.PM.peers.lock.Unlock()

	//send a signed announce message(payload doesn't matter)
	announce := announceData{}
	announce.sign(key)
	lPeer.SendAnnounce(announce)

	l.PM.peers.lock.Lock()
	if len(l.PM.peers.peers) == 0 {
		t.Fatal("peer list after receiving message should not be empty")
	}
	l.PM.peers.lock.Unlock()
}

//TODO(b00ris) it's failing test. it should be fixed by https://github.com/status-im/go-ethereum/issues/51
func TestULCShouldNotSyncWithTwoPeersOneHaveEmptyChain(t *testing.T) {
	t.Skip()
	f1 := newFullPeerPair(t, 1, 4, testChainGen)
	f2 := newFullPeerPair(t, 3, 0, nil)
	ulcConf := &ulc{minTrustedFraction: 100, trustedKeys: make(map[string]struct{})}
	ulcConf.trustedKeys[f1.ID.String()] = struct{}{}
	ulcConf.trustedKeys[f2.ID.String()] = struct{}{}
	ulcConfig := &eth.ULCConfig{
		MinTrustedFraction: 100,
		TrustedNodes:       []string{f1.ID.String(), f2.ID.String()},
	}
	l := newLightPeer(t, ulcConfig)
	l.PM.ulc.minTrustedFraction = 100

	fPeer1, lPeer1, err := connectPeers(f1, l, 2)
	fPeer2, lPeer2, err := connectPeers(f2, l, 2)
	if err != nil {
		t.Fatal(err)
	}

	_, _ = lPeer1, lPeer2

	l.PM.synchronise(fPeer1)
	l.PM.synchronise(fPeer2)

	time.Sleep(time.Second)
	if l.PM.blockchain.CurrentHeader() != nil {
		t.Fatal("Should be empty")
	}
}

type pairPeer struct {
	Name string
	ID   discover.NodeID
	PM   *ProtocolManager
}

func connectPeers(full, light pairPeer, version int) (*peer, *peer, error) {
	// Create a message pipe to communicate through
	app, net := p2p.MsgPipe()

	peerLight := full.PM.newPeer(version, NetworkId, p2p.NewPeer(light.ID, light.Name, nil), net)
	peerFull := light.PM.newPeer(version, NetworkId, p2p.NewPeer(full.ID, full.Name, nil), app)

	// Start the peerLight on a new thread
	errc1 := make(chan error, 1)
	errc2 := make(chan error, 1)
	go func() {
		select {
		case light.PM.newPeerCh <- peerFull:
			errc1 <- light.PM.handle(peerFull)
		case <-light.PM.quitSync:
			errc1 <- p2p.DiscQuitting
		}
	}()
	go func() {
		select {
		case full.PM.newPeerCh <- peerLight:
			errc2 <- full.PM.handle(peerLight)
		case <-full.PM.quitSync:
			errc2 <- p2p.DiscQuitting
		}
	}()

	select {
	case <-time.After(time.Millisecond * 100):
	case err := <-errc1:
		return nil, nil, fmt.Errorf("peerLight handshake error: %v", err)
	case err := <-errc2:
		return nil, nil, fmt.Errorf("peerFull handshake error: %v", err)
	}

	return peerFull, peerLight, nil
}

func newFullPeerPair(t *testing.T, index int, numberOfblocks int, chainGen func(int, *core.BlockGen)) pairPeer {
	db := ethdb.NewMemDatabase()

	pmFull := newTestProtocolManagerMust(t, false, numberOfblocks, chainGen, nil, nil, db, nil)

	peerPairFull := pairPeer{
		Name: "full node",
		PM:   pmFull,
	}
	rand.Read(peerPairFull.ID[:])
	return peerPairFull
}

func newLightPeer(t *testing.T, ulcConfig *eth.ULCConfig) pairPeer {
	peers := newPeerSet()
	dist := newRequestDistributor(peers, make(chan struct{}))
	rm := newRetrieveManager(peers, dist, nil)
	ldb := ethdb.NewMemDatabase()

	odr := NewLesOdr(ldb, light.NewChtIndexer(ldb, true), light.NewBloomTrieIndexer(ldb, true), eth.NewBloomIndexer(ldb, light.BloomTrieFrequency), rm)

	pmLight := newTestProtocolManagerMust(t, true, 0, nil, peers, odr, ldb, ulcConfig)
	peerPairLight := pairPeer{
		Name: "ulc node",
		PM:   pmLight,
	}
	rand.Read(peerPairLight.ID[:])

	return peerPairLight
}
