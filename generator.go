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

package main

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcutil"
	"log"
	"os"
	"phantom/socket/wire"
	"sort"
	"strconv"
	"strings"
	"time"
)

type MasternodePing struct {
	Name string
	OutpointHash string
	OutpointIndex uint32
	PrivateKey string
	PingTime time.Time
	MagicMessage string
	SentinelVersion uint32
	DaemonVersion uint32
	HashQueue *Queue
}

type pingSlice []MasternodePing

func (p pingSlice) Len() int {
	return len(p)
}

func (p pingSlice) Less(i, j int) bool {
	return p[i].PingTime.Before(p[j].PingTime)
}

func (p pingSlice) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func determinePingTime(unixTime string) (time.Time) {

	i, err := strconv.ParseInt(unixTime, 10, 64)
	if err != nil {
		panic(err)
	}
	base := time.Unix(i, 0)

	difference := time.Now().UTC().Sub(base)

	//var bump uint32
	bump := uint32(difference.Minutes() / 10) + 1
	result := time.Minute * time.Duration(bump) * 10

	return base.Add(result)
}

func GeneratePingsFromMasternodeFile(filePath string, pingChannel chan MasternodePing, queue *Queue, magicMessage string, sentinelVersion uint32) {

	file, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	pings := make(pingSlice, 0)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())

		outputIndex, err := strconv.Atoi(fields[4])
		if err != nil {
			log.Println("Error reading masternode index.")
		}

		pings = append(pings, MasternodePing{fields[0], fields[3], uint32(outputIndex), fields[2], determinePingTime(fields[5]), magicMessage, sentinelVersion, daemonVersion, queue})

	}

	//sort the pings by time
	sort.Sort(pings)

	//we have a sorted list of pings -- add them to the channel
	for _, ping := range pings {
		//fmt.Println("Enabling: ", ping.Name)
		log.Printf("%s : Enabling.\n", ping.Name)
		pingChannel <- ping
	}

}

func (ping *MasternodePing) GenerateMasternodePing() (wire.MsgMNP){
	mnp := wire.MsgMNP{}

	//add sentinel support
	if sentinelVersion > 0 {
		mnp.SentinelEnabled = true
		mnp.SentinelVersion = sentinelVersion
	}

	//add daemon support
	if daemonVersion > 0 {
		mnp.DaemonEnabled = true
		mnp.DaemonVersion = daemonVersion
	}

	mnp.BlockHash = *ping.HashQueue.Peek()

	//setup the outpoint
	var outpointHash chainhash.Hash
	chainhash.Decode(&outpointHash,ping.OutpointHash)
	outpoint := wire.NewOutPoint(&outpointHash, ping.OutpointIndex)
	txIn := wire.NewTxIn(outpoint, nil, nil)

	mnp.Vin = *txIn

	//setup the time
	mnp.SigTime = uint64(ping.PingTime.Add(time.Second * 3).UTC().Unix()) //generate a deterministic time

	//sign the ping
	wif, err := btcutil.DecodeWIF(ping.PrivateKey)
	if err != nil {
		log.Println(err)
	}

	signatureHash := GenerateMNPSignature(ping.MagicMessage, mnp.Vin.PreviousOutPoint.Hash.String(), mnp.Vin.PreviousOutPoint.Index, mnp.Vin.SignatureScript, mnp.BlockHash.String(), mnp.SigTime, *wif.PrivKey)
	if err != nil {
		log.Println(err)
	}

	//push the bytes to the mnp
	mnp.VchSig = signatureHash

	return mnp
}

func GenerateMNPSignature(magicMessage string, hash string, n uint32, scriptSig []byte, blockHash string, sigTime uint64, privKey btcec.PrivateKey) []byte {
	var buf bytes.Buffer
	wire.WriteVarString(&buf, 0, magicMessage) //"DarkCoin Signed Message:\n" - $PAC || "ProtonCoin Signed Message:\n" - ANDS
	wire.WriteVarString(&buf, 0, fmt.Sprintf("CTxIn(COutPoint(%s, %d), scriptSig=%s)%s%s", hash, n, hex.EncodeToString(scriptSig), blockHash, strconv.FormatInt(int64(sigTime), 10)))
	expectedMessageHash := chainhash.DoubleHashB(buf.Bytes())

	sig, _ := btcec.SignCompact(btcec.S256(), &privKey, expectedMessageHash, false)

	return sig
}
