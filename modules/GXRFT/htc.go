package main

import "VRCFaceTrackingGo/pkg/tracking"

// Mapping inspired by
// https://github.com/VRCFaceTracking/SRanipalTrackingModule/blob/master/SRanipalExtTrackingModule/SRanipalTrackingInterface.cs

// htcEye indexes XR_HTC_facial_tracking eye weights.
type htcEye int

const (
	htcLeftBlink htcEye = iota
	htcLeftWide
	htcRightBlink
	htcRightWide
	htcLeftSqueeze
	htcRightSqueeze
	htcLeftDown
	htcRightDown
	htcLeftOut
	htcRightIn
	htcLeftIn
	htcRightOut
	htcLeftUp
	htcRightUp

	htcEyesCount
)

// htcLip indexes XR_HTC_facial_tracking lip weights.
type htcLip int

const (
	htcJawRight htcLip = iota
	htcJawLeft
	htcJawForward
	htcJawOpen
	htcMouthApeShape
	htcMouthUpperRight
	htcMouthUpperLeft
	htcMouthLowerRight
	htcMouthLowerLeft
	htcMouthUpperOverturn
	htcMouthLowerOverturn
	htcMouthPout
	htcMouthSmileRight
	htcMouthSmileLeft
	htcMouthSadRight
	htcMouthSadLeft
	htcCheekPuffRight
	htcCheekPuffLeft
	htcCheekSuck
	htcMouthUpperUpRight
	htcMouthUpperUpLeft
	htcMouthLowerDownRight
	htcMouthLowerDownLeft
	htcMouthUpperInside
	htcMouthLowerInside
	htcMouthLowerOverlay
	htcTongueLongStep1
	htcTongueLeft
	htcTongueRight
	htcTongueUp
	htcTongueDown
	htcTongueRoll
	htcTongueLongStep2
	htcTongueUpRightMorph
	htcTongueUpLeftMorph
	htcTongueDownRightMorph
	htcTongueDownLeftMorph

	htcLipCount
)

func setEyesHtc(p []float32, data *tracking.TrackingData) {
	w := func(s htcEye) float32 { return p[s] }
	set := func(u tracking.UnifiedExpression, v float32) { data.Shapes[u].Weight = v }

	eye := &data.Eye
	eye.Right.Openness = openness(w(htcRightBlink), w(htcRightSqueeze))
	eye.Left.Openness = openness(w(htcLeftBlink), w(htcLeftSqueeze))

	eye.Right.Gaze = tracking.Vector2{X: w(htcRightOut) - w(htcRightIn), Y: w(htcRightUp) - w(htcRightDown)}
	eye.Left.Gaze = tracking.Vector2{X: w(htcLeftIn) - w(htcLeftOut), Y: w(htcLeftUp) - w(htcLeftDown)}
	setFixedPupil(eye)

	set(tracking.EyeWideLeft, w(htcLeftWide))
	set(tracking.EyeWideRight, w(htcRightWide))
	set(tracking.EyeSquintLeft, w(htcLeftSqueeze))
	set(tracking.EyeSquintRight, w(htcRightSqueeze))

	// HTC has no brow tracking; emulate it from the eyelids.
	set(tracking.BrowInnerUpLeft, w(htcLeftWide))
	set(tracking.BrowOuterUpLeft, w(htcLeftWide))
	set(tracking.BrowInnerUpRight, w(htcRightWide))
	set(tracking.BrowOuterUpRight, w(htcRightWide))
	set(tracking.BrowPinchLeft, w(htcLeftSqueeze))
	set(tracking.BrowLowererLeft, w(htcLeftSqueeze))
	set(tracking.BrowPinchRight, w(htcRightSqueeze))
	set(tracking.BrowLowererRight, w(htcRightSqueeze))
}

func setLipHtc(p []float32, data *tracking.TrackingData) {
	w := func(s htcLip) float32 { return p[s] }
	set := func(u tracking.UnifiedExpression, v float32) { data.Shapes[u].Weight = v }

	// Jaw
	set(tracking.JawOpen, w(htcJawOpen)+w(htcMouthApeShape))
	set(tracking.JawLeft, w(htcJawLeft))
	set(tracking.JawRight, w(htcJawRight))
	set(tracking.JawForward, w(htcJawForward))
	set(tracking.MouthClosed, w(htcMouthApeShape))

	// Mouth and lip
	set(tracking.MouthUpperUpRight, w(htcMouthUpperRight)-w(htcMouthUpperOverturn))
	set(tracking.MouthUpperDeepenRight, w(htcMouthUpperRight)-w(htcMouthUpperOverturn))
	set(tracking.MouthUpperUpLeft, w(htcMouthUpperLeft)-w(htcMouthUpperOverturn))
	set(tracking.MouthUpperDeepenLeft, w(htcMouthUpperLeft)-w(htcMouthUpperOverturn))

	set(tracking.MouthLowerDownLeft, w(htcMouthLowerLeft)-w(htcMouthLowerOverturn))
	set(tracking.MouthLowerDownRight, w(htcMouthLowerRight)-w(htcMouthLowerOverturn))

	set(tracking.LipPuckerUpperLeft, w(htcMouthPout))
	set(tracking.LipPuckerLowerLeft, w(htcMouthPout))
	set(tracking.LipPuckerUpperRight, w(htcMouthPout))
	set(tracking.LipPuckerLowerRight, w(htcMouthPout))

	set(tracking.LipFunnelUpperLeft, w(htcMouthUpperOverturn))
	set(tracking.LipFunnelUpperRight, w(htcMouthUpperOverturn))
	set(tracking.LipFunnelLowerLeft, w(htcMouthUpperOverturn))
	set(tracking.LipFunnelLowerRight, w(htcMouthUpperOverturn))

	set(tracking.LipSuckUpperLeft, w(htcMouthUpperInside))
	set(tracking.LipSuckUpperRight, w(htcMouthUpperInside))
	set(tracking.LipSuckLowerLeft, w(htcMouthLowerInside))
	set(tracking.LipSuckLowerRight, w(htcMouthLowerInside))

	set(tracking.MouthUpperLeft, w(htcMouthUpperLeft))
	set(tracking.MouthUpperRight, w(htcMouthUpperRight))
	set(tracking.MouthLowerLeft, w(htcMouthLowerLeft))
	set(tracking.MouthLowerRight, w(htcMouthLowerRight))

	set(tracking.MouthCornerPullLeft, w(htcMouthSmileLeft))
	set(tracking.MouthCornerPullRight, w(htcMouthSmileRight))
	set(tracking.MouthCornerSlantLeft, w(htcMouthSmileLeft))
	set(tracking.MouthCornerSlantRight, w(htcMouthSmileRight))
	set(tracking.MouthFrownLeft, w(htcMouthSadLeft))
	set(tracking.MouthFrownRight, w(htcMouthSadRight))

	set(tracking.MouthRaiserUpper, w(htcMouthLowerOverlay)-w(htcMouthUpperInside))
	set(tracking.MouthRaiserLower, w(htcMouthLowerOverlay))

	// Cheek
	set(tracking.CheekPuffLeft, w(htcCheekPuffLeft))
	set(tracking.CheekPuffRight, w(htcCheekPuffRight))
	set(tracking.CheekSuckLeft, w(htcCheekSuck))
	set(tracking.CheekSuckRight, w(htcCheekSuck))

	// Tongue
	set(tracking.TongueOut, (w(htcTongueLongStep1)+w(htcTongueLongStep2))/2)
	set(tracking.TongueUp, w(htcTongueUp))
	set(tracking.TongueDown, w(htcTongueDown))
	set(tracking.TongueLeft, w(htcTongueLeft))
	set(tracking.TongueRight, w(htcTongueRight))
	set(tracking.TongueRoll, w(htcTongueRoll))

	// Emulated Unified Expressions
	set(tracking.CheekSquintLeft, w(htcMouthSmileLeft))
	set(tracking.CheekSquintRight, w(htcMouthSmileRight))
	set(tracking.MouthDimpleLeft, w(htcMouthSmileLeft))
	set(tracking.MouthDimpleRight, w(htcMouthSmileRight))
	// The upstream module drives both sides from the right sad weight.
	set(tracking.MouthStretchLeft, w(htcMouthSadRight))
	set(tracking.MouthStretchRight, w(htcMouthSadRight))
}
