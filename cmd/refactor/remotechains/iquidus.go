package remotechains

import (
	"encoding/json"
	"github.com/breakcrypto/phantom/pkg/socket/wire"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"phantom/cmd/refactor/database"
	"strconv"
	"time"
)

type GetPeerInfoResponse struct {
	PossiblePeers []PossiblePeer
}

type PossiblePeer struct {
	Addr            string        `json:"addr"`
	Addrlocal       string        `json:"addrlocal"`
}

type IquidusExplorer struct {
	BaseURL string
}

func (i IquidusExplorer) GetBlockHash(blockNumber uint64) (chainhash.Hash, error) {
	var strBlockHash string

	blockCount, _ := i.GetChainHeight()

	response, err := http.Get(i.BaseURL + "/api/getblockhash?index=" + strconv.Itoa(blockCount-12))
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

func (i IquidusExplorer) GetPeers(portFilter uint32) ([]database.Peer, error) {
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
				possiblePeers = addPossiblePeer(peer, possiblePeers, portFilter)
			}
		}

		if possiblePeer.Addr != "" {
			possible, _ = SplitAddress(possiblePeer.Addrlocal)
			if err == nil {
				peer := database.Peer{Address:possible.IP.String(),Port:uint32(possible.Port),LastSeen:time.Now()}
				possiblePeers = addPossiblePeer(peer, possiblePeers, portFilter)
			}
		}
	}

	return possiblePeers, nil
}

func addPossiblePeer(peer database.Peer, peers []database.Peer, portFilter uint32) []database.Peer {
	if peer.Port == portFilter {
		for _, prevPeer := range peers {
			if peer.Address == prevPeer.Address && peer.Port == prevPeer.Port {
				return peers
			}
		}
		return append(peers, peer)
	}
	return peers
}

func SplitAddress(pair string) (wire.NetAddress, error) {
	host, port, err := net.SplitHostPort(pair)
	if err != nil {
		log.Println(err)
	}

	parsedPort, err := strconv.Atoi(port)

	return wire.NetAddress{time.Now(),
		0,
		net.ParseIP(host),
		uint16(parsedPort)}, nil
}

func (i IquidusExplorer) GetChainHeight() (blockCount int, err error) {
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

func (i IquidusExplorer) GetTransaction(txid string) (string, error) {
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
