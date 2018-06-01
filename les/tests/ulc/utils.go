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
	"fmt"

	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/les"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/simulations"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/rpc"
)

// testNode is a wrapper around a node.Service and provides logging
// for debugging.
type testNode struct {
	service node.Service
}

func (t *testNode) Protocols() []p2p.Protocol {
	log.Trace("calling Protocols()", "protocols", t.service.Protocols())
	return t.service.Protocols()
}

func (t *testNode) APIs() []rpc.API {
	log.Trace("calling APIs()")
	return t.service.APIs()
}

func (t *testNode) Start(server *p2p.Server) error {
	log.Trace("calling Start()")
	return t.service.Start(server)
}

func (t *testNode) Stop() error {
	log.Trace("calling Stop()")
	return t.service.Stop()
}

// newEthConfig generates the basic configuration for the test servers.
func newEthConfig(ulc *eth.ULCConfig) *eth.Config {
	cfg := eth.DefaultConfig
	cfg.NetworkId = 3
	cfg.LightServ = 50
	if ulc != nil {
		cfg.ULC = ulc
	}
	return &cfg
}

// newLesServerService creates a LES server as node service.
func newLesServerService(ctx *adapters.ServiceContext, testMode string) (node.Service, error) {
	log.Info("new LES server service", "id", ctx.Config.Name)
	config := newEthConfig(nil)
	fullNode, err := eth.New(ctx.NodeContext, config)
	if err != nil {
		return nil, fmt.Errorf("cannot create full node: %v", err)
	}
	lesServer, err := les.NewLesServer(fullNode, config)
	if err != nil {
		return nil, fmt.Errorf("cannot create LES server service: %v", err)
	}
	fullNode.AddLesServer(lesServer)
	return &testNode{service: fullNode}, nil
}

// newULCService creates a ULC client as node service.
func newULCService(ctx *adapters.ServiceContext, testMode string) (node.Service, error) {
	log.Info("new ULC service", "id", ctx.Config.Name)
	// Retrieve infos about the LES server nodes.
	infoLes01, err := simulations.DefaultClient.GetNode("les01")
	if err != nil {
		return nil, fmt.Errorf("cannot retrieve info of LES server service 01: %v", err)
	}
	infoLes02, err := simulations.DefaultClient.GetNode("les02")
	if err != nil {
		return nil, fmt.Errorf("cannot retrieve info of LES server service 02: %v", err)
	}
	infoLes03, err := simulations.DefaultClient.GetNode("les03")
	if err != nil {
		return nil, fmt.Errorf("cannot retrieve info of LES server service 03: %v", err)
	}
	// Create configuration and start ULC.
	config := newEthConfig(&eth.ULCConfig{
		MinTrustedFraction: 50,
		TrustedNodes: []string{
			infoLes01.ID,
			infoLes02.ID,
			infoLes03.ID,
		}})
	lightEthereum, err := les.New(ctx.NodeContext, config)
	if err != nil {
		return nil, fmt.Errorf("cannot create ULC service: %v", err)
	}
	return &testNode{service: lightEthereum}, nil
}
