package dnsseed

import (
	log "github.com/sirupsen/logrus"
	"net"
	"os"
	"phantom/cmd/refactor/database"
	"time"
)

func LoadDnsSeeds(nshost string, defaultPort uint32) (peers []database.Peer) {
	ips, err := net.LookupIP(nshost)
	if err != nil {
		log.Error(os.Stderr, "Could not get IPs: %v\n", err)
	}
	for _, ip := range ips {
		peers = append(peers, database.Peer{Address:ip.String(), Port:defaultPort, LastSeen:time.Now()})
	}
	return peers
}