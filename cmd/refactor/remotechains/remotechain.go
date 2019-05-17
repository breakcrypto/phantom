package remotechains

import (
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"phantom/cmd/refactor/database"
)

type RemoteChain interface {
	GetBlockHash(blockNumber uint64) (chainhash.Hash, error)
	GetPeers(portFilter uint32) ([]database.Peer, error)
	GetChainHeight() (int, error)
	GetTransaction(id string) (string, error)
}
