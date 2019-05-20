package remotechains

import (
	"errors"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"phantom/cmd/refactor/database"
	"strconv"
	"strings"
)

type CryptoidExplorer struct {
	BaseURL string
}

func (e *CryptoidExplorer) GetBlockHash(blockNumber int) (bootstrapHash chainhash.Hash, err error) {
	response, err := http.Get(e.BaseURL + "?q=getblockhash&height=" + strconv.Itoa(blockNumber))
	if err != nil {
		log.Printf("%s", err)
		return chainhash.Hash{}, err
	} else {
		defer response.Body.Close()
		contents, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Printf("%s", err)
			return chainhash.Hash{}, err
		}
		//strip off the "s
		chainhash.Decode(&bootstrapHash,strings.Trim(string(contents),`"`))
	}
	return bootstrapHash, nil
}

func (e *CryptoidExplorer) GetPeers(portFilter uint32) ([]database.Peer, error) {
	return nil, errors.New("Cryptoid doesn't support peers.")
}

func (e *CryptoidExplorer) GetChainHeight() (blockCount int, err error) {
	response, err := http.Get(e.BaseURL + "?q=getblockcount")
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

func (e *CryptoidExplorer) GetTransaction(txid string) (string, error) {
	response, err := http.Get(e.BaseURL + "?q=txinfo&t=" + txid)
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

func (e *CryptoidExplorer) SetURL(url string) {
	url = strings.TrimRight(url,"/")
}

func (e *CryptoidExplorer) SetUsername(username string) {}
func (e *CryptoidExplorer) SetPassword(password string) {}
