package remotechains

import (
	"encoding/json"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/pkg/errors"
	"io/ioutil"
	log "github.com/sirupsen/logrus"
	"net/http"
	"phantom/cmd/refactor/database"
	"strconv"
)

//https://gist.github.com/jackzampolin/da3201b89d23dd5fa3becb0185da1fb2

type InsightExplorer struct {
	BaseURL string
}

type SyncResult struct {
	BlockChainHeight int `json:"blockChainHeight"`
}

type BlockIndexResult struct {
	BlockHash string `json:"blockHash"`
}

func (i *InsightExplorer) GetBlockHash(blockNumber int) (chainhash.Hash, error) {
	var result BlockIndexResult

	response, err := http.Get(i.BaseURL + "/api/block-index/" + strconv.Itoa(blockNumber))
	if err != nil {
		log.Printf("%s", err)
		return chainhash.Hash{}, err
	} else {
		defer response.Body.Close()
		contents, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Error("%s", err)
			return chainhash.Hash{}, err
		}

		err = json.Unmarshal([]byte(contents), &result)
		if err != nil {
			log.Error(err)
			return chainhash.Hash{}, err
		}
	}

	var bootstrapHash chainhash.Hash
	chainhash.Decode(&bootstrapHash,result.BlockHash)

	return bootstrapHash, nil
}

func (i *InsightExplorer) GetPeers(portFilter uint32) ([]database.Peer, error) {
	return nil, errors.New("Insight doesn't support peers.")
}

func (i *InsightExplorer) GetChainHeight() (blockCount int, err error) {
	var result SyncResult

	response, err := http.Get(i.BaseURL + "/api/sync")

	if err != nil {
		log.Printf("%s", err)
		return -1, err
	} else {
		defer response.Body.Close()
		contents, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Error("%s", err)
			return -1, err
		}

		err = json.Unmarshal([]byte(contents), &result)
		if err != nil {
			log.Error(err)
			return -1, err
		}
	}

	return result.BlockChainHeight, nil
}

func (i *InsightExplorer) GetTransaction(txid string) (string, error) {
	response, err := http.Get(i.BaseURL + "/api/tx/" + txid)
	if err != nil {
		log.Printf("%s", err)
		return "", err
	} else {
		defer response.Body.Close()
		contents, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Printf("%s", err)
			return "", err
		}
		return string(contents), nil
	}
	return "", nil
}

func (i *InsightExplorer) SetURL(url string) {
	if url[len(url)-1] == '/' {
		url = url[0:len(url)-1]
	}
	i.BaseURL = url
}

func (i *InsightExplorer) SetUsername(username string) {}
func (i *InsightExplorer) SetPassword(password string) {}