// Copyright (c) 2013-2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"bytes"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
)

// MsgPong implements the Message interface and represents a bitcoin pong
// message which is used primarily to confirm that a connection is still valid
// in response to a bitcoin ping message (MsgPing).
//
// This message was not added until protocol versions AFTER BIP0031Version.
type MsgMNP struct {
	// Unique value associated with message that is used to identify
	// specific ping message.
	Vin TxIn
	UseOutpointForm bool
	BlockHash chainhash.Hash
	SigTime uint64
	VchSig []byte
	SentinelIsCurrent bool
	SentinelVersion uint32
	DaemonVersion uint32
}

// BtcDecode decodes r using the bitcoin protocol encoding into the receiver.
// This is part of the Message interface implementation.
func (msg *MsgMNP) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {

	byteHolder, err := ioutil.ReadAll(r)
	if err != nil {
		log.Error(err)
		return err
	}

	//reset the reader
	r = bytes.NewReader(byteHolder)

	if !msg.UseOutpointForm {
		//read the tx
		err := readTxIn(r, pver, 0, &msg.Vin)
		if err != nil || msg.Vin.Sequence != 4294967295 {
			//return err
			//fallback to Outpoint form before reusing
			msg.UseOutpointForm = true
			//reset the reader
			r = bytes.NewReader(byteHolder)
		}
	}

	if msg.UseOutpointForm {
		err := readOutPoint(r, pver, 0, &msg.Vin.PreviousOutPoint)
		if err != nil {
			return err
		}
	}

	//fmt.Println("VIN: ", msg.Vin)

	//read the hash
	_, err = io.ReadFull(r, msg.BlockHash[:])
	if err != nil {
		return err
	}

	//fmt.Println("BLOCKHASH: ", msg.BlockHash)

	msg.SigTime, err = binarySerializer.Uint64(r, littleEndian)
	if err != nil {
		return err
	}

	//fmt.Println("PING TIME: ", time.Unix(int64(msg.SigTime), 0).UTC().String())

	msg.VchSig, err = ReadVarBytes(r, pver, MaxMessagePayload,
		"vchSig")
	if err != nil { //avoid infinite loops with enc == 1
		if enc != 1 {
			msg.UseOutpointForm = true
			r = bytes.NewReader(byteHolder)
			return msg.BtcDecode(r, pver, 1)
		} else {
			return err
		}
	}

	//fmt.Println("SIG: ", msg.VchSig)

	val, err := binarySerializer.Uint8(r)
	if err != nil {
		return nil //ignore decode errors and just move along
	}

	//fmt.Println("SENTINEL ENABLED: ", val)
	if val != 0 {
		msg.SentinelIsCurrent = true
	}

	msg.SentinelVersion, err = binarySerializer.Uint32(r, littleEndian)
	if err != nil {
		return nil //ignore decode errors and just move along
	}

	msg.DaemonVersion, err = binarySerializer.Uint32(r, littleEndian)
	if err != nil {
		return nil //ignore decode errors and just move along
	}

	return err
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
// This is part of the Message interface implementation.
func (msg *MsgMNP) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {

	if !msg.UseOutpointForm {
		writeTxIn(w, pver, 0, &msg.Vin)
	} else {
		writeOutPoint(w, pver, 0, &msg.Vin.PreviousOutPoint)
	}

	_, err := w.Write(msg.BlockHash[:])
	if err != nil {
		return err
	}

	writeElement(w, msg.SigTime)

	WriteVarBytes(w, pver, msg.VchSig[:])

	if msg.SentinelVersion > 0 { //defaults to false
		var enabled uint8 = 1
		writeElement(w, enabled)
		writeElement(w, msg.SentinelVersion)
	}

	if msg.SentinelVersion > 0 {
		writeElement(w, msg.DaemonVersion)
	}

	return err
}

func (msg *MsgMNP) Serialize(w io.Writer) error {
	// At the current time, there is no difference between the wire encoding
	// at protocol version 0 and the stable long-term storage format.  As
	// a result, make use of BtcEncode.
	//
	// Passing a encoding type of WitnessEncoding to BtcEncode for MsgTx
	// indicates that the transaction's witnesses (if any) should be
	// serialized according to the new serialization structure defined in
	// BIP0144.
	return msg.BtcEncode(w, 0, 0)
}

// Command returns the protocol command string for the message.  This is part
// of the Message interface implementation.
func (msg *MsgMNP) Command() string {
	return CmdMNP
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.  This is part of the Message interface implementation.
func (msg *MsgMNP) MaxPayloadLength(pver uint32) uint32 {
	//vin + blockhash + sigTime + vchSig
	return 41+32+8+74+1+4
}

// NewMsgPong returns a new bitcoin pong message that conforms to the Message
// interface.  See MsgPong for details.
func NewMsgMNP() *MsgMNP {
	return &MsgMNP{}
}

func (msg *MsgMNP) GetHash() chainhash.Hash {
	buf := bytes.NewBuffer(make([]byte, 0, msg.MaxPayloadLength(0)))

	writeElement(buf, msg.SigTime)
	writeElement(buf, msg.Vin.PreviousOutPoint.String())

	byteData := buf.Bytes()
	hash := chainhash.DoubleHashH(byteData)

	return hash
}