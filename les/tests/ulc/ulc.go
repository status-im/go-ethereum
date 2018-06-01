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
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/simulations"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
)

// Flogs of the ULC simulation.
var (
	testMode    = flag.String("test", "trusted", `test to run (one of "trusted", "announce", "mixed")`)
	adapterType = flag.String("adapter", "sim", `node adapter to use (one of "sim", "exec" or "docker")`)
)

// main runs the test simulation node based on the flags and the node ID.
func main() {
	// Flag and logging.
	flag.Parse()
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlTrace, log.StreamHandler(os.Stderr, log.TerminalFormat(false))))

	// Create service map and register the services.
	services := map[string]adapters.ServiceFunc{
		"les": func(ctx *adapters.ServiceContext) (node.Service, error) {
			if ctx.Config.Name == "ulc" {
				// Node 01 is always the ULC node.
				return newULCService(ctx, *testMode)
			}
			// LES server nodes.
			return newLesServerService(ctx, *testMode)
		},
	}
	adapters.RegisterServices(services)

	// Create the NodeAdapter.
	var adapter adapters.NodeAdapter

	switch *adapterType {
	case "sim":
		log.Info("using sim adapter")
		adapter = adapters.NewSimAdapter(services)
	case "exec":
		tmpdir, err := ioutil.TempDir("", "ulc-testing")
		if err != nil {
			log.Crit("error creating temp dir", "err", err)
		}
		defer os.RemoveAll(tmpdir)
		log.Info("using exec adapter", "tmpdir", tmpdir)
		adapter = adapters.NewExecAdapter(tmpdir)
	case "docker":
		log.Info("using docker adapter")
		var err error
		adapter, err = adapters.NewDockerAdapter()
		if err != nil {
			log.Crit("error creating docker adapter", "err", err)
		}
	default:
		log.Crit(fmt.Sprintf("unknown node adapter %q", *adapterType))
	}

	// Start the HTTP API of the simulation server.
	log.Info("starting simulation server on 0.0.0.0:8888...")
	network := simulations.NewNetwork(adapter, &simulations.NetworkConfig{
		DefaultService: "les",
	})
	if err := http.ListenAndServe(":8888", simulations.NewServer(network)); err != nil {
		log.Crit("error starting simulation server", "err", err)
	}
}
