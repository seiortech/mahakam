package websocket

import (
	"encoding/binary"
	"errors"
)

type Frame struct {
	FIN           bool
	RSV1          bool
	RSV2          bool
	RSV3          bool
	OpCode        Opcode
	IsMasked      bool
	MaskingKey    [4]byte
	PayloadLength uint64
	Payload       []byte
}

func DecodeFrame(data []byte) (Frame, error) {
	frame := Frame{}
	offset := 0

	if len(data) < offset+1 {
		return frame, errors.New("data is too short")
	}

	frame.FIN = data[offset]&0x80 != 0
	frame.RSV1 = data[offset]&0x40 != 0
	frame.RSV2 = data[offset]&0x20 != 0
	frame.RSV3 = data[offset]&0x10 != 0
	frame.OpCode = Opcode(data[offset] & 0x0F)
	offset++

	if frame.RSV1 || frame.RSV2 || frame.RSV3 {
		return frame, errors.New("RSV must be 0")
	}

	if len(data) < offset+1 {
		return frame, errors.New("data is too short")
	}

	frame.IsMasked = data[offset]&0x80 != 0
	payloadLength := int(data[offset] & 0x7F)
	offset++

	switch payloadLength {
	case 126:
		if len(data) < offset+2 {
			return frame, errors.New("data is too short")
		}
		frame.PayloadLength = uint64(binary.BigEndian.Uint16(data[offset : offset+2]))
		offset += 2
	case 127:
		if len(data) < offset+8 {
			return frame, errors.New("data is too short")
		}
		frame.PayloadLength = uint64(binary.BigEndian.Uint64(data[offset : offset+8]))
		offset += 8
	default:
		frame.PayloadLength = uint64(payloadLength)
	}

	if frame.IsMasked {
		if len(data) < offset+4 {
			return frame, errors.New("data is too short")
		}
		copy(frame.MaskingKey[:], data[offset:offset+4])
		offset += 4
	}

	if len(data) < offset+int(frame.PayloadLength) {
		return frame, errors.New("data is too short")
	}

	payloadStart := offset
	payloadEnd := offset + int(frame.PayloadLength)
	frame.Payload = data[payloadStart:payloadEnd]
	offset = payloadEnd

	if frame.IsMasked {
		for i := range frame.Payload {
			frame.Payload[i] = frame.Payload[i] ^ frame.MaskingKey[i%4]
		}
	}

	return frame, nil
}

func EncodeFrame(payload []byte, opcode Opcode) ([]byte, error) {
	msgFrame := make([]byte, 0, 14+len(payload))
	msgFrame = append(msgFrame, (byte(0x80) | byte(opcode)))

	payloadLen := len(payload)
	var extendedPayloadLen []byte

	if payloadLen <= 125 {
		msgFrame = append(msgFrame, byte(payloadLen))
	} else if payloadLen <= 65535 {
		msgFrame = append(msgFrame, 126)
		extendedPayloadLen = make([]byte, 2)
		binary.BigEndian.PutUint16(extendedPayloadLen, uint16(payloadLen))
	} else {
		msgFrame = append(msgFrame, 127)
		extendedPayloadLen = make([]byte, 8)
		binary.BigEndian.PutUint64(extendedPayloadLen, uint64(payloadLen))
	}

	if extendedPayloadLen != nil {
		msgFrame = append(msgFrame, extendedPayloadLen...)
	}

	msgFrame = append(msgFrame, payload...)

	return msgFrame, nil
}
