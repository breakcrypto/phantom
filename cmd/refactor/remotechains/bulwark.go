package remotechains

import (
	"encoding/json"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"phantom/cmd/refactor/database"
	"strconv"
	"strings"
	"time"
)

type BulwarkBlockResult struct {
	Block struct {
		Hash          string    `json:"hash"`
	} `json:"block"`
}


type BulwarkPeer struct {
	IP          string    `json:"ip"`
	Port        int       `json:"port"`
}

type BulwarkPeers []BulwarkPeer

type BulwarkExplorer struct {
	BaseURL string
}

func (e *BulwarkExplorer) GetBlockHash(blockNumber int) (chainhash.Hash, error) {
	var result BulwarkBlockResult

	response, err := http.Get(e.BaseURL + "/block/" + strconv.Itoa(blockNumber))
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
	chainhash.Decode(&bootstrapHash,result.Block.Hash)

	return bootstrapHash, nil
}

func (e *BulwarkExplorer) GetPeers(portFilter uint32) ([]database.Peer, error) {
	var possiblePeers []database.Peer

	response, err := http.Get(e.BaseURL + "/peer")
	if err != nil {
		log.Printf("%s", err)
		return nil, err
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Printf("%s", err)
		return nil, err
	}

	var bulwarkPeers BulwarkPeers
	err = json.Unmarshal(body, &bulwarkPeers)
	if err != nil {
		log.Printf("%s", err)
		return nil, err
	}

	for _, possiblePeer := range bulwarkPeers {
		peer := database.Peer{Address:possiblePeer.IP,Port:uint32(possiblePeer.Port),LastSeen:time.Now()}
		possiblePeers = AddPossiblePeer(peer, possiblePeers, portFilter)
	}

	return possiblePeers, nil
}

func (e *BulwarkExplorer) GetChainHeight() (blockCount int, err error) {
	response, err := http.Get(e.BaseURL + "/getblockcount")
	if err != nil {
		log.Printf("%s", err)
		return -1, err
	} else {
		defer response.Body.Close()
		contents, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Printf("%s", err)
			return -1, err
		}
		blockCount, _ = strconv.Atoi(string(contents))
	}
	return blockCount, nil
}

func (e *BulwarkExplorer) GetTransaction(txid string) (string, error) {
	response, err := http.Get(e.BaseURL + "/tx/" + txid)
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

func (e *BulwarkExplorer) SetURL(url string) {
	e.BaseURL = strings.TrimRight(url,"/")
}

func (e *BulwarkExplorer) SetUsername(username string) {}
func (e *BulwarkExplorer) SetPassword(password string) {}
