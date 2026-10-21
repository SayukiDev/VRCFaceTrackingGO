package plugin

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"

	"VRCFaceTrackingGo/internal/module/adapter"
	"VRCFaceTrackingGo/pkg/tracking"
)

// The host sends one request at a time and waits for its response; all
// numbers are little endian.
//
//	request:  op (1 byte) [support (1 byte), only for opInitialize]
//	response: statusOK   payload
//	          statusErr  length (uint32) message (length bytes)
//
// Payloads: opInitialize returns the granted support (1 byte), opUpdate the
// tracking data (dataSize bytes), opTeardown nothing.
const (
	opInitialize byte = 'I'
	opUpdate     byte = 'U'
	opTeardown   byte = 'T'

	statusOK  byte = 0
	statusErr byte = 1
)

// Support is one byte: bit 0 eye, bit 1 expression.
const (
	supportEye        byte = 1 << 0
	supportExpression byte = 1 << 1
)

func encodeSupport(s adapter.Support) byte {
	var b byte
	if s.Eye {
		b |= supportEye
	}
	if s.Expression {
		b |= supportExpression
	}
	return b
}

func decodeSupport(b byte) adapter.Support {
	return adapter.Support{Eye: b&supportEye != 0, Expression: b&supportExpression != 0}
}

// Tracking data is a fixed list of float32: per eye (left, right) gaze X,
// gaze Y, pupil diameter and openness, then max and min dilation, every
// shape weight in tracking.UnifiedExpression order, and the head pose.
const (
	eyeFloats   = 2*4 + 2
	headFloats  = 6
	dataFloats  = eyeFloats + int(tracking.ExpressionCount) + headFloats
	dataSize    = dataFloats * 4
	maxErrorLen = 64 * 1024
)

// fields lists every transmitted float of d in wire order.
func fields(d *tracking.TrackingData) []*float32 {
	f := make([]*float32, 0, dataFloats)
	for _, e := range []*tracking.SingleEyeData{&d.Eye.Left, &d.Eye.Right} {
		f = append(f, &e.Gaze.X, &e.Gaze.Y, &e.PupilDiameterMM, &e.Openness)
	}
	f = append(f, &d.Eye.MaxDilation, &d.Eye.MinDilation)
	for i := range d.Shapes {
		f = append(f, &d.Shapes[i].Weight)
	}
	h := &d.Head
	return append(f, &h.Yaw, &h.Pitch, &h.Roll, &h.PosX, &h.PosY, &h.PosZ)
}

func appendData(buf []byte, d *tracking.TrackingData) []byte {
	for _, v := range fields(d) {
		buf = binary.LittleEndian.AppendUint32(buf, math.Float32bits(*v))
	}
	return buf
}

func readData(r io.Reader) (tracking.TrackingData, error) {
	var buf [dataSize]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return tracking.TrackingData{}, err
	}
	var d tracking.TrackingData
	for i, v := range fields(&d) {
		*v = math.Float32frombits(binary.LittleEndian.Uint32(buf[i*4:]))
	}
	return d, nil
}

// appendResult appends statusOK and payload, or statusErr and err.
func appendResult(buf []byte, err error, payload func([]byte) []byte) []byte {
	if err != nil {
		msg := err.Error()
		if len(msg) > maxErrorLen {
			msg = msg[:maxErrorLen]
		}
		buf = append(buf, statusErr)
		buf = binary.LittleEndian.AppendUint32(buf, uint32(len(msg)))
		return append(buf, msg...)
	}
	buf = append(buf, statusOK)
	if payload != nil {
		buf = payload(buf)
	}
	return buf
}

// remoteError is an error reported by the module.
type remoteError string

func (e remoteError) Error() string { return string(e) }

// readStatus reads a response status. A statusErr response is returned as
// a remoteError.
func readStatus(r *bufio.Reader) error {
	status, err := r.ReadByte()
	if err != nil {
		return err
	}
	switch status {
	case statusOK:
		return nil
	case statusErr:
		var n [4]byte
		if _, err := io.ReadFull(r, n[:]); err != nil {
			return err
		}
		size := binary.LittleEndian.Uint32(n[:])
		if size > maxErrorLen {
			return errors.New("module error message too long")
		}
		msg := make([]byte, size)
		if _, err := io.ReadFull(r, msg); err != nil {
			return err
		}
		return remoteError(msg)
	}
	return fmt.Errorf("unknown response status %d", status)
}
