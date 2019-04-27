// Copyright (c) 2013-2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"io"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
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
	BlockHash chainhash.Hash
	SigTime uint64
	VchSig []byte
	SentinelEnabled bool
	SentinelIsCurrent bool
	SentinelVersion uint32
	DaemonEnabled bool
	DaemonVersion uint32
}

// BtcDecode decodes r using the bitcoin protocol encoding into the receiver.
// This is part of the Message interface implementation.
func (msg *MsgMNP) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
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

	//read the hash
	_, err = io.ReadFull(r, msg.BlockHash[:])
	if err != nil {
		return err
	}

	msg.SigTime, err = binarySerializer.Uint64(r, littleEndian)
	if err != nil {
		return err
	}

	msg.VchSig, err = ReadVarBytes(r, pver, MaxMessagePayload,
		"vchSig")

	if msg.SentinelEnabled { //defaults to false
		val, _ := binarySerializer.Uint8(r)
		if val != 0 {
			msg.SentinelIsCurrent = true
		}
		msg.SentinelVersion, err = binarySerializer.Uint32(r, littleEndian)
	}

	if msg.DaemonEnabled { //defaults to false
		msg.DaemonVersion, err = binarySerializer.Uint32(r, littleEndian)
	}

	return err
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
// This is part of the Message interface implementation.
func (msg *MsgMNP) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {

	writeTxIn(w, pver, 0, &msg.Vin)

	_, err := w.Write(msg.BlockHash[:])
	if err != nil {
		return err
	}

	writeElement(w, msg.SigTime)

	WriteVarBytes(w, pver, msg.VchSig[:])

	if msg.SentinelEnabled { //defaults to false
		var enabled uint8 = 1
		writeElement(w, enabled)
		writeElement(w, msg.SentinelVersion)
	}

	if msg.DaemonEnabled {
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
