package phantom

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
)

type CoinConf struct {
	Name                string `json:"name"`
	Magicbytes          string `json:"magicbytes"`
	Port                uint    `json:"port"`
	ProtocolNumber      uint    `json:"protocol_number"`
	MagicMessage        string `json:"magic_message"`
	MagicMessageNewline bool   `json:"magic_message_newline,omitempty"`
	BootstrapURL        string `json:"bootstrap_url,omitempty"`
	SentinelVersion     string `json:"sentinel_version,omitempty""`
	DaemonVersion       string `json:"daemon_version,omitempty""`
	BootstrapIPs        string `json:"bootstrap_ips,omitempty""`
	UserAgent           string `json:"user_agent,omitempty""`
}

func LoadCoinConf(path string) (CoinConf, error) {
	var coinConf CoinConf

	coinConfJson, err := os.Open(path)
	defer coinConfJson.Close()
	if err != nil {
		log.Println(err)
		return CoinConf{}, err
	}

	bytes, err := ioutil.ReadAll(coinConfJson)
	if err != nil {
		log.Println(err)
		return CoinConf{}, err
	}


	err = json.Unmarshal(bytes, &coinConf)
	if err != nil {
		log.Println(err)
		return CoinConf{}, err
	}

	return coinConf, nil
}
