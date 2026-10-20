// Native face tracking OSC input (/tracking/face/v1).
//
// See https://vrc-beta-docs.netlify.app/avatars/face-tracking/osc

package vrchat

import (
	"math"

	"VRCFaceTracking/pkg/tracking"
)

// FaceAddress is the OSC address VRChat listens on for native face tracking.
const FaceAddress = "/tracking/face/v1"

// Payload layout. Keys 0..13 are derived from eye data; the remaining keys
// are the Unified Expressions shapes in their upstream order, which is the
// same order as tracking.UnifiedExpression.
const (
	eyeLookOutRight = iota
	eyeLookInRight
	eyeLookUpRight
	eyeLookDownRight
	eyeLookOutLeft
	eyeLookInLeft
	eyeLookUpLeft
	eyeLookDownLeft
	eyeClosedRight
	eyeClosedLeft
	eyeDilationRight
	eyeDilationLeft
	eyeConstrictRight
	eyeConstrictLeft

	shapeOffset
)

// FacePayloadSize is the exact blob size VRChat expects.
const FacePayloadSize = shapeOffset + int(tracking.ExpressionCount)

// maxGazeAngle is the gaze angle in degrees that maps to a full EyeLook*
// weight.
const maxGazeAngle = 45

// EncodeFace converts d into the 102 byte payload, one byte per tracking key.
func EncodeFace(d *tracking.TrackingData) []byte {
	p := make([]byte, FacePayloadSize)

	lookRight := gaze(d.Eye.Right.Gaze)
	lookLeft := gaze(d.Eye.Left.Gaze)
	// Outward is +X for the right eye and -X for the left eye.
	p[eyeLookOutRight] = toByte(lookRight.X)
	p[eyeLookInRight] = toByte(-lookRight.X)
	p[eyeLookUpRight] = toByte(lookRight.Y)
	p[eyeLookDownRight] = toByte(-lookRight.Y)
	p[eyeLookOutLeft] = toByte(-lookLeft.X)
	p[eyeLookInLeft] = toByte(lookLeft.X)
	p[eyeLookUpLeft] = toByte(lookLeft.Y)
	p[eyeLookDownLeft] = toByte(-lookLeft.Y)

	p[eyeClosedRight] = toByte(1 - d.Eye.Right.Openness)
	p[eyeClosedLeft] = toByte(1 - d.Eye.Left.Openness)

	// Normalised dilation is centred on 0.5; above dilates, below constricts.
	right := dilation(&d.Eye, d.Eye.Right.PupilDiameterMM)
	left := dilation(&d.Eye, d.Eye.Left.PupilDiameterMM)
	p[eyeDilationRight] = toByte((right - .5) * 2)
	p[eyeDilationLeft] = toByte((left - .5) * 2)
	p[eyeConstrictRight] = toByte((.5 - right) * 2)
	p[eyeConstrictLeft] = toByte((.5 - left) * 2)

	for i, s := range d.Shapes {
		p[shapeOffset+i] = toByte(s.Weight)
	}
	return p
}

// gaze converts a tangent-space gaze vector into -1..1 per axis, where 1 is
// maxGazeAngle.
func gaze(v tracking.Vector2) tracking.Vector2 {
	return tracking.Vector2{X: v.ToYaw() / maxGazeAngle, Y: -v.ToPitch() / maxGazeAngle}
}

// dilation normalises a pupil diameter against the observed range. It
// returns 0.5 (neutral) while the range is unknown.
func dilation(e *tracking.EyeData, diameter float32) float32 {
	if e.MaxDilation <= e.MinDilation {
		return .5
	}
	n := (diameter - e.MinDilation) / (e.MaxDilation - e.MinDilation)
	if math.IsNaN(float64(n)) {
		return .5
	}
	return n
}

func toByte(v float32) byte {
	return byte(math.Round(float64(min(max(v, 0), 1)) * 255))
}
