package remotechains

import (
	"encoding/json"
	"github.com/breakcrypto/phantom/pkg/socket/wire"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"io/ioutil"
	log "github.com/sirupsen/logrus"
	"net/http"
	"phantom/cmd/refactor/database"
	"strconv"
	"strings"
	"time"
)

type IquidusExplorer struct {
	BaseURL string
}

func (i *IquidusExplorer) GetBlockHash(blockNumber int) (chainhash.Hash, error) {
	var strBlockHash string

	response, err := http.Get(i.BaseURL + "/api/getblockhash?index=" + strconv.Itoa(blockNumber))
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

		strBlockHash = string(contents)
	}

	var bootstrapHash chainhash.Hash
	chainhash.Decode(&bootstrapHash,strBlockHash)

	return bootstrapHash, nil
}

func (i *IquidusExplorer) GetPeers(portFilter uint32) ([]database.Peer, error) {
	var possiblePeers []database.Peer

	response, err := http.Get(i.BaseURL + "/api/getpeerinfo")
	if err != nil {
		log.Printf("%s", err)
		return nil, err
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Printf("%s", err)
		return nil, err
	}

	var s = new([]PossiblePeer)
	err = json.Unmarshal(body, &s)
	if err != nil {
		log.Printf("%s", err)
		return nil, err
	}

	for _, possiblePeer := range *s {
		var possible wire.NetAddress

		if possiblePeer.Addr != "" {
			possible, _ = SplitAddress(possiblePeer.Addr)
			if err == nil {
				peer := database.Peer{Address:possible.IP.String(),Port:uint32(possible.Port),LastSeen:time.Now()}
				possiblePeers = AddPossiblePeer(peer, possiblePeers, portFilter)
			}
		}

		if possiblePeer.Addr != "" {
			possible, _ = SplitAddress(possiblePeer.Addrlocal)
			if err == nil {
				peer := database.Peer{Address:possible.IP.String(),Port:uint32(possible.Port),LastSeen:time.Now()}
				possiblePeers = AddPossiblePeer(peer, possiblePeers, portFilter)
			}
		}
	}

	return possiblePeers, nil
}

func (i *IquidusExplorer) GetChainHeight() (blockCount int, err error) {
	response, err := http.Get(i.BaseURL + "/api/getblockcount")
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

func (i *IquidusExplorer) GetTransaction(txid string) (string, error) {
	response, err := http.Get(i.BaseURL + "/api/getrawtransaction?txid=" + txid + "&decrypt=1")
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

func (i *IquidusExplorer) SetURL(url string) {
	i.BaseURL = strings.TrimRight(url,"/")
}

func (i *IquidusExplorer) SetUsername(username string) {}
func (i *IquidusExplorer) SetPassword(password string) {}
