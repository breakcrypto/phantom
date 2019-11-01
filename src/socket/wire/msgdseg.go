package wire

import (
"io"
)

// MsgPong implements the Message interface and represents a bitcoin pong
// message which is used primarily to confirm that a connection is still valid
// in response to a bitcoin ping message (MsgPing).
//
// This message was not added until protocol versions AFTER BIP0031Version.
type MsgDSEG struct {
	// Unique value associated with message that is used to identify
	// specific ping message.
	Vin TxIn
}

// BtcDecode decodes r using the bitcoin protocol encoding into the receiver.
// This is part of the Message interface implementation.
func (msg *MsgDSEG) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
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

	return err
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
// This is part of the Message interface implementation.
func (msg *MsgDSEG) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
	//msg.Vin.PreviousOutPoint.Index = MaxTxInSequenceNum
	//msg.Vin.Sequence = MaxTxInSequenceNum
	return writeTxIn(w, pver, 0, &msg.Vin)
}

// Command returns the protocol command string for the message.  This is part
// of the Message interface implementation.
func (msg *MsgDSEG) Command() string {
	return CmdDESG
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.  This is part of the Message interface implementation.
func (msg *MsgDSEG) MaxPayloadLength(pver uint32) uint32 {
	//vin + addr + pubKeyCollateralAddress + pubKeyMasternode + sig +
	//sigTime + nProtovolVersion + 	MNP + nLastDsq
	//ping = vin + blockhash + sigTime + vchSig
	return 41+18+65+65+73+8+4+(41+32+8+73)+8
}

// NewMsgPong returns a new bitcoin pong message that conforms to the Message
// interface.  See MsgPong for details.
func NewMsgDSEG() *MsgDSEG {
	return &MsgDSEG{}
}

