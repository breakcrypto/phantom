package remotechains

import (
	"encoding/json"
	"errors"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"io/ioutil"
	log "github.com/sirupsen/logrus"
	"net/http"
	"phantom/cmd/refactor/database"
	"strconv"
	"strings"
)

type CoinExplorerBlockResult struct {
	Result  struct {
	Hash              string    `json:"hash"`
	} `json:"result"`
}

type CoinExplorerBlockLatestResult struct {
	Result  struct {
		Height            int   	`json:"height"`
	} `json:"result"`
}

type CoinExplorer struct {
	BaseURL string
}

func (e *CoinExplorer) GetBlockHash(blockNumber int) (chainhash.Hash, error) {
	var result CoinExplorerBlockResult

	response, err := http.Get(e.BaseURL + "/block?height=" + strconv.Itoa(blockNumber))
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
	chainhash.Decode(&bootstrapHash,result.Result.Hash)

	return bootstrapHash, nil
}

func (e *CoinExplorer) GetPeers(portFilter uint32) ([]database.Peer, error) {
	return nil, errors.New("CoinExplorer doesn't support peers.")
}

func (e *CoinExplorer) GetChainHeight() (blockNumber int, err error) {
	var result CoinExplorerBlockLatestResult

	response, err := http.Get(e.BaseURL + "/block/latest")
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
	return result.Result.Height, nil
}

func (e *CoinExplorer) GetTransaction(txid string) (string, error) {
	response, err := http.Get(e.BaseURL + "/transaction?txid=" + txid)
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

func (e *CoinExplorer) SetURL(url string) {
	e.BaseURL = strings.TrimRight(url,"/")
}

func (e *CoinExplorer) SetUsername(username string) {}
func (e *CoinExplorer) SetPassword(password string) {}
