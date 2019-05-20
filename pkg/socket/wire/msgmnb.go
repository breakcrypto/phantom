// Copyright (c) 2013-2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"bytes"
	"encoding/binary"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"io"
)

type CService struct {
	IpAddress [16]byte
	Port uint16
}

func (service *CService) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
	err := writeElement(w, &service.IpAddress)
	if err != nil {
		return err
	}

	return binary.Write(w, bigEndian, service.Port)
}

func (service *CService) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	err := readElement(r, &service.IpAddress)
	if err != nil {
		return err
	}

	service.Port, err = binarySerializer.Uint16(r, bigEndian)
	if err != nil {
		return err
	}

	return err
}

// MsgPong implements the Message interface and represents a bitcoin pong
// message which is used primarily to confirm that a connection is still valid
// in response to a bitcoin ping message (MsgPing).
//
// This message was not added until protocol versions AFTER BIP0031Version.
type MsgMNB struct {
	// Unique value associated with message that is used to identify
	// specific ping message.
	Vin TxIn
	Addr CService
	PubKeyCollateralAddress []byte
	PubKeyMasternode []byte
	Sig []byte
	SigTime uint64
	ProtocolVersion uint32
	LastPing MsgMNP
	LastDsq uint64
}

// BtcDecode decodes r using the bitcoin protocol encoding into the receiver.
// This is part of the Message interface implementation.
func (msg *MsgMNB) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	// NOTE: <= is not a mistake here.  The BIP0031 was defined as AFTER
	// the version unlike most others.
	//if pver >= BIP0031Version {
	//	str := fmt.Sprintf("pong message invalid for protocol "+
	//		"version %d", pver)
	//	return messageError("MsgPong.BtcDecode", str)
	//}

	//read the tx
	err := readTxIn(r, pver, 0, &msg.Vin)
	if err != nil {
		return err
	}

	//fmt.Println("MNB VIN: ", msg.Vin)

	//read the service
	err = msg.Addr.BtcDecode(r, pver, enc)
	if err != nil {
		return err
	}

	//fmt.Println("MNB ADDR: ", msg.Addr)

	msg.PubKeyCollateralAddress, err = ReadVarBytes(r, pver, MaxMessagePayload,
		"PubKeyCollateralAddress")

	msg.PubKeyMasternode, err = ReadVarBytes(r, pver, MaxMessagePayload,
		"PubKeyMasternode")

	msg.Sig, err = ReadVarBytes(r, pver, MaxMessagePayload,
		"Sig")

	msg.SigTime, err = binarySerializer.Uint64(r, littleEndian)
	if err != nil {
		return err
	}

	msg.ProtocolVersion, err = binarySerializer.Uint32(r, littleEndian)
	if err != nil {
		return err
	}

	//fmt.Println("PROTOCOL: ", msg.ProtocolVersion)

	//decode the ping
	msg.LastPing.BtcDecode(r, pver, enc)

	//fmt.Println("PING TIME: ", time.Unix(int64(msg.LastPing.SigTime), 0).UTC().String())

	//TODO MAKE IT SO MSGMNB/MNP CAN ASK THE PHANTOM FOR COIN SETTINGS TO ASSIST IN DECODE GUESSING
	msg.LastDsq, _ = binarySerializer.Uint64(r, littleEndian) //ignore lastdsq errors
	//if err != nil {
	//	return err
	//}
	//fmt.Println("LAST DSQ: ", msg.LastDsq)

	return err
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
// This is part of the Message interface implementation.
func (msg *MsgMNB) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {

	writeTxIn(w, pver, 0, &msg.Vin)
	msg.Addr.BtcEncode(w, pver, enc)
	WriteVarBytes(w, pver, msg.PubKeyCollateralAddress[:])
	WriteVarBytes(w, pver, msg.PubKeyMasternode[:])
	WriteVarBytes(w, pver, msg.Sig[:])
	writeElement(w, msg.SigTime)
	writeElement(w, msg.ProtocolVersion)
	msg.LastPing.BtcEncode(w, pver, enc)
	writeElement(w, msg.LastDsq)

	return nil
}

// Command returns the protocol command string for the message.  This is part
// of the Message interface implementation.
func (msg *MsgMNB) Command() string {
	return CmdMNB
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.  This is part of the Message interface implementation.
func (msg *MsgMNB) MaxPayloadLength(pver uint32) uint32 {
	//vin + addr + pubKeyCollateralAddress + pubKeyMasternode + sig +
	//sigTime + nProtovolVersion + 	MNP + nLastDsq
	//ping = vin + blockhash + sigTime + vchSig
	return 41+18+65+65+73+8+4+(41+32+8+73)+8
}

// NewMsgPong returns a new bitcoin pong message that conforms to the Message
// interface.  See MsgPong for details.
func NewMsgMNB() *MsgMNB {
	return &MsgMNB{}
}

func (msg *MsgMNB) Serialize(w io.Writer) error {
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

func (msg *MsgMNB) GetHash() chainhash.Hash {
	buf := bytes.NewBuffer(make([]byte, 0, msg.MaxPayloadLength(0)))

	writeElement(buf, msg.SigTime)
	WriteVarBytes(buf, 0, msg.PubKeyCollateralAddress[:])

	byteData := buf.Bytes()
	hash := chainhash.DoubleHashH(byteData)

	return hash
}