# Phantom-node Daemon

Phantom nodes requires no static IP address, no copy of the blockchain, and no proof-of-service. As such, you can run a node on any IP address of your liking: `1.1.1.1` or `8.8.8.8` if you wish. The phantom daemon is extremely lightweight allowing you to run hundreds of nodes from a modest machine if you wished. And, possibly most importantly, you can move your currently running masternodes to phantom nodes without restarting since a real IP address is no longer a requirement.

The phantom daemon is custom built wallet designed to replicate only what is required for pre-EVO masternodes to run; it replaces the masternode daemon piece. It does not handle any wallet private keys and has no access to your coins. You will still need a wallet to start your masternodes, but once started, the phatom node system will handle the rest for you.

## Contact information

* email: breakcrypto@gmail.com
* twitter: @_breakcrypto
* discord: breakcrypto#0011

## Donation Addresses
BTC: 151HTde9NgwbMMbMmqqpJYruYRL4SLZg1S

LTC: LhBx1TUyp7wiYuMxjefAGUGZVzuHRtPBA7

DOGE: DBahutcjEAxfwQEW7kzft2y8dhZN2VtcG5

# A note from the developer

Phantoms have been released to make it easier, and less costly, for masternode supporters to host their own nodes. Masternode hosting companies are free to utilize the phantom system as long as they comply with the terms of the Server Side Public License. 

# Downloads

* [Windows](https://github.com/breakcrypto/phantom/releases/download/v0.0.1/phantom-windows-amd64.exe)
* [Linux](https://github.com/breakcrypto/phantom/releases/download/v0.0.1/phantom-linux-amd64)
* [OSX](https://github.com/breakcrypto/phantom/releases/download/v0.0.1/phantom-darwin-amd64)
* [ARM](https://github.com/breakcrypto/phantom/releases/download/v0.0.1/phantom-linux-arm)

# Setup 

The setup is simple: copy your masternode.conf, modify it slightly, launch the phantom executable.

## Masternode.txt setup

Copy your masternode.conf to the same folder as the phantom executable. Rename it to masternode.txt. Remove any comment lines from the top of the file (i.e. delete any line starting with #). At the end of each line add a epoch time ( https://www.unixtimestamp.com ). The epoch timestamp is utilized to allow you to run multiple phantom node setups in a deterministic manner, creating a highly-available configuration.

**Example**

`masternode.conf`
```
# Masternode config file
# Format: alias IP:port masternodeprivkey collateral_output_txid collateral_output_index
mn1 45.50.22.125:17817 73HaYBVUCYjEMeeH1Y4sBGLALQZE1Yc1K64xiqgX37tGBDQL8Xg 2bcd3c84c84f87eaa86e4e56834c92927a07f9e18718810b92e0d0324456a67c 1
```

becomes:

`masternode.txt`
```
mn1 45.50.22.125:17817 73HaYBVUCYjEMeeH1Y4sBGLALQZE1Yc1K64xiqgX37tGBDQL8Xg 2bcd3c84c84f87eaa86e4e56834c92927a07f9e18718810b92e0d0324456a67c 1 1555847365
```

comments removed, epoch timestamp added to the end.

## Run the phantom executable

```
./phantom -magicbytes="E4D2411C" -port=1929 -protocol_number=70209 -magic_message="ProtonCoin Signed Message:" -bootstrap_ips="51.15.236.48:1929" -bootstrap_url="http://explorer.anodoscrypto.com:3001" -max_connections=10
```

## Coin recipes that have been tested

**ANDS**

* magic_hex = "E4D2411C"
* magic_message = "ProtonCoin Signed Message:"
* port = 1929
* protocol_number = 70209
* bootstrap_url = http://explorer.anodoscrypto.com:3001

**$PAC** 

* magic_hex = "2C61E5C8"
* magic_message = "DarkCoin Signed Message:"
* port = 7112
* protocol_number = 70215
* bootstrap_url = http://explorer.paccoin.net

**SPARKS**

* magic_hex = "D4C3B21A"
* magic_message = "DarkCoin Signed Message:"
* port = 8890
* protocol_number = 70210
* bootstrap_url = http://explorer.sparkspay.io

**XBI**

* magic_hex = "FEF8A489"
* magic_message = "DarkNet Signed Message:"
* port = 7339
* protocol_number = 70997
* boostrap_url = http://explorer.bitcoinincognito.org

## Available Flags

```
  -bootstrap_hash string
    	Hash to bootstrap the pings with ( top - 12 )
  -bootstrap_ips string
    	IP address to bootstrap the network
  -bootstrap_url string
    	Explorer to bootstrap from.
  -daemon_version string
    	The string to use for the sentinel version number (i.e. 1.20.0) (default "0.0.0.0")
  -magic_message string
    	the signing message
  -magic_message_newline
    	add a new line to the magic message (default true)
  -magicbytes string
    	a hex string for the magic bytes
  -max_connections uint
    	the number of peers to maintain (default 10)
  -port uint
    	the default port number
  -protocol_number uint
    	the protocol number to connect and ping with
  -sentinel_version string
    	The string to use for the sentinel version number (i.e. 1.20.0) (default "0.0.0")
```

**Hints on where to get the information**

* magicbytes
  * chainparams.cpp
  * pchMessageStart[3] + pchMessageStart[2] + pchMessageStart[1] + pchMessageStart[0]
  
* magic_message
  * main.cpp or validation.cpp
  * strMessageMagic 
  
* default_port
  * chainparams.cpp
  * nDefaultPort
  
* protocol_version
  * PROTOCOL_VERSION
  * version.h
  
* sentinel_version
  * converted form of DEFAULT_SENTINEL_VERSION from masternode.h or
  * CLIENT_SENTINEL_VERSION from clientversion.h
  
* daemon_version
  * CLIENT_MASTERNODE_VERSION from clientversion.h

## Building (using Docker)

```
docker run --rm -it -v "$PWD":/go/src/phantom -w /go/src/phantom golang:1.12.4 ./build.sh 
```
