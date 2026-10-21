package main

import (
	"encoding/binary"
	"math"

	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/tracking"
)

// Wire format of the ALVR "VRCFaceTracking" sink (little endian): a datagram
// is a sequence of sections, each an 8 byte ASCII prefix followed by the
// float32 parameters of that prefix. The parameter count is implied by the
// prefix.

const prefixSize = 8

// consumer maps the parameters of one section into the tracking data.
type consumer struct {
	count int
	apply func(p []float32, data *tracking.TrackingData)
}

var consumers = map[string]consumer{
	"EyesQuat":       {8, setEyesQuat},
	"CombQuat":       {4, setCombEyesQuat},
	"FaceFb\x00\x00": {int(fbFace1Count), setFace1Fb},
	"Face2Fb\x00":    {int(fbFace2Count), setFace2Fb},
	"FacePico":       {int(picoCount), setFacePico},
	"EyesHtc\x00":    {int(htcEyesCount), setEyesHtc},
	"LipHtc\x00\x00": {int(htcLipCount), setLipHtc},
}

// section is one decoded prefix with its parameters.
type section struct {
	prefix string
	params []float32
}

// parsePacket splits a datagram into sections. Parsing stops at the first
// unknown prefix, because its parameter count and therefore the start of the
// next section are unknown. Truncated parameters are read as zero.
func parsePacket(data []byte) (sections []section, unknown string) {
	for len(data) >= prefixSize {
		prefix := string(data[:prefixSize])
		data = data[prefixSize:]
		c, ok := consumers[prefix]
		if !ok {
			return sections, prefix
		}
		params := make([]float32, c.count)
		for i := range params {
			if len(data) < 4 {
				data = nil
				break
			}
			params[i] = math.Float32frombits(binary.LittleEndian.Uint32(data))
			data = data[4:]
		}
		sections = append(sections, section{prefix: prefix, params: params})
	}
	return sections, ""
}

// apply writes every section into data in packet order.
func apply(sections []section, data *tracking.TrackingData) {
	for _, s := range sections {
		consumers[s.prefix].apply(s.params, data)
	}
}
