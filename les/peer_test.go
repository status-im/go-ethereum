package les

import (
	"crypto/rand"
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

//ulc connects to trusted peer and mark it
func TestPeer_Handshake_AnnounceTypeSignedAndOnlyAnnounceRequests_ForTrustedPeers_PeerInTrusted_Success(t *testing.T) {
	var id discover.NodeID
	rand.Read(id[:])

	//current ulc server
	s := generateLesServer()
	s.ulc = newULC(&eth.ULCConfig{
		TrustedNodes: []string{id.String()},
	})
	s.ulc.trusted[id.String()] = struct{}{}

	//peer to connect(on ulc side)
	p := peer{
		Peer:    p2p.NewPeer(id, "test peer", []p2p.Cap{}),
		version: protocol_version,
		rw: &rwStub{
			WriteHook: func(recvList keyValueList) {
				//checking that ulc sends to peer allowedRequests=onlyAnnounceRequests and announceType = announceTypeSigned
				recv := recvList.decode()
				var a, reqType uint64
				err := recv.get("allowedRequests", &a)
				if err != nil {
					t.Fatal(err)
				}
				if a != onlyAnnounceRequests {
					t.Fatal("Expected onlyAnnounceRequests")
				}
				err = recv.get("announceType", &reqType)
				if err != nil {
					t.Fatal(err)
				}

				if reqType != announceTypeSigned {
					t.Fatal("Expected announceTypeSigned")
				}
			},
		},
		network: test_networkid,
	}

	err := p.Handshake(td, hash, headNum, genesis, s)
	if err != nil {
		t.Fatalf("Handshake error: %s", err)
	}

	if p.announceType != announceTypeSigned {
		t.Fatal("Incorrect announceType")
	}
}

func TestPeer_Handshake_AnnounceTypeSigned_ForTrustedPeers_PeerNotInTrusted_Fail(t *testing.T) {
	var id discover.NodeID
	rand.Read(id[:])

	p := peer{
		Peer:    p2p.NewPeer(id, "test peer", []p2p.Cap{}),
		version: protocol_version,
		rw: &rwStub{
			WriteHook: func(recvList keyValueList) {
				//checking that ulc sends to peer allowedRequests=onlyAnnounceRequests and announceType = announceTypeSigned
				recv := recvList.decode()
				var a, reqType uint64
				err := recv.get("allowedRequests", &a)
				if err != nil {
					t.Fatal(err)
				}

				if a != noRequests {
					t.Fatal("Expected noRequests")
				}
				err = recv.get("announceType", &reqType)
				if err != nil {
					t.Fatal(err)
				}

				if reqType == announceTypeSigned {
					t.Fatal("Expected not announceTypeSigned")
				}
			},
		},
		network: test_networkid,
	}

	s := generateLesServer()
	s.ulc = newULC(&eth.ULCConfig{
		TrustedNodes: []string{},
	})

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
	ReadHook  func(l keyValueList) keyValueList
	WriteHook func(l keyValueList)
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

	if s.ReadHook != nil {
		payload = s.ReadHook(payload)
	}

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

	if s.WriteHook != nil {
		s.WriteHook(recvList)
	}

	return nil
}
