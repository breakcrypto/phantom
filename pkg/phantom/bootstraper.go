/**
*    Copyright (C) 2019-present C2CV Holdings, LLC.
*
*    This program is free software: you can redistribute it and/or modify
*    it under the terms of the Server Side Public License, version 1,
*    as published by C2CV Holdings, LLC.
*
*    This program is distributed in the hope that it will be useful,
*    but WITHOUT ANY WARRANTY; without even the implied warranty of
*    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
*    Server Side Public License for more details.
*
*    You should have received a copy of the Server Side Public License
*    along with this program. If not, see
*    <http://www.mongodb.com/licensing/server-side-public-license>.
*
*    As a special exception, the copyright holders give permission to link the
*    code of portions of this program with the OpenSSL library under certain
*    conditions as described in each individual source file and distribute
*    linked combinations including the program with the OpenSSL library. You
*    must comply with the Server Side Public License in all respects for
*    all of the code used other than as permitted herein. If you modify file(s)
*    with this exception, you may extend this exception to your version of the
*    file(s), but you are not obligated to do so. If you do not wish to do so,
*    delete this exception statement from your version. If you delete this
*    exception statement from all source files in the program, then also delete
*    it in the license file.
*/

package phantom

import (
	"encoding/json"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"io/ioutil"
	"log"
	"net/http"
	"phantom/pkg/socket/wire"
	"strconv"
)

type Bootstrapper struct {
	BaseURL string
}

type GetPeerInfoResponse struct {
	PossiblePeers []PossiblePeer
}

type PossiblePeer struct {
	Addr            string        `json:"addr"`
	Addrlocal       string        `json:"addrlocal"`
}

func (b Bootstrapper) LoadBlockHash() (chainhash.Hash, error) {
	var strBlockHash string

	response, err := http.Get(b.BaseURL + "/api/getblockcount")
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
		blockCount, _ := strconv.Atoi(string(contents))

		response, err := http.Get(b.BaseURL + "/api/getblockhash?index=" + strconv.Itoa(blockCount-12))
		if err != nil {
			log.Printf("%s", err)
			return chainhash.Hash{}, err
		} else {
			defer response.Body.Close()
			contents, err = ioutil.ReadAll(response.Body)
			if err != nil {
				log.Printf("%s", err)
				return chainhash.Hash{}, err
			}

			strBlockHash = string(contents)
		}
	}

	var bootstrapHash chainhash.Hash
	chainhash.Decode(&bootstrapHash,strBlockHash)

	return bootstrapHash, nil
}

func (b Bootstrapper) LoadPossiblePeers(portFilter uint16) ([]wire.NetAddress, error) {
	var possiblePeers []wire.NetAddress

	response, err := http.Get(b.BaseURL + "/api/getpeerinfo")
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
				possiblePeers = addPossiblePeer(possible, possiblePeers, portFilter)
			}
		}

		if possiblePeer.Addr != "" {
			possible, _ = SplitAddress(possiblePeer.Addrlocal)
			if err == nil {
				possiblePeers = addPossiblePeer(possible, possiblePeers, portFilter)
			}
		}
	}

	return possiblePeers, nil
}

func addPossiblePeer(peer wire.NetAddress, peers []wire.NetAddress, portFilter uint16) []wire.NetAddress {
	if peer.Port == portFilter {
		for _, prevPeer := range peers {
			if peer.IP.String() == prevPeer.IP.String() && peer.Port == prevPeer.Port {
				return peers
			}
		}
		return append(peers, peer)
	}
	return peers
}
