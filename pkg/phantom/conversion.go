package phantom

import (
	"errors"
	log "github.com/sirupsen/logrus"
	"net"
	"phantom/cmd/refactor/database"
	"strconv"
	"strings"
	"time"
	"phantom/pkg/socket/wire"
)

func ConvertVersionStringToInt(str string) uint32 {
	version := 0
	parts := strings.Split(str, ".")
	for _, part := range parts {
		version <<= 8
		value, _ := strconv.Atoi(part)
		version |= value
	}
	return uint32(version)
}

func SplitAddress(pair string) (wire.NetAddress, error) {
	ipPort := strings.Split(pair, ":")
	if len(ipPort) != 2 {
		return wire.NetAddress{}, errors.New("invalid ip:port pair")
	}
	ip := ipPort[0]
	port, _ := strconv.Atoi(ipPort[1])
	return wire.NetAddress{time.Now(),
		0,
		net.ParseIP(ip),
		uint16(port)}, nil
}

func SplitAddressList(bootstraps string) (peers []database.Peer) {
	for _, bootstrap := range strings.Split(bootstraps, ",") {
		host, port, err := net.SplitHostPort(bootstrap)
		if err != nil {
			log.Error(err)
		}
		portVal, err := strconv.Atoi(port)
		if err != nil {
			log.Error(err)
		}
		peers = append(peers, database.Peer{Address:host, Port:uint32(portVal), LastSeen:time.Now()})
	}
	return peers
}
