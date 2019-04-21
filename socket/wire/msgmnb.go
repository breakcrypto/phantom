// Copyright (c) 2013-2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"io"
)

type CService struct {
	IpAddress [16]byte
	Port uint16
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

	//read the service
	err = msg.Addr.BtcDecode(r, pver, enc)
	if err != nil {
		return err
	}

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

	//decode the ping
	msg.LastPing.BtcDecode(r, pver, enc)

	return err
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
// This is part of the Message interface implementation.
func (msg *MsgMNB) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
	return writeElement(w, msg.Vin)
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
