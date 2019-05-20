package main

import (
	"bufio"
	"bytes"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	log "github.com/sirupsen/logrus"
	"net"
	"phantom/cmd/refactor/database"
	"phantom/cmd/refactor/events"
	"phantom/cmd/refactor/generator"
	"phantom/pkg/socket/wire"
	"strconv"
	"strings"
	"sync"
	"time"
)

type PeerConnection struct {
	MagicBytes uint32
	PeerInfo database.Peer
	ProtocolNumber uint32
	MagicMessage string
	SentinelVersion uint32
	DaemonVersion uint32
	UseOutpointFormat bool
	BroadcastListen bool
	Autosense bool
	UserAgent string
	OutboundEvents chan events.Event
	InboundEvents chan events.Event
	Status int8
	mutex sync.Mutex
}

//CONVERT READS AND WRITES INTO FUNCTIONS THAT TAKE BYTES AND SET TIMEOUTS/ERRORS

func (pinger *PeerConnection) Start(bootstrapHash *chainhash.Hash, userAgent string) {

	//log.Printf("%s : STARTING CLIENT", pinger.PeerInfo.Address)

	var connectionAttempts uint8 = 0

	var messageMap map[string]wire.Message

	var magic = wire.BitcoinNet(pinger.MagicBytes)

	me := wire.NetAddress{
		Timestamp: time.Time{},
		Services:  0,
		IP:        net.ParseIP("8.8.8.8"),
		Port:      uint16(pinger.PeerInfo.Port),
	}

	you := wire.NetAddress{
		Timestamp: time.Time{},
		Services:  0,
		IP:        net.ParseIP(pinger.PeerInfo.Address),
		Port:      uint16(pinger.PeerInfo.Port),
	}

	version := wire.MsgVersion{
		ProtocolVersion: int32(pinger.ProtocolNumber),
		Services:        0,
		Timestamp:       time.Unix(time.Now().Unix(), 0),
		AddrYou:         you,
		AddrMe:          me,
		Nonce:           0xDEADBEEF,
		UserAgent:       userAgent,
		LastBlock:       0,
		DisableRelayTx:  true,
	}

	//tcpAddr, err := net.ResolveTCPAddr("tcp4", pinger.PeerInfo.Address + ":" + strconv.Itoa(int(pinger.PeerInfo.Port)))
	//if(err != nil) {
	//	log.Error(err)
	//	pinger.SetStatus(-1)
	//
	//	pinger.OutboundEvents <- Event{Type:PeerDisconnect, Data:pinger}
	//
	//	return
	//}

	//setup the ping inv map
	messageMap = make(map[string]wire.Message)

	for {

		if connectionAttempts >= 10 || len(pinger.InboundEvents) > 10 {
			log.Debug("Unable to connect -- closing connection.")
			pinger.SetStatus(-1)
			pinger.OutboundEvents <- events.Event{Type:events.PeerDisconnect, Data:pinger}
			return
		}

		d := net.Dialer{Timeout: time.Second * 5}
		conn, err := d.Dial("tcp", net.JoinHostPort(
			pinger.PeerInfo.Address, strconv.Itoa(int(pinger.PeerInfo.Port))))

		if (err != nil) {
			log.Debug(err)
			connectionAttempts += 10
			continue
		}

		var buf bytes.Buffer
		wire.WriteMessageN(&buf, &version, pinger.ProtocolNumber, magic)
		conn.SetWriteDeadline(time.Now().Add(30 * time.Second))
		conn.Write(buf.Bytes())

		bufReader := bufio.NewReader(conn)

		for {

			if pinger.GetStatus() < 0 {
				pinger.OutboundEvents <- events.Event{Type:events.PeerDisconnect, Data:pinger}
				return
			}

			//connection failed, set the status to -1 and let it be reaped
			if connectionAttempts >= 10 || len(pinger.InboundEvents) > 10 {
				log.Debug("Unable to connect -- closing connection.")
				pinger.SetStatus(-1)

				pinger.OutboundEvents <- events.Event{Type:events.PeerDisconnect, Data:pinger}

				return
			}

			//wait up to 2 minutes for a PING to come in
			conn.SetReadDeadline(time.Now().Add(2 * time.Minute))
			_, msg, _, errRead := wire.ReadMessageN(bufReader, pinger.ProtocolNumber, magic)

			if (errRead != nil) {
				if err, ok := errRead.(net.Error); ok && err.Timeout() {
					log.Error("Connection timeout. Bail.")
					pinger.OutboundEvents <- events.Event{Type:events.PeerDisconnect, Data:pinger}
					return
				}

				if strings.Contains(errRead.Error(), "unhandled command") {
					//log.Println(err)
					continue
				}

				log.Debugf("%s : %s", pinger.PeerInfo.Address, errRead)
				connectionAttempts++
				continue
			} else {

				log.Debug("COMMAND: ", msg.Command())

				connectionAttempts = 0

				if (msg.Command() == "inv") {
					inv := msg.(*wire.MsgInv)

					for _, inventory := range (inv.InvList) {

						log.Debug("INVENTORY TYPE: ", inventory.Type)

						if inventory.Type.String() == "MSG_BLOCK" {
							log.Debug("New block received: " + inventory.Hash.String())
							pinger.OutboundEvents <- events.Event{events.NewBlock, &inventory.Hash}
						}

						if inventory.Type == 14 && pinger.BroadcastListen {
							//MNANNOUNCE RECEIVED FOR OUR NODE
							getdata := wire.MsgGetData{}
							getdata.AddInvVect(inventory)

							var buf bytes.Buffer
							_, err := wire.WriteMessageN(&buf, &getdata, pinger.ProtocolNumber, magic)

							conn.SetWriteDeadline(time.Now().Add(30 * time.Second))
							_, err = conn.Write(buf.Bytes())
							if err, ok := err.(net.Error); ok && err.Timeout() {
								log.Error("Connection timeout. Bail.")
								pinger.OutboundEvents <- events.Event{Type:events.PeerDisconnect, Data:pinger}
								return
							}
						}

						if inventory.Type == 15 && pinger.Autosense {
							//PULL MNPS TO ENABLE AUTOSENSE
							getdata := wire.MsgGetData{}
							getdata.AddInvVect(inventory)

							var buf bytes.Buffer
							wire.WriteMessageN(&buf, &getdata, pinger.ProtocolNumber, magic)

							conn.SetWriteDeadline(time.Now().Add(30 * time.Second))
							_, err = conn.Write(buf.Bytes())
							if err, ok := err.(net.Error); ok && err.Timeout() {
								log.Error("Connection timeout. Bail.")
								pinger.OutboundEvents <- events.Event{Type:events.PeerDisconnect, Data:pinger}
								return
							}
						}
					}
				}

				if (msg.Command() == "version") {
					verack := wire.MsgVerAck{}

					var buf bytes.Buffer
					wire.WriteMessageN(&buf, &verack, pinger.ProtocolNumber, magic)

					conn.SetWriteDeadline(time.Now().Add(30 * time.Second))
					_, err = conn.Write(buf.Bytes())
					if err, ok := err.(net.Error); ok && err.Timeout() {
						log.Error("Connection timeout. Bail.")
						pinger.OutboundEvents <- events.Event{Type:events.PeerDisconnect, Data:pinger}
						return
					}

					pinger.SetStatus(1) //we're connected and ready to start pinging

					//ignore the request but relay our own 'getaddr' request
					getaddr := wire.MsgGetAddr{}

					var bufAddr bytes.Buffer
					wire.WriteMessageN(&bufAddr, &getaddr, pinger.ProtocolNumber, magic)

					conn.SetWriteDeadline(time.Now().Add(30 * time.Second))
					_, err = conn.Write(bufAddr.Bytes())
					if err, ok := err.(net.Error); ok && err.Timeout() {
						log.Error("Connection timeout. Bail.")
						pinger.OutboundEvents <- events.Event{Type:events.PeerDisconnect, Data:pinger}
						return
					}

					log.Debug("Sending getaddr")

					defaultHash := chainhash.Hash{}
					if *bootstrapHash != defaultHash {
						getblocks := wire.MsgGetBlocks{}

						getblocks.BlockLocatorHashes = []*chainhash.Hash{bootstrapHash}
						getblocks.ProtocolVersion = pinger.ProtocolNumber

						var bufBlocks bytes.Buffer
						wire.WriteMessageN(&bufBlocks, &getblocks, pinger.ProtocolNumber, magic)

						conn.SetWriteDeadline(time.Now().Add(30 * time.Second))
						_, err = conn.Write(bufBlocks.Bytes())
						if err, ok := err.(net.Error); ok && err.Timeout() {
							log.Error("Connection timeout. Bail.")
							pinger.OutboundEvents <- events.Event{Type:events.PeerDisconnect, Data:pinger}
							return
						}

						log.Debug("Sending getblocks to bootstrap")
					}
				}

				if (msg.Command() == "ping") {

					ping := msg.(*wire.MsgPing)

					pong := wire.MsgPong{Nonce: ping.Nonce}

					var buf bytes.Buffer
					wire.WriteMessageN(&buf, &pong, pinger.ProtocolNumber, magic)

					conn.SetWriteDeadline(time.Now().Add(30 * time.Second))
					_, err = conn.Write(buf.Bytes())
					if err, ok := err.(net.Error); ok && err.Timeout() {
						log.Error("Connection timeout. Bail.")
						pinger.OutboundEvents <- events.Event{Type:events.PeerDisconnect, Data:pinger}
						return
					}

					log.Info(pinger.PeerInfo.Address, " : PONG!")

					//clear out the message map
					for hash, message := range messageMap {
						if message.Command() == "mnp" {
							ping := message.(*wire.MsgMNP)
							pingTime := time.Unix(int64(ping.SigTime), 0)
							//if the ping is more than 5 minutes old, delete it
							if pingTime.Add(time.Minute * 5).Before(time.Now().UTC()) {
								delete(messageMap, hash)
							}
						}

						if message.Command() == "mnb" {
							mnb := message.(*wire.MsgMNB)
							pingTime := time.Unix(int64(mnb.LastPing.SigTime), 0)
							//if the ping is more than 5 minutes old, delete it
							if pingTime.Add(time.Minute * 5).Before(time.Now().UTC()) {
								delete(messageMap, hash)
							}
						}
					}
				}

				if (msg.Command() == "addr") {
					msgAddr := msg.(*wire.MsgAddr)
					for _, addr := range msgAddr.AddrList {
						//log.Println("PEER: ", addr.IP, ":", addr.Port)
						pinger.OutboundEvents <- events.Event{events.NewAddr,addr}
					}
				}

				//let broadcast channels relay back broadcasts
				if (msg.Command() == "mnb") {
					mnb := msg.(*wire.MsgMNB)
					log.Debug("Masternode broadcast detected for: ", mnb.Vin.PreviousOutPoint.String())
					pinger.OutboundEvents <- events.Event{events.NewMasternodeBroadcast,mnb}

				}

				//let broadcast channels relay back broadcasts
				if msg.Command() == "mnp" && pinger.Autosense {
					mnp := msg.(*wire.MsgMNP)
					log.Debug("Masternode ping detected. Sending back to daemon for analysis.")
					pinger.OutboundEvents <- events.Event{events.NewMasternodePing,mnp}
				}

				//non-blocking select
				select {
				case event := <-pinger.InboundEvents:
					switch event.Type {
					case events.NewPhantomPing:

						ping := event.Data.(*generator.MasternodePing)

						log.Debug("Relaying ping to the network for: ", ping.Name)

						mnp := ping.GenerateMasternodePing(ping.UseOutpoint, pinger.SentinelVersion, pinger.DaemonVersion)

						//check to see if this is a broadcast relay
						if ping.BroadcastTemplate != nil {
							//USING BROADCAST TEMPLATE

							mnb := *ping.BroadcastTemplate
							mnb.LastPing = mnp

							inv := wire.MsgInv{}
							invVec := wire.InvVect{}
							invVec.Type = 14
							invVec.Hash = mnb.GetHash()
							inv.AddInvVect(&invVec)

							var buf bytes.Buffer
							wire.WriteMessageN(&buf, &inv, pinger.ProtocolNumber, magic)

							byteData := buf.Bytes()

							conn.SetWriteDeadline(time.Now().Add(30 * time.Second))
							_, err = conn.Write(byteData)
							if err, ok := err.(net.Error); ok && err.Timeout() {
								log.Error("Connection timeout. Bail.")
								pinger.OutboundEvents <- events.Event{Type:events.PeerDisconnect, Data:pinger}
								return
							}

							messageMap[invVec.Hash.String()] = &mnb
						}

						log.Debug(mnp.Vin.PreviousOutPoint.String())
						pingTime := time.Unix(int64(mnp.SigTime), 0)
						log.Debug("Ping time: ", pingTime.UTC().String())

						//ALWAYS SEND THE PINGS
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

						conn.SetWriteDeadline(time.Now().Add(30 * time.Second))
						_, err = conn.Write(buf.Bytes())
						if err, ok := err.(net.Error); ok && err.Timeout() {
							log.Error("Connection timeout. Bail.")
							pinger.OutboundEvents <- events.Event{Type:events.PeerDisconnect, Data:pinger}
							return
						}

						//store the ping
						messageMap[invVec.Hash.String()] = &mnp
					default:
						//log.Info("That shouldn't happen", event)
					}

				default:
					//log.Info("No message received.")
				}

				//this should really be a hashMap with expiring entries
				if (msg.Command() == "getdata") {

					getData := msg.(*wire.MsgGetData)

					for _, inv := range getData.InvList {
						//check the map
						str := inv.Hash.String()
						if val, ok := messageMap[str]; ok {
							//do something here
							var buf bytes.Buffer
							wire.WriteMessageN(&buf, val, pinger.ProtocolNumber, magic)

							conn.SetWriteDeadline(time.Now().Add(30 * time.Second))
							_, err = conn.Write(buf.Bytes())
							if err, ok := err.(net.Error); ok && err.Timeout() {
								log.Error("Connection timeout. Bail.")
								pinger.OutboundEvents <- events.Event{Type:events.PeerDisconnect, Data:pinger}
								return
							}

							log.Debug("Inv sent successfully.")
						} else {
							log.Error("Hash not found - all is well.")
						}
					}
				}
			}
		}

		//we've disconnected, so try again
		connectionAttempts++
		log.Printf("%s : There's been an error, attempting to reconnect.", pinger.PeerInfo.Address)
		time.Sleep(1 * time.Minute)
	}
}

func (pinger *PeerConnection) SetStatus(status int8) {
	pinger.mutex.Lock()
	defer pinger.mutex.Unlock()

	pinger.Status = status
}

func (pinger *PeerConnection) GetStatus() (int8) {
	pinger.mutex.Lock()
	defer pinger.mutex.Unlock()

	return pinger.Status
}