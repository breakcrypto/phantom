package main

import (
	"flag"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	log "github.com/sirupsen/logrus"
	"phantom/cmd/refactor/analyzer"
	"phantom/cmd/refactor/blockqueue"
	"phantom/cmd/refactor/broadcaststore"
	"phantom/cmd/refactor/coinconf"
	"phantom/cmd/refactor/database"
	"phantom/cmd/refactor/dnsseed"
	"phantom/cmd/refactor/events"
	"phantom/cmd/refactor/generator"
	"phantom/cmd/refactor/remotechains"
	"phantom/pkg/phantom"
	"phantom/pkg/socket/wire"
	"strconv"
	"strings"
	"time"
)

type PeerCollection struct {
	PeerConnections []*PeerConnection
}

type PhantomDaemon struct {
	MaxConnections uint
	BootstrapIPs string
	DNSSeeds string
	BootstrapHash chainhash.Hash
	BootstrapExplorer string
	MasternodeConf string
	CoinCon phantom.CoinConf
	DefaultPort uint
	PeerConnections []database.Peer
	BroadcastListen bool
	//for the peers
	PeerConnectionTemplate PeerConnection
}

func (p *PeerCollection) RemovePeer(peerToRemove *PeerConnection) {
	for i, peer := range p.PeerConnections {
		if peer.PeerInfo.Address == peerToRemove.PeerInfo.Address && peer.PeerInfo.Port == peerToRemove.PeerInfo.Port {
			p.PeerConnections = append(p.PeerConnections[:i], p.PeerConnections[i+1:]...)
		}
	}
}

func (p *PeerCollection) AddPeer(peerToAdd *PeerConnection) {
	p.PeerConnections = append(p.PeerConnections, peerToAdd)
}

func (p *PeerCollection) Contains(peerToCheck *database.Peer) bool {
	for _, peer := range p.PeerConnections {
		if peer.PeerInfo.Address == peerToCheck.Address && peer.PeerInfo.Port == peerToCheck.Port {
			return true
		}
	}
	return false
}

var peerCollection PeerCollection

func init() {
	// Only log the warning severity or above.
	//log.SetLevel(log.DebugLevel)
}

func main() {
	const VERSION = "0.1.0"

	var done chan bool

	phantom.Preamble(VERSION)
	time.Sleep(10 * time.Second)

	phantomDaemon := PhantomDaemon{}

	var magicHex string
	var magicMsgNewLine bool
	var protocolNum uint
	var bootstrapHashStr string
	var sentinelString string
	var daemonString string
	var coinConfString string
	var debugLogging bool

	flag.StringVar(&coinConfString, "coin_conf", "", "Name of the file to load the coin information from.")

	flag.StringVar(&phantomDaemon.MasternodeConf, "masternode_conf",
		"masternode.txt",
		"Name of the file to load the masternode information from.")

	flag.UintVar(&phantomDaemon.MaxConnections, "max_connections",
		10,
		"the number of peers to maintain")

	flag.StringVar(&magicHex, "magicbytes",
		"",
		"a hex string for the magic bytes")

	flag.UintVar(&phantomDaemon.DefaultPort, "port",
		0,
		"the default port number")

	flag.UintVar(&protocolNum, "protocol_number",
		0,
		"the protocol number to connect and ping with")

	flag.StringVar(&phantomDaemon.PeerConnectionTemplate.MagicMessage, "magic_message",
		"",
		"the signing message")

	flag.BoolVar(&magicMsgNewLine,
		"magic_message_newline",
		true,
		"add a new line to the magic message")

	flag.StringVar(&phantomDaemon.BootstrapIPs, "bootstrap_ips",
		"",
		"IP addresses to bootstrap the network (i.e. \"1.1.1.1:1234,2.2.2.2:1234\")")

	flag.StringVar(&phantomDaemon.DNSSeeds, "dns_seeds",
		"",
		"DNS seed addresses to bootstrap the network (i.e. \"dns.coin.com,dns1.coin.net\")")

	flag.StringVar(&bootstrapHashStr, "bootstrap_hash",
		"",
		"Hash to bootstrap the pings with ( top - 12 )")

	flag.StringVar(&phantomDaemon.BootstrapExplorer,
		"bootstrap_url",
		"",
		"Explorer to bootstrap from.")

	flag.StringVar(&sentinelString,
		"sentinel_version",
		"",
		"The string to use for the sentinel version number (i.e. 1.20.0)")

	flag.StringVar(&daemonString,
		"daemon_version",
		"",
		"The string to use for the sentinel version number (i.e. 1.20.0)")

	flag.StringVar(&phantomDaemon.PeerConnectionTemplate.UserAgent,
		"user_agent",
		"@_breakcrypto's phantoms",
		"The user agent string to connect to remote peers with.")

	flag.BoolVar(&phantomDaemon.BroadcastListen,
		"broadcast_listen",
		false,
		"If set to true, the phantom will listen for new broadcasts and cache them for 4 hours.")

	flag.BoolVar(&phantomDaemon.PeerConnectionTemplate.Autosense,
		"autosense",
		true,
		"If set to true, the phantom will listen for new broadcasts and cache them for 4 hours.")

	flag.BoolVar(&debugLogging,
		"debug",
		false,
		"Enable debug output.")

	flag.Parse()

	if debugLogging {
		log.SetLevel(log.DebugLevel)
	}

	var coinConf = coinconf.CoinConf{}

	if coinConfString != "" {
		var err error
		coinConf, err = coinconf.LoadCoinConf(coinConfString)
		if err != nil {
			log.Fatal(err)
		} else {
			if phantomDaemon.MasternodeConf == "" {
				phantomDaemon.MasternodeConf = coinConf.MasternodeConf
			}

			if coinConf.MaxConnections != nil {
				phantomDaemon.MaxConnections = uint(*coinConf.MaxConnections)
			}

			if magicHex == "" {
				magicHex = coinConf.Magicbytes
			}

			if phantomDaemon.DefaultPort == 0 {
				phantomDaemon.DefaultPort = uint(coinConf.Port)
			}

			if protocolNum == 0 {
				protocolNum = uint(coinConf.ProtocolNumber)
			}

			if phantomDaemon.PeerConnectionTemplate.MagicMessage == "" {
				phantomDaemon.PeerConnectionTemplate.MagicMessage = coinConf.MagicMessage
			}

			if magicMsgNewLine && coinConf.MagicMessageNewline != nil {
				magicMsgNewLine = *coinConf.MagicMessageNewline
			}

			if phantomDaemon.BootstrapIPs == "" {
				phantomDaemon.BootstrapIPs = coinConf.BootstrapIPs
			}

			if phantomDaemon.DNSSeeds == "" {
				phantomDaemon.DNSSeeds = coinConf.DNSSeeds
			}

			if bootstrapHashStr == "" {
				bootstrapHashStr = coinConf.BootstrapHash
			}

			if phantomDaemon.BootstrapExplorer == "" {
				phantomDaemon.BootstrapExplorer = coinConf.BootstrapURL
			}

			if sentinelString == "" {
				sentinelString = coinConf.SentinelVersion
			}

			if daemonString == "" {
				daemonString = coinConf.DaemonVersion
			}

			if phantomDaemon.PeerConnectionTemplate.UserAgent == "@_breakcrypto's phantoms" &&
				coinConf.UserAgent != "" {
				phantomDaemon.PeerConnectionTemplate.UserAgent = coinConf.UserAgent
			}

			if !phantomDaemon.BroadcastListen && coinConf.BroadcastListen != nil {
				phantomDaemon.BroadcastListen = *coinConf.BroadcastListen
			}

			if phantomDaemon.PeerConnectionTemplate.Autosense && coinConf.Autosense != nil {
				phantomDaemon.PeerConnectionTemplate.Autosense = *coinConf.Autosense
			}
		}
	}

	magicMsgNewLine = true

	magicBytes64, _ := strconv.ParseUint(magicHex, 16, 32)
	phantomDaemon.PeerConnectionTemplate.MagicBytes = uint32(magicBytes64)

	phantomDaemon.PeerConnectionTemplate.ProtocolNumber = uint32(protocolNum)

	if sentinelString != "" {
		phantomDaemon.PeerConnectionTemplate.SentinelVersion = phantom.ConvertVersionStringToInt(sentinelString)
	}

	if daemonString != "" {
		//fmt.Println("ENABLING DAEMON.")
		phantomDaemon.PeerConnectionTemplate.DaemonVersion = phantom.ConvertVersionStringToInt(daemonString)
	}

	if magicMsgNewLine {
		phantomDaemon.PeerConnectionTemplate.MagicMessage = phantomDaemon.PeerConnectionTemplate.MagicMessage + "\n"
	}

	if phantomDaemon.BootstrapIPs != "" {
		database.GetInstance().StorePeers(phantom.SplitAddressList(phantomDaemon.BootstrapIPs))
	}

	if bootstrapHashStr != "" {
		chainhash.Decode(&phantomDaemon.BootstrapHash,bootstrapHashStr)
	}

	log.WithFields(log.Fields{
		"masternode_conf":  phantomDaemon.MasternodeConf,
		"magic_bytes":      strings.ToUpper(strconv.FormatInt(
			int64(phantomDaemon.PeerConnectionTemplate.MagicBytes), 16)),
		"magic_message":    phantomDaemon.PeerConnectionTemplate.MagicMessage,
		"protocol_number":  phantomDaemon.PeerConnectionTemplate.ProtocolNumber,
		"bootstrap_ips":    phantomDaemon.BootstrapIPs,
		"bootstrap_url":    phantomDaemon.BootstrapExplorer,
		"bootstrap_hash":   phantomDaemon.BootstrapHash.String(),
		"autosense":        phantomDaemon.PeerConnectionTemplate.Autosense,
		"broadcast_listen": phantomDaemon.BroadcastListen,
		"daemon_version":   phantomDaemon.PeerConnectionTemplate.DaemonVersion,
		"sentinel_version": phantomDaemon.PeerConnectionTemplate.SentinelVersion,
		"user_agent":       phantomDaemon.PeerConnectionTemplate.UserAgent,
		"dns_seeds":        phantomDaemon.DNSSeeds,
		"default_port":     phantomDaemon.DefaultPort,
	}).Info("Using the following settings.")

	phantomDaemon.Start()

	//wait
	<-done
}

func (p *PhantomDaemon) Start() {

	//load the peer database
	peerdb := database.GetInstance()
	queue := blockqueue.GetInstance()

	//allocate the peer channels
	peerChannels := p.allocatePeerChannels(p.MaxConnections)

	//setup the event channel that all peers will broadcast to
	var daemonEventChannel = make(chan events.Event)

	//load the bootstrap values
	//load blockhash
	//load peers

	var hash chainhash.Hash

	if p.BootstrapExplorer != "" {
		var bootstrap remotechains.RemoteChain = remotechains.IquidusExplorer{BaseURL:p.BootstrapExplorer}
		peers, err := bootstrap.GetPeers(uint32(p.DefaultPort))
		if err != nil {
			log.Error("Failed to load bootstrap peers")
		}
		peerdb.StorePeers(peers)

		height, err := bootstrap.GetChainHeight()
		if err != nil {
			log.Error("Failed to load bootstrap height")
		}
		hash, err = bootstrap.GetBlockHash(uint64(height-12))
		if err != nil {
			log.Error("Failed to load bootstrap height")
		}
		log.WithFields(log.Fields{
			"hash": hash.String(),
		}).Info("Bootstrap hash value")
	}

	defaultHash := chainhash.Hash{}
	if hash == defaultHash {
		hash = p.BootstrapHash
	}

	//force the bootstrap
	queue.ForceHash(hash)

	//load the dnsseeds if there are any
	if p.DNSSeeds != "" {
		for _, seed := range strings.Split(p.DNSSeeds, ",") {
			peerdb.StorePeers(dnsseed.LoadDnsSeeds(seed, uint32(p.DefaultPort)))
		}
	}

	//start the analyzer
	mnpAnalyzer := analyzer.GetInstance()
	mnpAnalyzer.Threshold = 10

	//start processing events before spawning off peers
	go p.processEvents(daemonEventChannel)

	//spawn off the peers
	peers := peerdb.GetRandomPeers(p.MaxConnections)
	for i, peer := range peers {
		peerConn := PeerConnection{
			MagicBytes: p.PeerConnectionTemplate.MagicBytes,
			ProtocolNumber: p.PeerConnectionTemplate.ProtocolNumber,
			MagicMessage: p.PeerConnectionTemplate.MagicMessage,
			PeerInfo:peer,
			InboundEvents:peerChannels[i],
			OutboundEvents:daemonEventChannel,
			SentinelVersion:p.PeerConnectionTemplate.SentinelVersion,
			DaemonVersion:p.PeerConnectionTemplate.DaemonVersion,
			UseOutpointFormat:p.PeerConnectionTemplate.UseOutpointFormat,
			Autosense:p.PeerConnectionTemplate.Autosense,
		}

		log.WithField("peer ip", peer).Debug("Starting new peer.")
		go peerConn.Start(&hash, "break")
		peerCollection.AddPeer(&peerConn)
	}

	//start the ping generator
		//auto-sense enabled?
		//if so don't start sending pings until we've reached consensus

	//TODO Make this a channel gate vs. polling
	for p.PeerConnectionTemplate.Autosense {
		//do nothing until the auto-sense finishes
		time.Sleep(time.Second * 10)
		log.Info("Sleeping, waiting on autosense to complete.")
	}

	time.Sleep(time.Second * 30)

	go p.generatePings(peerChannels...)
}

func (p *PhantomDaemon) processEvents(eventChannel chan events.Event) {
	for event := range eventChannel {
		//process the event
		switch event.Type {
		case events.NewMasternodePing:
			mnp := event.Data.(*wire.MsgMNP)
			p.processNewMasternodePing(mnp)
		case events.NewMasternodeBroadcast:
			mnb := event.Data.(*wire.MsgMNB)
			broadcaststore.GetInstance().StoreBroadcast(mnb)
		case events.NewBlock:
			hash := event.Data.(*chainhash.Hash)
			//log.WithField("hash", hash.String()).Info("New block.")
			blockqueue.GetInstance().AddHash(*hash)
		case events.NewAddr:
			addr := event.Data.(*wire.NetAddress)
			//log.WithField("addr", addr.IP).Debug("New address found. Saving.")
			database.GetInstance().StorePeer(database.Peer{Address:addr.IP.String(), Port:uint32(addr.Port), LastSeen:time.Now()})
		case events.PeerDisconnect:
			peer := event.Data.(*PeerConnection)
			log.WithField("ip", peer.PeerInfo.Address).Debug("Handled peer disconnection.")
			p.processPeerDisconenct(peer)
		}
	}
}

func (p *PhantomDaemon) allocatePeerChannels(numConn uint) []chan events.Event {
	var peerChannels []chan events.Event
	for i := 0; i < int(numConn); i++ {
		peerChannels = append(peerChannels, make(chan events.Event, 1500))
	}
	return peerChannels
}

func (p *PhantomDaemon) processNewMasternodePing(ping *wire.MsgMNP) {
	log.Debug("Analyzing ping.")
	//we have a new ping
	if analyzer.GetInstance().AnalyzePing(ping) {
		//we have enough to analyze and make a decent guess at the network settings
		useOut, sentinel, daemon := analyzer.GetInstance().GetResults()

		log.Info("-------------------------")
		log.Info("--- CONSENSUS REACHED ---")
		log.Info("-------------------------")
		log.WithFields(log.Fields{
			"Outpoint form":useOut,
			"Sentinel version":sentinel,
			"Daemon version":daemon,
		}).Info("Consensus reached.")
		log.Info("-------------------------")
		log.Info("-------------------------")

		//set the daemon settings for new peer connections
		p.PeerConnectionTemplate.Autosense = false
		p.PeerConnectionTemplate.UseOutpointFormat = useOut
		p.PeerConnectionTemplate.SentinelVersion = sentinel
		p.PeerConnectionTemplate.DaemonVersion = daemon

		//update existing peers
		for _, peer := range peerCollection.PeerConnections {
			peer.Autosense = false
			peer.UseOutpointFormat = p.PeerConnectionTemplate.UseOutpointFormat
			peer.SentinelVersion = p.PeerConnectionTemplate.SentinelVersion
			peer.DaemonVersion = p.PeerConnectionTemplate.DaemonVersion
		}
	}
}

func (p *PhantomDaemon)  generatePings(channels ...chan events.Event) {
	for {
		startTime := time.Now()
		generator.GeneratePingsFromMasternodeFile(
			p.MasternodeConf,
			p.PeerConnectionTemplate.MagicMessage,
			p.PeerConnectionTemplate.UseOutpointFormat,
			p.PeerConnectionTemplate.SentinelVersion,
			p.PeerConnectionTemplate.DaemonVersion,
			channels...,
		)
		sleepTime := time.Now().Add(time.Minute * 10).Sub(startTime)

		//sleep for the remaining 10 minutes, if there are any.
		if sleepTime > 0 {
			time.Sleep(sleepTime)
		}
	}
}

func (p *PhantomDaemon) processPeerDisconenct(peer *PeerConnection) {
	//a peer has closed out
	peerCollection.RemovePeer(peer)

	//turn the channel to not accepting new events (wrap, set a status code)

	//bleed the events out of the channel if they exist

	//now take the pinger channel from it and reuse in a newly created peer
	newPeer := p.spawnNewPeer(peer.InboundEvents, peer.OutboundEvents)
	peerCollection.AddPeer(newPeer)
	go newPeer.Start(blockqueue.GetInstance().GetTop(), "break")
}

func (p *PhantomDaemon) spawnNewPeer(inboundEventChannel chan events.Event, outboundEventChannel chan events.Event) *PeerConnection {
	//get a new address from the database
	peerdb := database.GetInstance()

	var peerInfo database.Peer
	for peerInfo = peerdb.GetRandomPeer(); peerCollection.Contains(&peerInfo); peerInfo = peerdb.GetRandomPeer() {
		//loop until it fills
		//fmt.Println(peerInfo.Address)
	}

	log.WithField("ip", peerInfo.Address).Debug("Spawned new peer.")

	//drain the channel before assigning to remove any stale pings
	drainChannel(inboundEventChannel)

	peer := PeerConnection{
		MagicBytes: p.PeerConnectionTemplate.MagicBytes,
		ProtocolNumber: p.PeerConnectionTemplate.ProtocolNumber,
		MagicMessage: p.PeerConnectionTemplate.MagicMessage,
		PeerInfo:peerInfo,
		InboundEvents:inboundEventChannel,
		OutboundEvents:outboundEventChannel,
		SentinelVersion:p.PeerConnectionTemplate.SentinelVersion,
		DaemonVersion:p.PeerConnectionTemplate.DaemonVersion,
		UseOutpointFormat:p.PeerConnectionTemplate.UseOutpointFormat,
		Autosense:p.PeerConnectionTemplate.Autosense,
	}

	return &peer
}

func drainChannel(channel chan events.Event) {
	for len(channel) > 0 {
		<- channel
	}
}