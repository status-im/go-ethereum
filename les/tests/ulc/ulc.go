// Copyright 2018 The go-ethereum Authors
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

package main

import (
	"flag"
	"net/http"
	"os"

	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/les"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/simulations"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
)

// Flogs of the ULC simulation.
var (
	testSim = flag.String("testsim", "trusted", `test simulation to run (one of "trusted", "announce" or "mixed")`)
)

// main runs the test simulations based on the flags.
func main() {
	flag.Parse()
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlTrace, log.StreamHandler(os.Stderr, log.TerminalFormat(false))))

	var network *simulations.Network

	switch *testSim {
	case "trusted":
		network = testTrustedServerNodes()
	case "announce":
		network = testTrustedAnnounceOnlyServerNodes()
	case "mixed":
		network = testMixedServerNodes()
	default:
		log.Crit("invalid test simulation", "testsim", *testSim)
	}

	log.Info("starting three trusted nodes simulation on 0.0.0.0:8888...")
	if err := http.ListenAndServe(":8888", simulations.NewServer(network)); err != nil {
		log.Crit("error starting simulation server", "err", err)
	}
}

// ----------
// TEST SIMULATIONS
// ----------

// testTrustedServerNodes tests syncing and receiving announces
// from three trusted LES server nodes.
func testTrustedServerNodes() *simulations.Network {
	services := map[string]adapters.ServiceFunc{
		"full-one": func(ctx *adapters.ServiceContext) (node.Service, error) {
			return newLesServerService(ctx.NodeContext, newEthConfig(nil)), nil
		},
		"full-two": func(ctx *adapters.ServiceContext) (node.Service, error) {
			return newLesServerService(ctx.NodeContext, newEthConfig(nil)), nil
		},
		"full-three": func(ctx *adapters.ServiceContext) (node.Service, error) {
			return newLesServerService(ctx.NodeContext, newEthConfig(nil)), nil
		},
		"ulc": func(ctx *adapters.ServiceContext) (node.Service, error) {
			ulcConfig := &eth.ULCConfig{
				MinTrustedFraction: 50,
				TrustedNodes:       []string{"full-one", "full-two", "full-three"},
			}
			return newULCService(ctx.NodeContext, newEthConfig(ulcConfig)), nil
		},
	}
	adapters.RegisterServices(services)
	adapter := adapters.NewSimAdapter(services)

	return simulations.NewNetwork(adapter, &simulations.NetworkConfig{
		DefaultService: "ulc",
	})
}

// testTrustedAnnounceOnlyServerNodes tests syncing and receiving announces
// from trusted LES server nodes. These only send announces.
func testTrustedAnnounceOnlyServerNodes() *simulations.Network {
	return nil
}

// testMixedServerNodes tests syncing and receiving announces in a network
// of trusted and untrusted LES server nodes. Only trusted ones are
// accepted.
func testMixedServerNodes() *simulations.Network {
	return nil
}

// ----------
// HELPERS
// ----------

// newEthConfig generates the basic configuration for the test servers.
func newEthConfig(ulc *eth.ULCConfig) *eth.Config {
	cfg := eth.DefaultConfig
	if ulc != nil {
		cfg.ULC = ulc
	}
	return &cfg
}

// newLesServerService creates a LES server as node service.
func newLesServerService(ctx *node.ServiceContext, config *eth.Config) node.Service {
	fullNode, err := eth.New(ctx, config)
	if err != nil {
		log.Crit("cannot create full node", "err", err)
	}
	lesServer, err := les.NewLesServer(fullNode, config)
	if err != nil {
		log.Crit("cannot create LES server service", "err", err)
	}
	fullNode.AddLesServer(lesServer)
	return fullNode
}

// newULCService creates a ULC client as node service.
func newULCService(ctx *node.ServiceContext, config *eth.Config) node.Service {
	lightEthereum, err := les.New(ctx, config)
	if err != nil {
		log.Crit("cannot create ULC service", "err", err)
	}
	return lightEthereum

}
