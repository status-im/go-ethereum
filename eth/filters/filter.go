// Copyright 2014 The go-ethereum Authors
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

package filters

import (
	//	"math"
	"time"

	"github.com/ethereum/go-ethereum/common"
	//	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/rpc"
	"golang.org/x/net/context"
)

// Filter can be used to retrieve and filter logs
type Filter struct {
	apiBackend ethapi.Backend

	created time.Time

	db         ethdb.Database
	begin, end int64
	addresses  []common.Address
	topics     [][]common.Hash
}

// New creates a new filter which uses a bloom filter on blocks to figure out whether
// a particular block is interesting or not.
func New(apiBackend ethapi.Backend) *Filter {
	return &Filter{
		apiBackend: apiBackend,
		db:         apiBackend.ChainDb(),
	}
}

// SetBeginBlock sets the earliest block for filtering.
// -1 = latest block (i.e., the current block)
// hash = particular hash from-to
func (f *Filter) SetBeginBlock(begin int64) {
	f.begin = begin
}

// SetEndBlock sets the latest block for filtering.
func (f *Filter) SetEndBlock(end int64) {
	f.end = end
}

// SetAddresses matches only logs that are generated from addresses that are included
// in the given addresses.
func (f *Filter) SetAddresses(addr []common.Address) {
	f.addresses = addr
}

// SetTopics matches only logs that have topics matching the given topics.
func (f *Filter) SetTopics(topics [][]common.Hash) {
	f.topics = topics
}

// Run filters logs with the current parameters set
func (f *Filter) Find(ctx context.Context) ([]Log, error) {
	head, _ := f.apiBackend.HeaderByNumber(ctx, rpc.LatestBlockNumber)
	headBlockNumber := head.Number.Uint64()

	var beginBlockNo uint64 = uint64(f.begin)
	if f.begin == -1 {
		beginBlockNo = headBlockNumber
	}
	var endBlockNo uint64 = uint64(f.end)
	if f.end == -1 {
		endBlockNo = headBlockNumber
	}

	// if no addresses are present we can't make use of fast search which
	// uses the mipmap bloom filters to check for fast inclusion and uses
	// higher range probability in order to ensure at least a false positive
	//	if len(self.addresses) == 0 {
	return f.getLogs(ctx, beginBlockNo, endBlockNo)
	//	}
	//	return self.mipFind(beginBlockNo, endBlockNo, 0)
}

/*func (self *Filter) mipFind(start, end uint64, depth int) (logs vm.Logs) {
>>>>>>> les: light client protocol and API
	level := core.MIPMapLevels[depth]
	// normalise numerator so we can work in level specific batches and
	// work with the proper range checks
	for num := start / level * level; num <= end; num += level {
		// find addresses in bloom filters
		bloom := core.GetMipmapBloom(f.db, num, level)
		for _, addr := range f.addresses {
			if bloom.TestBytes(addr[:]) {
				// range check normalised values and make sure that
				// we're resolving the correct range instead of the
				// normalised values.
				start := uint64(math.Max(float64(num), float64(start)))
				end := uint64(math.Min(float64(num+level-1), float64(end)))
				if depth+1 == len(core.MIPMapLevels) {
					logs = append(logs, f.getLogs(start, end)...)
				} else {
					logs = append(logs, f.mipFind(start, end, depth+1)...)
				}
				// break so we don't check the same range for each
				// possible address. Checks on multiple addresses
				// are handled further down the stack.
				break
			}
		}
	}

	return logs
}*/

func (f *Filter) getLogs(ctx context.Context, start, end uint64) (logs []Log, err error) {
	for i := start; i <= end; i++ {
		header, err := f.apiBackend.HeaderByNumber(ctx, rpc.BlockNumber(i))
		if header == nil || err != nil {
			return logs, err
		}

		// Use bloom filtering to see if this block is interesting given the
		// current parameters
		if f.bloomFilter(header.Bloom) {
			// Get the logs of the block
			receipts, err := f.apiBackend.GetReceipts(ctx, header.Hash())
			if err != nil {
				return nil, err
			}
			var unfiltered []Log
			for _, receipt := range receipts {
				rl := make([]Log, len(receipt.Logs))
				for i, l := range receipt.Logs {
					rl[i] = Log{l, false}
				}
				unfiltered = append(unfiltered, rl...)
			}
			logs = append(logs, filterLogs(unfiltered, f.addresses, f.topics)...)
		}
	}

	return logs, nil
}

func includes(addresses []common.Address, a common.Address) bool {
	for _, addr := range addresses {
		if addr == a {
			return true
		}
	}

	return false
}

func filterLogs(logs []Log, addresses []common.Address, topics [][]common.Hash) []Log {
	var ret []Log

	// Filter the logs for interesting stuff
Logs:
	for _, log := range logs {
		if len(addresses) > 0 && !includes(addresses, log.Address) {
			continue
		}

		logTopics := make([]common.Hash, len(topics))
		copy(logTopics, log.Topics)

		// If the to filtered topics is greater than the amount of topics in logs, skip.
		if len(topics) > len(log.Topics) {
			continue Logs
		}

		for i, topics := range topics {
			var match bool
			for _, topic := range topics {
				// common.Hash{} is a match all (wildcard)
				if (topic == common.Hash{}) || log.Topics[i] == topic {
					match = true
					break
				}
			}

			if !match {
				continue Logs
			}
		}

		ret = append(ret, log)
	}

	return ret
}

func (f *Filter) bloomFilter(bloom types.Bloom) bool {
	if len(f.addresses) > 0 {
		var included bool
		for _, addr := range f.addresses {
			if types.BloomLookup(bloom, addr) {
				included = true
				break
			}
		}

		if !included {
			return false
		}
	}

	for _, sub := range f.topics {
		var included bool
		for _, topic := range sub {
			if (topic == common.Hash{}) || types.BloomLookup(bloom, topic) {
				included = true
				break
			}
		}
		if !included {
			return false
		}
	}

	return true
}
