package les

import (
	"testing"

	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/node"
)

// TestULCThreeTrustedNodes tests syncing and receiving announces
// from trusted LES server nodes.
func TestULCTrustedNodes(t *testing.T) {

}

// TestULCTrustedAnnounceOnlyNodes tests syncing and receiving announces
// from trusted LES server nodes. These only send announces.
func TestULCTrustedAnnounceOnlyNodes(t *testing.T) {

}

// TestULCMixedNodes tests syncing and receiving announces in a network
// of trusted and untrusted LES server nodes. Only trusted ones are
// accepted.
func TestULCMixedNodes(t *testing.T) {

}

// newLESNode creates a LES test server.
func newLESNode(t *testing.T, config *eth.Config) *LesServer {
	ctx := &node.ServiceContext{}
	fullNode, err := eth.New(ctx, config)
	if err != nil {
		t.Fatalf("cannot create full node: %v", err)
	}
	lesServer, err := NewLesServer(fullNode, config)
	if err != nil {
		t.Fatalf("cannot create LES test server: %v", err)
	}
	fullNode.AddLesServer(lesServer)
	return lesServer
}
