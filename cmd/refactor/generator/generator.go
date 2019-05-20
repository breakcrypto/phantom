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

package generator

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcutil"
	log "github.com/sirupsen/logrus"
	"os"
	"phantom/cmd/refactor/blockqueue"
	"phantom/cmd/refactor/broadcaststore"
	"phantom/cmd/refactor/events"
	"phantom/pkg/socket/wire"
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
	UseOutpoint bool
	SentinelVersion uint32
	DaemonVersion uint32
	BroadcastTemplate *wire.MsgMNB
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

func GeneratePingsFromMasternodeFile(filePath string,
	magicMessage string,
	useOutpoint bool,
	sentinelVersion uint32,
	daemonVersion uint32,
	channels ...chan events.Event) {

	currentTime := time.Now().UTC()

	file, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	pings := make(pingSlice, 0)

	scanner := bufio.NewScanner(file)

	i := 0
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) < 1 || line[0] == '#' {
			continue
		}

		fields := strings.Fields(line)

		//add an epoch if missing and alert
		if len(fields) == 5 {
			log.Warn("No epoch time found for: ", fields[0], " assuming one.")
			fields = append(fields, strconv.FormatInt(currentTime.Add(time.Duration(i*7) * time.Second).Unix() - 540, 10))
			i++
		}

		if len(fields) != 6 {
			log.Println("Error processing: ", line)
			continue
		}

		outputIndex, err := strconv.Atoi(fields[4])
		if err != nil {
			log.Println("Error reading masternode index value.")
		}

		ping := MasternodePing{fields[0],
			fields[3],
			uint32(outputIndex),
			fields[2],
			determinePingTime(fields[5]),
			magicMessage,
			useOutpoint,
			sentinelVersion,
			daemonVersion,
			nil,
		}

		//check for a broadcast template
		broadcast := broadcaststore.GetInstance().GetBroadcast(ping.OutpointHash +
		":" + strconv.Itoa(int(ping.OutpointIndex)))

		//provide the template
		if broadcast != nil {
			log.Debug("Broadcast template located for: ", broadcast.Vin.PreviousOutPoint.String())
			ping.BroadcastTemplate = broadcast
		}

		pings = append(pings, ping)
	}

	//sort the pings by time
	sort.Sort(pings)

	//we have a sorted list of pings -- add them to the channel
	for _, ping := range pings {
		log.Info("Generating a ping for: ", ping.Name)

		sleepTime := ping.PingTime.Sub(time.Now())

		log.Debug(time.Now().UTC())
		log.Info(ping.Name, " - ", ping.PingTime.UTC())

		if sleepTime > 0 {
			log.Info("Sleeping for ", sleepTime.String())
			time.Sleep(sleepTime)
		}

		log.Debug(ping.Name, " ping being transmitted to the network.")

		for _, channel := range channels {
			pingCopy := ping

			select {
			case channel <- events.Event{events.NewPhantomPing, &pingCopy}:
				log.Debug("Relayed ping: ", ping.Name)
			default:
				//fmt.Println("no message sent.")
			}
		}
	}
}

func (ping *MasternodePing) GenerateMasternodePing(useOutpoint bool, sentinelVersion uint32, daemonVersion uint32) (wire.MsgMNP){

	mnp := wire.MsgMNP{UseOutpointForm:useOutpoint}

	//add sentinel support
	if sentinelVersion > 0 {
		mnp.SentinelVersion = sentinelVersion
	}

	//add daemon support
	if daemonVersion > 0 {
		mnp.DaemonVersion = daemonVersion
	}

	mnp.BlockHash = *blockqueue.GetInstance().GetTop()

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
