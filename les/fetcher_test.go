package les

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"
	"testing"
)

func TestFetcher_ULC_Peer_Selector(t *testing.T) {
	ftn1 := &fetcherTreeNode{
		hash: common.StringToHash("1"),
		td:   big.NewInt(1),
	}
	ftn2 := &fetcherTreeNode{
		hash:   common.StringToHash("2"),
		td:     big.NewInt(2),
		parent: ftn1,
	}
	ftn3 := &fetcherTreeNode{
		hash:   common.StringToHash("3"),
		td:     big.NewInt(3),
		parent: ftn2,
	}
	lf := lightFetcher{
		pm: &ProtocolManager{
			server: &LesServer{
				ulc: &ulc{
					trusted: map[string]struct{}{
						"peer1": {},
						"peer2": {},
						"peer3": {},
						"peer4": {},
					},
					minTrustedFraction: 70,
				},
			},
		},
		maxConfirmedTd: ftn1.td,
		peers: map[*peer]*fetcherPeerInfo{
			&peer{
				id: "peer1",
			}: {
				nodeByHash: map[common.Hash]*fetcherTreeNode{
					ftn1.hash: ftn1,
					ftn2.hash: ftn2,
				},
			},
			&peer{
				id: "peer2",
			}: {
				nodeByHash: map[common.Hash]*fetcherTreeNode{
					ftn1.hash: ftn1,
					ftn2.hash: ftn2,
				},
			},
			&peer{
				id: "peer3",
			}: {
				nodeByHash: map[common.Hash]*fetcherTreeNode{
					ftn1.hash: ftn1,
					ftn2.hash: ftn2,
					ftn3.hash: ftn3,
				},
			},
			&peer{
				id: "peer4",
			}: {
				nodeByHash: map[common.Hash]*fetcherTreeNode{
					ftn1.hash: ftn1,
				},
			},
		},
		chain: &lightChainStub{
			tds: map[common.Hash]*big.Int{},
			headers: map[common.Hash]*types.Header{
				ftn1.hash: {},
				ftn2.hash: {},
				ftn3.hash: {},
			},
		},
	}
	bestHash, bestAmount, bestTD, sync := lf.itFindBestValuesForULC()

	if bestTD == nil {
		t.Fatal("Empty result")
	}
	if bestTD.Cmp(ftn2.td) != 0 {
		t.Fatal("bad td", bestTD)
	}
	if bestHash != ftn2.hash {
		t.Fatal("bad hash", bestTD)
	}

	_, _ = bestAmount, sync
}

type lightChainStub struct {
	BlockChain
	tds     map[common.Hash]*big.Int
	headers map[common.Hash]*types.Header
}

func (l *lightChainStub) GetHeader(hash common.Hash, number uint64) *types.Header {
	if h, ok := l.headers[hash]; ok {
		return h
	}

	return nil
}

func (l *lightChainStub) LockChain()   {}
func (l *lightChainStub) UnlockChain() {}

func (l *lightChainStub) GetTd(hash common.Hash, number uint64) *big.Int {
	if td, ok := l.tds[hash]; ok {
		return td
	}
	return nil
}
