package remotechains

import (
	"encoding/json"
	"errors"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"phantom/cmd/refactor/database"
	"strconv"
	"strings"
)

type BlockbookBlockResult struct {
	Hash          string    `json:"blockHash"`
}

type BlockbookInfoResult struct {
	Blockbook struct {
		BestHeight      int       `json:"bestHeight"`
	} `json:"blockbook"`
}

type BlockbookExplorer struct {
	BaseURL string
}

func (e *BlockbookExplorer) GetBlockHash(blockNumber int) (chainhash.Hash, error) {
	var result BlockbookBlockResult

	response, err := http.Get(e.BaseURL + "/block-index/" + strconv.Itoa(blockNumber))
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
	chainhash.Decode(&bootstrapHash,result.Hash)

	return bootstrapHash, nil
}

func (e *BlockbookExplorer) GetPeers(portFilter uint32) ([]database.Peer, error) {
	return nil, errors.New("Blockbook doesn't support peers.")
}

func (e *BlockbookExplorer) GetChainHeight() (blockCount int, err error) {
	var result BlockbookInfoResult

	response, err := http.Get(e.BaseURL + "/")
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

	return result.Blockbook.BestHeight, nil
}

func (e *BlockbookExplorer) GetTransaction(txid string) (string, error) {
	return "", errors.New("Blockbook doesn't support transactions yet.")
}

func (e *BlockbookExplorer) SetURL(url string) {
	e.BaseURL = strings.TrimRight(url,"/")
}

func (e *BlockbookExplorer) SetUsername(username string) {}
func (e *BlockbookExplorer) SetPassword(password string) {}
