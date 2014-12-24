package miner

import (
	"sync"

	"github.com/NebulousLabs/Sia/consensus"
)

type Miner struct {
	// Block variables - helps the miner construct the next block.
	parent            consensus.BlockID
	transactions      []consensus.Transaction
	address           consensus.CoinAddress
	target            consensus.Target
	earliestTimestamp consensus.Timestamp

	threads              int // how many threads the miner usually uses.
	desiredThreads       int // 0 if not mining.
	runningThreads       int
	iterationsPerAttempt uint64

	blockChan chan consensus.Block
	sync.RWMutex
}

// New takes a block channel down which it drops blocks that it mines. It also
// takes a thread count, which it uses to spin up miners on separate threads.
func New(blockChan chan consensus.Block, threads int) (m *Miner) {
	return &Miner{
		threads:              threads,
		iterationsPerAttempt: 256 * 1024,
		blockChan:            blockChan,
	}
}

// Info() returns a JSON struct which can be parsed by frontends for displaying
// information to the user.
func (m *Miner) Info() ([]byte, error) {
	return nil, nil
}

// SubsidyAddress returns the address that is currently being used by the miner
// while searching for blocks.
func (m *Miner) SubsidyAddress() consensus.CoinAddress {
	m.Lock()
	defer m.Unlock()

	return m.address
}

// Update changes what block the miner is mining on. Changes include address
// and target.
func (m *Miner) Update(parent consensus.BlockID, transactions []consensus.Transaction, target consensus.Target, address consensus.CoinAddress, earliestTimestamp consensus.Timestamp) error {
	m.Lock()
	defer m.Unlock()

	m.parent = parent
	m.transactions = transactions
	m.target = target
	m.address = address
	m.earliestTimestamp = earliestTimestamp
	return nil
}
