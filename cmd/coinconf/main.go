package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"../../src/phantom"
)

var coinName string
var gitUrl string
var gitParts []string
var explorer string

//SIMPLE UTILITY TO GENERATE A COINCONF FOR A GIVEN COIN.
func main() {
	coinConf := phantom.CoinConf{}

	flag.StringVar(&coinName, "coin_name", "", "the name of the coin")
	flag.StringVar(&gitUrl, "git_hub", "", "the git url")
	flag.StringVar(&explorer, "explorer", "", "the bootstrap explorer")

	flag.Parse()

	gitParts = strings.Split(gitUrl, "/")

	if len(gitParts) == 5 {
		gitParts = append(gitParts, "tree")
		gitParts = append(gitParts, "master")
	}

	name := strings.ToUpper(coinName)
	magicBytes := LoadMagicBytes()
	port := LoadPort()
	magicMessage := LoadMagicMessage()
	protocolVersion := LoadProtocolVersion()
	sentinelVersion := LoadSentinelVersion()
	daemonVersion := LoadDaemonVersion()

	coinConf.Name = name

	if explorer != "" {
		coinConf.BootstrapURL = explorer
	}

	coinConf.Magicbytes = magicBytes
	coinConf.MagicMessage = magicMessage
	coinConf.MagicMessageNewline = true

	parsedPort, err := strconv.Atoi(port)
	if err != nil {
		log.Fatal("Error parsing port")
	}
	coinConf.Port = uint(parsedPort)

	parsedProtocl, err := strconv.Atoi(protocolVersion)
	if err != nil {
		log.Fatal("Error parsing protocol version")
	}
	coinConf.ProtocolNumber = uint(parsedProtocl)

	if sentinelVersion != "" {
		coinConf.SentinelVersion = ConvertVersionHexToString(sentinelVersion)
	}

	if daemonVersion != "" {
		coinConf.DaemonVersion = ConvertVersionHexToString(daemonVersion)
	}

	coinConfJson, err := json.Marshal(coinConf)
	if err != nil {
		log.Fatal("Error building json")
	}

	err = ioutil.WriteFile(strings.ToLower(coinName) + ".json", coinConfJson, 0644)
	if err != nil {
		fmt.Println(err)
	}
}

func LoadMagicBytes() string {
	data, err := LoadFile(UrlForFile("chainparams.cpp"))
	if err != nil {
		log.Fatal()
	}

	re := regexp.MustCompile(`pchMessageStart\[\d\] = 0x(..);`)
	result := re.FindAllStringSubmatch(data, 4)

	return strings.ToUpper(result[3][1] + result[2][1] + result[1][1] + result[0][1])
}

func LoadPort() string {
	data, err := LoadFile(UrlForFile("chainparams.cpp"))
	if err != nil {
		log.Fatal()
	}

	re := regexp.MustCompile(`nDefaultPort = (\d+);`)
	return re.FindStringSubmatch(data)[1]
}

func LoadMagicMessage() string {
	data, err := LoadFile(UrlForFile("validation.cpp"))
	if err != nil {
		log.Fatal()
	}

	re := regexp.MustCompile(`const .*string strMessageMagic = "(.*)\\n";`)
	matches := re.FindStringSubmatch(data)

	if len(matches) > 0 {
		return matches[1]
	}

	data, err = LoadFile(UrlForFile("main.cpp"))
	if err != nil {
		log.Fatal()
	}

	matches = re.FindStringSubmatch(data)
	if len(matches) > 0 {
		return matches[1]
	}

	log.Fatal("No magic message found.")
	return ""
}

func LoadProtocolVersion() string {
	data, err := LoadFile(UrlForFile("version.h"))
	if err != nil {
		log.Fatal()
	}

	re := regexp.MustCompile(`static const int PROTOCOL_VERSION = (\d+)`)
	return re.FindStringSubmatch(data)[1]
}

func LoadSentinelVersion() string {
	data, err := LoadFile(UrlForFile("masternode.h"))
	if err != nil {
		log.Fatal()
	}

	re := regexp.MustCompile(`#define MIN_SENTINEL_VERSION 0x(\d+)`)
	matches := re.FindStringSubmatch(data)
	if len(matches) > 0 {
		return matches[1]
	}

	re = regexp.MustCompile(`#define DEFAULT_SENTINEL_VERSION 0x(\d+)`)
	matches = re.FindStringSubmatch(data)
	if len(matches) > 0 {
		return matches[1]
	}

	data, err = LoadFile(UrlForFile("clientversion.h"))
	matches = re.FindStringSubmatch(data)
	if err != nil {
		log.Fatal()
	}

	re = regexp.MustCompile(`#define CLIENT_SENTINEL_VERSION (\d+)`)
	if len(matches) > 0 {
		return matches[1]
	}

	return ""
}

func LoadDaemonVersion() string {
	data, err := LoadFile(UrlForFile("clientversion.h"))
	if err != nil {
		log.Fatal()
	}

	re := regexp.MustCompile(`#define CLIENT_MASTERNODE_VERSION (\d+)`)
	matches := re.FindStringSubmatch(data)
	if len(matches) > 0 {
		return matches[1]
	}

	return ""
}

func LoadFile(url string) (string, error) {
	response, err := http.Get(url)
	if err != nil {
		log.Printf("%s", err)
		return "", err
	}
	defer response.Body.Close()
	bytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Printf("%s", err)
		return "", err
	}

	return string(bytes), err
}

func UrlForFile(file string) string {
	url := "https://raw.githubusercontent.com/" + gitParts[3] + "/" + gitParts[4] + "/" + gitParts[6] + "/src/" + file
	return url
}

func ConvertVersionHexToString(str string) string {
	result := ""
	for i := 0; i<len(str); i += 2 {
		result += str[i:i+2]
	}
	return result
}
