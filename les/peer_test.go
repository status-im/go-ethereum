package les

import (
	"crypto/rand"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/les/flowcontrol"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/rlp"
	"math/big"
	"testing"
)

const (
	test_networkid   = 10
	protocol_version = 2123
)

var (
	hash    = common.StringToHash("some string")
	genesis = common.StringToHash("genesis hash")
	headNum = uint64(1234)
	td      = big.NewInt(123)
)

func TestPeer_Handshake_AnnounceTypeSigned_ForTrustedPeers_PeerInTrusted_Success(t *testing.T) {
	var id discover.NodeID
	rand.Read(id[:])

	p := peer{
		Peer:    p2p.NewPeer(id, "test peer", []p2p.Cap{}),
		version: protocol_version,
		rw:      &rwStub{},
		network: test_networkid,
	}
	s := generateLesServer()
	s.ulc = newULC(&eth.ULCConfig{
		TrustedNodes: []string{id.String()},
	})
	s.ulc.trusted[id.String()] = struct{}{}

	err := p.Handshake(td, hash, headNum, genesis, s)
	if err != nil {
		t.Fatalf("Handshake error: %s", err)
	}
	if p.announceType != announceTypeSigned {
		t.Fatal("Incorrect announceType")
	}
}

func TestPeer_Handshake_AnnounceTypeSigned_ForTrustedPeers_PeerNotInTrusted_Fail(t *testing.T) {
	p := peer{
		version: protocol_version,
		rw:      &rwStub{},
		network: test_networkid,
	}

	s := generateLesServer()

	err := p.Handshake(td, hash, headNum, genesis, s)
	if err != nil {
		t.Fatal(err)
	}
	if p.announceType == announceTypeSigned {
		t.Fatal("Incorrect announceType")
	}
}

func generateLesServer() *LesServer {
	s := &LesServer{
		defParams: &flowcontrol.ServerParams{
			BufLimit:    uint64(300000000),
			MinRecharge: uint64(50000),
		},
		fcManager: flowcontrol.NewClientManager(1, 2, 3),
		fcCostStats: &requestCostStats{
			stats: make(map[uint64]*linReg, len(reqList)),
		},
	}
	for _, code := range reqList {
		s.fcCostStats.stats[code] = &linReg{cnt: 100}
	}
	return s
}

type rwStub struct {
	WriteAssert func(m p2p.Msg)
}

func (s *rwStub) ReadMsg() (p2p.Msg, error) {
	payload := keyValueList{}
	payload = payload.add("protocolVersion", uint64(protocol_version))
	payload = payload.add("networkId", uint64(test_networkid))
	payload = payload.add("headTd", td)
	payload = payload.add("headHash", hash)
	payload = payload.add("headNum", headNum)
	payload = payload.add("genesisHash", genesis)

	payload = payload.add("serveHeaders", nil)
	payload = payload.add("serveChainSince", uint64(0))
	payload = payload.add("serveStateSince", uint64(0))
	payload = payload.add("txRelay", nil)
	payload = payload.add("flowControl/BL", uint64(300000000))
	payload = payload.add("flowControl/MRR", uint64(50000))

	size, p, err := rlp.EncodeToReader(payload)
	if err != nil {
		return p2p.Msg{}, err
	}

	return p2p.Msg{
		Size:    uint32(size),
		Payload: p,
	}, nil
}
func (s *rwStub) WriteMsg(m p2p.Msg) error {
	recvList := keyValueList{}
	if err := m.Decode(&recvList); err != nil {
		return err
	}

	recv := recvList.decode()
	fmt.Println("WriteMsg: ", recv)
	for k, v := range recv {
		fmt.Println(k, v)
	}
	return nil
}
