package les

import (
	"testing"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/simulations"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/p2p"
	"fmt"
	"time"
	"github.com/ethereum/go-ethereum/eth"
	"sync"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/les/flowcontrol"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum/go-ethereum/eth/gasprice"
	"github.com/ethereum/go-ethereum/core/bloombits"
	"github.com/ethereum/go-ethereum/core/rawdb"
)

func TestSim(t *testing.T)  {
	var a []*LesServer
	var cli *LightEthereum
	services := map[string]adapters.ServiceFunc{
		"server": func(ctx *adapters.ServiceContext) (node.Service, error) {
			fmt.Println("Staret lesServer")
			db := ethdb.NewMemDatabase()

			srv,err:=NewTestLesServer(db, 4, testChainGen, nil)
			if err!=nil {
				return nil, err
			}
			a=append(a,srv)
			return &stubLesServer{srv}, nil
		},
		"ulc": func(ctx *adapters.ServiceContext) (node.Service, error) {
			fmt.Println("Staret lesCli")
			var err error
			cli, err = NewTestsLightEthereum(ctx.NodeContext, &eth.Config{
			})
			if err!= nil {
				t.Log("err: ", err)
			}
			t.Log("bc", cli.blockchain.CurrentHeader().Number)
			t.Log("bc", cli.blockchain.CurrentHeader().Hash())

			return cli, nil
		},
	}
	//adapters.RegisterServices(services)

	adapter := adapters.NewSimAdapter(services)

	//log.Info("starting simulation server on 0.0.0.0:8888...")
	network := simulations.NewNetwork(adapter, &simulations.NetworkConfig{
		DefaultService: "server",
	})
	n, err:=network.NewNodeWithConfig(adapters.RandomNodeConfig())
	n2, err:=network.NewNodeWithConfig(adapters.RandomNodeConfig())

	ulcConfig:=adapters.RandomNodeConfig()
	ulcConfig.Services=[]string{"ulc"}
	nULC, err:=network.NewNodeWithConfig(ulcConfig)

	//nULC.Up=true
	//fmt.Println("ULC start:", err)
	//fmt.Println("------------------------")
	//
	//
	//
	err = network.Start(n.ID())
	err = network.Start(n2.ID())
	err = network.Start(nULC.ID())

	time.Sleep(time.Second)
	t.Log(n.Up)
	t.Log(n2.Up)
	t.Log(nULC.Up)
	err=network.Connect(n.ID(), n2.ID())
	t.Log(err)
	err=network.Connect(n.ID(), nULC.ID())
	t.Log("nUlc connect", err)
	//
	time.Sleep(time.Second)
	//clULC,err :=nULC.Client()
	//t.Log(err)
	//t.Log(clULC.SupportedModules())
	//clNode,err :=n.Client()
	//
	//var result map[string]string
	//ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	//defer cancel()
	//err = clNode.CallContext(ctx, &result, "eth_blockNumber")
	//t.Log(err)
	//t.Log(result)
	//
	//
	//ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	//defer cancel()
	//err = clULC.CallContext(ctx, &result, "eth_blockNumber")
	//t.Log(err)
	//t.Log(result)
	//
	//
	for i:=range a {
		t.Log(i, " : ", a[i].protocolManager.blockchain.CurrentHeader().Number,  a[i].protocolManager.blockchain.CurrentHeader().Hash())
	}
	//
	//t.Log("ULC", cli)
	//t.Log("ULC", cli.protocolManager)
	t.Log("ULC", cli.protocolManager.blockchain.CurrentHeader().Number)
	t.Log("ULC", cli.protocolManager.blockchain.CurrentHeader().Hash())
	//if err := http.ListenAndServe(":8888", simulations.NewServer(network)); err != nil {
	//	fmt.Println("Err:", err)
	//	//log.Crit("error starting simulation server", "err", err)
	//}
	//sim:=simulations.NewSimulation(network)
	//sim.Run()

}

type stubLesServer struct {
	*LesServer
}

func (s *stubLesServer) APIs() []rpc.API {
	return []rpc.API{}
}

func (s *stubLesServer) Start(peer *p2p.Server) error {
	s.LesServer.Start(peer)
	return nil
}

func (s *stubLesServer) Stop() error {
	s.LesServer.Stop()
	return nil
}
func (s *stubLesServer) Protocols() []p2p.Protocol {
	return s.LesServer.Protocols()
}


func NewTestLesServer(db ethdb.Database, blocks int, generator func(int, *core.BlockGen), ulcConfig *eth.ULCConfig) (*LesServer, error) {
	gspec  := core.Genesis{
		Config: params.TestChainConfig,
		Alloc:  core.GenesisAlloc{testBankAddress: {Balance: testBankFunds}},
	}
	genesis := gspec.MustCommit(db)
	evmux  := new(event.TypeMux)
	engine := ethash.NewFaker()


////chain init
	blockchain, _ := core.NewBlockChain(db, nil, gspec.Config, engine, vm.Config{})
	chtIndexer := light.NewChtIndexer(db, false)
//	chtIndexer.Start(blockchain)
//
	bbtIndexer := light.NewBloomTrieIndexer(db, false)
//
//	bloomIndexer := eth.NewBloomIndexer(db, params.BloomBitsBlocks)
//	bloomIndexer.AddChildIndexer(bbtIndexer)
//	bloomIndexer.Start(blockchain)

	gchain, _ := core.GenerateChain(gspec.Config, genesis, ethash.NewFaker(), db, blocks, generator)
	if _, err := blockchain.InsertChain(gchain); err != nil {
		panic(err)
	}




	quitSync := make(chan struct{})
	pm, err := NewProtocolManager(
		gspec.Config,
		false,
		ServerProtocolVersions,
		123, //some network id
		evmux,
		engine,
		newPeerSet(),
		blockchain,
		nil, //eth.TxPool(),
		db, //eth.ChainDb(),
		nil,
		nil,
		nil,
		quitSync,
		new(sync.WaitGroup),
		ulcConfig)
	if err != nil {
		return nil, err
	}

	lesTopics := make([]discv5.Topic, len(AdvertiseProtocolVersions))
	for i, pv := range AdvertiseProtocolVersions {
		lesTopics[i] = lesTopic(genesis.Hash(), pv)
	}

	srv := &LesServer{
		config: &eth.Config{
			LightPeers:50,
			LightServ:50,
		},
		protocolManager:  pm,
		quitSync:         quitSync,
		lesTopics:        lesTopics,
		chtIndexer:       chtIndexer,//light.NewChtIndexer(eth.ChainDb(), false),
		bloomTrieIndexer: bbtIndexer, //light.NewBloomTrieIndexer(eth.ChainDb(), false),
		onlyAnnounce:     false, //config.OnlyAnnounce,
	}

	logger := log.New()

	chtV1SectionCount, _, _ := srv.chtIndexer.Sections() // indexer still uses LES/1 4k section size for backwards server compatibility
	chtV2SectionCount := chtV1SectionCount / (light.CHTFrequencyClient / light.CHTFrequencyServer)
	if chtV2SectionCount != 0 {
		// convert to LES/2 section
		chtLastSection := chtV2SectionCount - 1
		// convert last LES/2 section index back to LES/1 index for chtIndexer.SectionHead
		chtLastSectionV1 := (chtLastSection+1)*(light.CHTFrequencyClient/light.CHTFrequencyServer) - 1
		chtSectionHead := srv.chtIndexer.SectionHead(chtLastSectionV1)
		chtRoot := light.GetChtV2Root(pm.chainDb, chtLastSection, chtSectionHead)
		logger.Info("Loaded CHT", "section", chtLastSection, "head", chtSectionHead, "root", chtRoot)
	}
	bloomTrieSectionCount, _, _ := srv.bloomTrieIndexer.Sections()
	if bloomTrieSectionCount != 0 {
		bloomTrieLastSection := bloomTrieSectionCount - 1
		bloomTrieSectionHead := srv.bloomTrieIndexer.SectionHead(bloomTrieLastSection)
		bloomTrieRoot := light.GetBloomTrieRoot(pm.chainDb, bloomTrieLastSection, bloomTrieSectionHead)
		logger.Info("Loaded bloom trie", "section", bloomTrieLastSection, "head", bloomTrieSectionHead, "root", bloomTrieRoot)
	}

	srv.chtIndexer.Start(blockchain)
	pm.server = srv

	srv.defParams = &flowcontrol.ServerParams{
		BufLimit:    300000000,
		MinRecharge: 50000,
	}
	srv.fcManager = flowcontrol.NewClientManager(uint64(srv.config.LightServ), 10, 1000000000)
	srv.fcCostStats = newCostStats(db)
	return srv, nil
}



func NewTestsLightEthereum(ctx *node.ServiceContext, config *eth.Config) (*LightEthereum, error) {
	chainDb := ethdb.NewMemDatabase()
	chainConfig, genesisHash, genesisErr := core.SetupGenesisBlock(chainDb, config.Genesis)
	if _, isCompat := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !isCompat {
		return nil, genesisErr
	}
	log.Info("Initialised chain configuration", "config", chainConfig)

	peers := newPeerSet()
	quitSync := make(chan struct{})

	leth := &LightEthereum{
		config:           config,
		chainConfig:      chainConfig,
		chainDb:          chainDb,
		eventMux:         ctx.EventMux,
		peers:            peers,
		reqDist:          newRequestDistributor(peers, quitSync),
		accountManager:   ctx.AccountManager,
		engine:           eth.CreateConsensusEngine(ctx, &config.Ethash, chainConfig, chainDb),
		shutdownChan:     make(chan bool),
		networkId:        config.NetworkId,
		bloomRequests:    make(chan chan *bloombits.Retrieval),
		bloomIndexer:     eth.NewBloomIndexer(chainDb, light.BloomTrieFrequency),
		chtIndexer:       light.NewChtIndexer(chainDb, true),
		bloomTrieIndexer: light.NewBloomTrieIndexer(chainDb, true),
	}

	var trustedNodes []string
	if leth.config.ULC != nil {
		trustedNodes = leth.config.ULC.TrustedServers
	}
	leth.relay = NewLesTxRelay(peers, leth.reqDist)
	leth.serverPool = newServerPool(chainDb, quitSync, &leth.wg, trustedNodes)
	leth.retriever = newRetrieveManager(peers, leth.reqDist, leth.serverPool)
	leth.odr = NewLesOdr(chainDb, leth.chtIndexer, leth.bloomTrieIndexer, leth.bloomIndexer, leth.retriever)
	var err error
	if leth.blockchain, err = light.NewLightChain(leth.odr, leth.chainConfig, leth.engine); err != nil {
		return nil, err
	}
	leth.bloomIndexer.Start(leth.blockchain)
	// Rewind the chain in case of an incompatible config upgrade.
	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		leth.blockchain.SetHead(compat.RewindTo)
		rawdb.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}

	leth.txPool = light.NewTxPool(leth.chainConfig, leth.blockchain, leth.relay)

	if leth.protocolManager, err = NewProtocolManager(
		leth.chainConfig,
		true,
		ClientProtocolVersions,
		config.NetworkId,
		leth.eventMux,
		leth.engine,
		leth.peers,
		leth.blockchain,
		nil,
		chainDb,
		leth.odr,
		leth.relay,
		leth.serverPool,
		quitSync,
		&leth.wg,
		config.ULC); err != nil {
		return nil, err
	}

	if leth.protocolManager.isULCEnabled() {
		leth.blockchain.DisableCheckFreq()
	}
	leth.ApiBackend = &LesApiBackend{leth, nil}

	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.GasPrice
	}
	leth.ApiBackend.gpo = gasprice.NewOracle(leth.ApiBackend, gpoParams)
	return leth, nil
}


func initNet(services map[string]adapters.ServiceFunc, nodeCount int) (*simulations.Network, error) {

	//add the streamer service to the node adapter
	adapter := adapters.NewSimAdapter(services)

	log.Info("Setting up Snapshot network")

	net := simulations.NewNetwork(adapter, &simulations.NetworkConfig{
		DefaultService: "server",
	})

	return net, nil
}