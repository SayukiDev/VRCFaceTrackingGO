package pico

import (
	"bytes"
	"encoding/binary"
	"errors"
)

// Wire format of the PICO Connect / Streaming Assistant face tracking
// stream (little endian, packed). The current protocol prefixes each body
// with a header; the legacy protocol sends the bare body.

type packetHeader struct {
	StartCode1   uint8
	StartCode2   uint8
	TrackingType uint8
	SubType      uint8
	MultiPack    uint8
	CurrentIndex uint8
	Version      uint16
	TimeStamp    uint64
}

type packetBody struct {
	TimeStamp        int64
	BlendShapeWeight [blendShapeCount]float32
	VideoInputValid  [10]float32
	LaughingProb     float32
	EmotionProb      [10]float32
	Reserved         [128]float32
}

const (
	headerSize = 16
	bodySize   = 8 + int(blendShapeCount)*4 + 10*4 + 4 + 10*4 + 128*4

	// trackingTypeFace marks a face tracking body in the current protocol.
	trackingTypeFace = 2
)

type weights [blendShapeCount]float32

var errNotFaceData = errors.New("pico: packet does not contain face data")

// parsePacket extracts the blend shape weights from a datagram.
func parsePacket(data []byte, legacy bool) (*weights, error) {
	if legacy {
		if len(data) < bodySize {
			return nil, errNotFaceData
		}
		return parseBody(data)
	}
	if len(data) < headerSize+bodySize {
		return nil, errNotFaceData
	}
	var header packetHeader
	if err := binary.Read(bytes.NewReader(data[:headerSize]), binary.LittleEndian, &header); err != nil {
		return nil, err
	}
	if header.TrackingType != trackingTypeFace {
		return nil, errNotFaceData
	}
	return parseBody(data[headerSize:])
}

func parseBody(data []byte) (*weights, error) {
	var body packetBody
	if err := binary.Read(bytes.NewReader(data[:bodySize]), binary.LittleEndian, &body); err != nil {
		return nil, err
	}
	w := weights(body.BlendShapeWeight)
	return &w, nil
}
