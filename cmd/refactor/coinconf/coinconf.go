package coinconf

import (
	"encoding/json"
	"io/ioutil"
	log "github.com/sirupsen/logrus"
	"os"
)

type CoinConf struct {
	Name                string `json:"name"`
	MaxConnections		*uint32 `json:max_connections,omitempty`
	Magicbytes          string `json:"magicbytes"`
	Port                uint32 `json:"port"`
	ProtocolNumber      uint32 `json:"protocol_number"`
	MagicMessage        string `json:"magic_message"`
	MagicMessageNewline *bool   `json:"magic_message_newline,omitempty"`
	BootstrapURL        string `json:"bootstrap_url,omitempty"`
	SentinelVersion     string `json:"sentinel_version,omitempty""`
	DaemonVersion       string `json:"daemon_version,omitempty""`
	BootstrapIPs        string `json:"bootstrap_ips,omitempty""`
	BootstrapHash       string `json:"bootstrap_hash,omitempty""`
	UserAgent           string `json:"user_agent,omitempty""`
	DNSSeeds			string `json:"dns_seeds,omitempty""`
	BroadcastListen     *bool   `json:"broadcast_listen,omitempty""`
	MasternodeConf      string `json:"masternode_conf,omitempty""`
	Autosense	        *bool   `json:"autosense,omitempty""`
}

func LoadCoinConf(path string) (CoinConf, error) {
	var coinConf CoinConf

	coinConfJson, err := os.Open(path)
	defer coinConfJson.Close()
	if err != nil {
		log.Fatal(err)
		return CoinConf{}, err
	}

	bytes, err := ioutil.ReadAll(coinConfJson)
	if err != nil {
		log.Fatal(err)
		return CoinConf{}, err
	}

	err = json.Unmarshal(bytes, &coinConf)
	if err != nil {
		log.Fatal(err)
		return CoinConf{}, err
	}

	return coinConf, nil
}