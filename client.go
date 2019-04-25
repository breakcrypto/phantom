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
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"log"
	"net"
	"phantom/socket/wire"
	"strconv"
	"strings"
	"sync"
	"time"
)

type PingerConnection struct {
	MagicBytes uint32
	IpAddress string
	Port uint16
	ProtocolNumber uint32
	BootstrapHash chainhash.Hash
	PingChannel chan MasternodePing
	AddrChannel chan wire.NetAddress
	HashChannel chan chainhash.Hash
	Status int8
	WaitGroup *sync.WaitGroup
	Mutex sync.Mutex
}

func (pinger *PingerConnection) Start() {

	log.Printf("%s : STARTING CLIENT\n", pinger.IpAddress)

	//make sure we close out the waitGroup
	defer pinger.WaitGroup.Done()

	var connectionAttempts uint8 = 0

	var pingMap map[string]wire.MsgMNP

	var magic = wire.BitcoinNet(pinger.MagicBytes)

	me := wire.NetAddress{
		Timestamp: time.Time{},
		Services:  0,
		IP:        net.ParseIP("8.8.8.8"),
		Port:      pinger.Port,
	}

	you := wire.NetAddress{
		Timestamp: time.Time{},
		Services:  0,
		IP:        net.ParseIP(pinger.IpAddress),
		Port:      pinger.Port,
	}

	version := wire.MsgVersion{
		ProtocolVersion: int32(pinger.ProtocolNumber),
		Services:        0,
		Timestamp:       time.Unix(time.Now().Unix(), 0),
		AddrYou:         you,
		AddrMe:          me,
		Nonce:           0xDEADBEEF,
		UserAgent:       "@_breakcrypto phantom",
		LastBlock:       0,
		DisableRelayTx:  true,
	}

	tcpAddr, err := net.ResolveTCPAddr("tcp4", pinger.IpAddress + ":" + strconv.Itoa(int(pinger.Port)))
	if(err != nil) {
		log.Println(err)
		pinger.SetStatus(-1)
		return
	}

	//setup the ping inv map
	pingMap = make(map[string]wire.MsgMNP)

	for {

		if connectionAttempts >= 10 || len(pinger.PingChannel) > 10 {
			log.Println("Unable to connect -- closing connection / channel too full.")
			pinger.SetStatus(-1)
			return
		}

		conn, err := net.DialTCP("tcp", nil, tcpAddr)
		if (err != nil) {
			log.Println(err)
			connectionAttempts++
			continue
		}

		var buf bytes.Buffer
		wire.WriteMessageN(&buf, &version, pinger.ProtocolNumber, magic)
		conn.Write(buf.Bytes())

		bufReader := bufio.NewReader(conn)

		for {

			if pinger.GetStatus() < 0 {
				return
			}

			//connection failed, set the status to -1 and let it be reaped
			if connectionAttempts >= 10 || len(pinger.PingChannel) > 10 {
				log.Println("Unable to connect -- closing connection / channel too full (inside).")
				pinger.SetStatus(-1)
				return
			}

			_, msg, _, err := wire.ReadMessageN(bufReader, pinger.ProtocolNumber, magic)

			if (err != nil) {
				if strings.Contains(err.Error(), "unhandled command") {
					continue
				}
				log.Printf("%s : %s\n", pinger.IpAddress, err)
				connectionAttempts++
				continue
			} else {

				connectionAttempts = 0

				if (msg.Command() == "inv") {
					inv := msg.(*wire.MsgInv)
					for _, inventory := range (inv.InvList) {
						if inventory.Type.String() == "MSG_BLOCK" {
							log.Println("New block received: \n" + inventory.Hash.String())
							pinger.HashChannel <- inventory.Hash
						}
					}
				}

				if (msg.Command() == "version") {
					verack := wire.MsgVerAck{}

					var buf bytes.Buffer
					wire.WriteMessageN(&buf, &verack, pinger.ProtocolNumber, magic)
					conn.Write(buf.Bytes())

					pinger.SetStatus(1) //we're connected and ready to start pinging

					//ignore the request but relay our own 'getaddr' request
					getaddr := wire.MsgGetAddr{}

					var bufAddr bytes.Buffer
					wire.WriteMessageN(&bufAddr, &getaddr, pinger.ProtocolNumber, magic)
					conn.Write(bufAddr.Bytes())

					log.Println("Sending getaddr")

					defaultHash := chainhash.Hash{}
					if pinger.BootstrapHash != defaultHash {
						getblocks := wire.MsgGetBlocks{}

						getblocks.BlockLocatorHashes = []*chainhash.Hash{&pinger.BootstrapHash}
						getblocks.ProtocolVersion = pinger.ProtocolNumber

						var bufBlocks bytes.Buffer
						wire.WriteMessageN(&bufBlocks, &getblocks, pinger.ProtocolNumber, magic)
						conn.Write(bufBlocks.Bytes())

						log.Println("Sending getblocks to bootstrap")
					}
				}

				if (msg.Command() == "ping") {

					ping := msg.(*wire.MsgPing)

					pong := wire.MsgPong{Nonce: ping.Nonce}

					var buf bytes.Buffer
					wire.WriteMessageN(&buf, &pong, pinger.ProtocolNumber, magic)
					conn.Write(buf.Bytes())

					log.Printf("%s : PONG!\n", pinger.IpAddress)

					//clear out the ping map
					for hash, ping := range pingMap {
						pingTime := time.Unix(int64(ping.SigTime), 0)
						//if the ping is more than a minute old, delete it
						if pingTime.Add(time.Minute * 1).Before(time.Now().UTC()) {
							delete(pingMap, hash)
						}
					}
				}

				if (msg.Command() == "addr") {
					msgAddr := msg.(*wire.MsgAddr)
					for _, addr := range msgAddr.AddrList {
						//log.Println("PEER: ", addr.IP, ":", addr.Port)
						pinger.AddrChannel <- *addr
					}
				}

				//non-blocking select
				select {
					case ping := <-pinger.PingChannel:
						log.Printf("REQUEST RECIEVED, RELAYING: %s\n",
							ping.Name)

						mnp := ping.GenerateMasternodePing()

						//serialize to a []byte
						w := new(bytes.Buffer)
						mnp.Serialize(w)
						mnpBytes := w.Bytes()

						inv := wire.MsgInv{}
						invVec := wire.InvVect{}
						invVec.Type = 15
						invVec.Hash = chainhash.DoubleHashH(mnpBytes)
						inv.AddInvVect(&invVec)

						//send the ping inv
						var buf bytes.Buffer
						wire.WriteMessageN(&buf, &inv, pinger.ProtocolNumber, magic)
						conn.Write(buf.Bytes())

						//store the ping
						pingMap[invVec.Hash.String()] = mnp

					default:
						//fmt.Println("no message received")
				}

				//this should really be a hashMap with expiring entries
				if (msg.Command() == "getdata") {

					getData := msg.(*wire.MsgGetData)

					for _, inv := range getData.InvList {
						//check the map
						str := inv.Hash.String()
						if val, ok := pingMap[str]; ok {
							//do something here
							var buf bytes.Buffer
							wire.WriteMessageN(&buf, &val, pinger.ProtocolNumber, magic)
							conn.Write(buf.Bytes())
						}
					}
				}
			}
		}

		//we've disconnected, so try again
		connectionAttempts++
		log.Printf("%s : There's been an error, attempting to reconnect.\n", pinger.IpAddress)
		time.Sleep(1 * time.Minute)
	}
}

func (pinger *PingerConnection) SetStatus(status int8) {
	pinger.Mutex.Lock()
	defer pinger.Mutex.Unlock()

	pinger.Status = status
}

func (pinger *PingerConnection) GetStatus() (int8) {
	pinger.Mutex.Lock()
	defer pinger.Mutex.Unlock()

	return pinger.Status
}
