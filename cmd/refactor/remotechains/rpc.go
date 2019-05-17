package remotechains

import (
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/pkg/errors"
	"phantom/cmd/refactor/database"
)

type RPCExplorer struct {
	BaseURL string
	Username string
	Password string
}

func (i *RPCExplorer) GetBlockHash(blockNumber int) (chainhash.Hash, error) {
	return chainhash.Hash{}, errors.New("Not yet implented.")
}

func (i *RPCExplorer) GetPeers(portFilter uint32) ([]database.Peer, error) {
	return nil, errors.New("Not yet implented.")
}

func (i *RPCExplorer) GetChainHeight() (blockCount int, err error) {
	return 0, errors.New("Not yet implented.")
}

func (i *RPCExplorer) GetTransaction(txid string) (string, error) {
	return "", errors.New("Not yet implented.")
}

func (i *RPCExplorer) SetURL(url string) {
	i.BaseURL = url
}

func (i *RPCExplorer) SetUsername(username string) {
	i.Username = username
}

func (i *RPCExplorer) SetPassword(password string) {
	i.Password = password
}
