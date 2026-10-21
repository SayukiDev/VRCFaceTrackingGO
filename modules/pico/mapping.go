package main

import "VRCFaceTracking/pkg/tracking"

const (
	smoothingFactor = 0.5
	cheekDiffThresh = 0.1
)

// mapper converts PICO blend shapes into Unified Expressions. It keeps the
// smoothing state used for the asymmetric cheek puff heuristic.
type mapper struct {
	lastMouthLeft  float32
	lastMouthRight float32
}

func (m *mapper) smooth(newValue float32, last *float32) float32 {
	*last += (newValue - *last) * smoothingFactor
	return *last
}

func updateEye(data *tracking.TrackingData, p *weights) {
	eye := &data.Eye
	eye.Left.Openness = 1 - p[eyeBlinkL]
	eye.Left.Gaze.X = p[eyeLookInL] - p[eyeLookOutL]
	eye.Left.Gaze.Y = p[eyeLookUpL] - p[eyeLookDownL]

	eye.Right.Openness = 1 - p[eyeBlinkR]
	eye.Right.Gaze.X = p[eyeLookOutR] - p[eyeLookInR]
	eye.Right.Gaze.Y = p[eyeLookUpR] - p[eyeLookDownR]

	s := &data.Shapes
	set := func(u tracking.UnifiedExpression, v float32) { s[u].Weight = v }

	set(tracking.BrowInnerUpLeft, p[browInnerUp])
	set(tracking.BrowInnerUpRight, p[browInnerUp])
	set(tracking.BrowOuterUpLeft, p[browOuterUpL])
	set(tracking.BrowOuterUpRight, p[browOuterUpR])
	set(tracking.BrowLowererLeft, p[browDownL])
	set(tracking.BrowPinchLeft, p[browDownL])
	set(tracking.BrowLowererRight, p[browDownR])
	set(tracking.BrowPinchRight, p[browDownR])

	set(tracking.EyeSquintLeft, p[eyeSquintL])
	set(tracking.EyeSquintRight, p[eyeSquintR])
	set(tracking.EyeWideLeft, p[eyeWideL])
	set(tracking.EyeWideRight, p[eyeWideR])
}

func (m *mapper) updateExpression(data *tracking.TrackingData, p *weights) {
	s := &data.Shapes
	set := func(u tracking.UnifiedExpression, v float32) { s[u].Weight = v }

	// Jaw
	set(tracking.JawOpen, p[jawOpen])
	set(tracking.JawLeft, p[jawLeft])
	set(tracking.JawRight, p[jawRight])
	set(tracking.JawForward, p[jawForward])
	set(tracking.MouthClosed, p[mouthClose])

	// Cheek. PICO only reports a combined puff; use the mouth shift to
	// guess which side is puffed.
	set(tracking.CheekSquintLeft, p[cheekSquintL])
	set(tracking.CheekSquintRight, p[cheekSquintR])

	mouthL := m.smooth(p[mouthLeft], &m.lastMouthLeft)
	mouthR := m.smooth(p[mouthRight], &m.lastMouthRight)
	puff := p[cheekPuff]
	switch {
	case puff > 0.1 && mouthL > mouthR+cheekDiffThresh:
		set(tracking.CheekPuffLeft, puff)
		set(tracking.CheekPuffRight, puff+mouthL)
	case puff > 0.1 && mouthR > mouthL+cheekDiffThresh:
		set(tracking.CheekPuffLeft, puff+mouthR)
		set(tracking.CheekPuffRight, puff)
	default:
		set(tracking.CheekPuffLeft, puff)
		set(tracking.CheekPuffRight, puff)
	}

	// Nose
	set(tracking.NoseSneerLeft, p[noseSneerL])
	set(tracking.NoseSneerRight, p[noseSneerR])

	// Mouth
	set(tracking.MouthUpperUpLeft, p[mouthUpperUpL])
	set(tracking.MouthUpperUpRight, p[mouthUpperUpR])
	set(tracking.MouthLowerDownLeft, p[mouthLowerDownL])
	set(tracking.MouthLowerDownRight, p[mouthLowerDownR])

	set(tracking.MouthFrownLeft, frown(p, p[mouthFrownL]))
	set(tracking.MouthFrownRight, frown(p, p[mouthFrownR]))

	set(tracking.MouthDimpleLeft, p[mouthDimpleL])
	set(tracking.MouthDimpleRight, p[mouthDimpleR])
	set(tracking.MouthUpperLeft, p[mouthLeft])
	set(tracking.MouthLowerLeft, p[mouthLeft])
	set(tracking.MouthUpperRight, p[mouthRight])
	set(tracking.MouthLowerRight, p[mouthRight])
	set(tracking.MouthPressLeft, p[mouthPressL])
	set(tracking.MouthPressRight, p[mouthPressR])
	set(tracking.MouthRaiserLower, p[mouthShrugLower])
	set(tracking.MouthRaiserUpper, p[mouthShrugUpper])

	rollLower := p[mouthRollLower]
	smileL := p[mouthSmileL] - rollLower
	smileR := p[mouthSmileR] - rollLower
	if rollLower < 0.2 {
		set(tracking.MouthCornerPullLeft, smileL)
		set(tracking.MouthCornerSlantLeft, smileL-rollLower)
		set(tracking.MouthCornerPullRight, smileR)
		set(tracking.MouthCornerSlantRight, smileR)
	} else {
		set(tracking.MouthCornerPullLeft, 0)
		set(tracking.MouthCornerSlantLeft, 0)
		set(tracking.MouthCornerPullRight, 0)
		set(tracking.MouthCornerSlantRight, 0)
	}

	set(tracking.MouthStretchLeft, p[mouthStretchL])
	set(tracking.MouthStretchRight, p[mouthStretchR])

	// Lip. A strong pucker without pressing reads as a funnel.
	pucker := p[mouthPucker]
	funnelL := p[mouthFunnel]
	funnelR := p[mouthFunnel]
	if pucker > 0.3 && p[mouthPressL] < 0.2 {
		funnelL = pucker
	}
	if pucker > 0.3 && p[mouthPressR] < 0.2 {
		funnelR = pucker
	}
	set(tracking.LipFunnelUpperLeft, funnelL)
	set(tracking.LipFunnelLowerLeft, funnelL)
	set(tracking.LipFunnelUpperRight, funnelR)
	set(tracking.LipFunnelLowerRight, funnelR)

	set(tracking.LipPuckerUpperLeft, pucker)
	set(tracking.LipPuckerUpperRight, pucker)
	set(tracking.LipPuckerLowerLeft, pucker)
	set(tracking.LipPuckerLowerRight, pucker)
	set(tracking.LipSuckUpperLeft, p[mouthRollUpper])
	set(tracking.LipSuckUpperRight, p[mouthRollUpper])
	set(tracking.LipSuckLowerLeft, rollLower)
	set(tracking.LipSuckLowerRight, rollLower)

	// Tongue
	set(tracking.TongueOut, p[tongueOut])
}

// frown dampens the frown while the jaw is open and boosts it when the
// lower lip is rolled in.
func frown(p *weights, v float32) float32 {
	switch {
	case p[jawOpen] > 0.1:
		return v / 2
	case p[mouthRollLower] > 0.2:
		return v*2.5 + p[mouthRollLower]
	}
	return v
}
