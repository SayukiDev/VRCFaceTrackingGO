package main

import (
	"math"

	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/tracking"
)

// fbShape indexes XR_FB_face_tracking(2) weights as forwarded by ALVR.
type fbShape int

const (
	fbBrowLowererL fbShape = iota
	fbBrowLowererR
	fbCheekPuffL
	fbCheekPuffR
	fbCheekRaiserL
	fbCheekRaiserR
	fbCheekSuckL
	fbCheekSuckR
	fbChinRaiserB
	fbChinRaiserT
	fbDimplerL
	fbDimplerR
	fbEyesClosedL
	fbEyesClosedR
	fbEyesLookDownL
	fbEyesLookDownR
	fbEyesLookLeftL
	fbEyesLookLeftR
	fbEyesLookRightL
	fbEyesLookRightR
	fbEyesLookUpL
	fbEyesLookUpR
	fbInnerBrowRaiserL
	fbInnerBrowRaiserR
	fbJawDrop
	fbJawSidewaysLeft
	fbJawSidewaysRight
	fbJawThrust
	fbLidTightenerL
	fbLidTightenerR
	fbLipCornerDepressorL
	fbLipCornerDepressorR
	fbLipCornerPullerL
	fbLipCornerPullerR
	fbLipFunnelerLB
	fbLipFunnelerLT
	fbLipFunnelerRB
	fbLipFunnelerRT
	fbLipPressorL
	fbLipPressorR
	fbLipPuckerL
	fbLipPuckerR
	fbLipStretcherL
	fbLipStretcherR
	fbLipSuckLB
	fbLipSuckLT
	fbLipSuckRB
	fbLipSuckRT
	fbLipTightenerL
	fbLipTightenerR
	fbLipsToward
	fbLowerLipDepressorL
	fbLowerLipDepressorR
	fbMouthLeft
	fbMouthRight
	fbNoseWrinklerL
	fbNoseWrinklerR
	fbOuterBrowRaiserL
	fbOuterBrowRaiserR
	fbUpperLidRaiserL
	fbUpperLidRaiserR
	fbUpperLipRaiserL
	fbUpperLipRaiserR

	// fbFace1Count is the weight count of XR_FB_face_tracking.
	fbFace1Count

	fbTongueTipInterdental fbShape = iota - 1
	fbTongueTipAlveolar
	fbTongueFrontDorsalPalate
	fbTongueMidDorsalPalate
	fbTongueBackDorsalVelar
	fbTongueOut
	fbTongueRetreat

	// fbFace2Count is the weight count of XR_FB_face_tracking2.
	fbFace2Count
)

func setFace1Fb(p []float32, data *tracking.TrackingData) {
	setFaceFb(p, data)
}

func setFace2Fb(p []float32, data *tracking.TrackingData) {
	setFaceFb(p, data)
	data.Shapes[tracking.TongueOut].Weight = p[fbTongueOut]
}

func setFaceFb(p []float32, data *tracking.TrackingData) {
	w := func(s fbShape) float32 { return p[s] }
	set := func(u tracking.UnifiedExpression, v float32) { data.Shapes[u].Weight = v }

	data.Eye.Right.Openness = openness(w(fbEyesClosedR), w(fbLidTightenerR))
	data.Eye.Left.Openness = openness(w(fbEyesClosedL), w(fbLidTightenerL))

	// Eyelids
	set(tracking.EyeSquintRight, w(fbLidTightenerR))
	set(tracking.EyeSquintLeft, w(fbLidTightenerL))
	set(tracking.EyeWideRight, w(fbUpperLidRaiserR))
	set(tracking.EyeWideLeft, w(fbUpperLidRaiserL))

	// Eyebrows
	set(tracking.BrowPinchRight, w(fbBrowLowererR))
	set(tracking.BrowPinchLeft, w(fbBrowLowererL))
	set(tracking.BrowLowererRight, w(fbBrowLowererR))
	set(tracking.BrowLowererLeft, w(fbBrowLowererL))
	set(tracking.BrowInnerUpRight, w(fbInnerBrowRaiserR))
	set(tracking.BrowInnerUpLeft, w(fbInnerBrowRaiserL))
	set(tracking.BrowOuterUpRight, w(fbOuterBrowRaiserR))
	set(tracking.BrowOuterUpLeft, w(fbOuterBrowRaiserL))

	// Cheeks
	set(tracking.CheekSquintRight, w(fbCheekRaiserR))
	set(tracking.CheekSquintLeft, w(fbCheekRaiserL))
	set(tracking.CheekPuffRight, w(fbCheekPuffR))
	set(tracking.CheekPuffLeft, w(fbCheekPuffL))
	set(tracking.CheekSuckRight, w(fbCheekSuckR))
	set(tracking.CheekSuckLeft, w(fbCheekSuckL))

	// Jaw
	set(tracking.JawOpen, w(fbJawDrop))
	set(tracking.JawRight, w(fbJawSidewaysRight))
	set(tracking.JawLeft, w(fbJawSidewaysLeft))
	set(tracking.JawForward, w(fbJawThrust))
	set(tracking.MouthClosed, w(fbLipsToward))

	// Lip push/pull. A raised upper lip cannot be sucked in at the same time.
	suckLimit := func(raiser float32) float32 {
		return 1 - float32(math.Pow(float64(raiser), 1.0/6.0))
	}
	set(tracking.LipSuckUpperRight, min(suckLimit(w(fbUpperLipRaiserR)), w(fbLipSuckRT)))
	set(tracking.LipSuckUpperLeft, min(suckLimit(w(fbUpperLipRaiserL)), w(fbLipSuckLT)))
	set(tracking.LipSuckLowerRight, w(fbLipSuckRB))
	set(tracking.LipSuckLowerLeft, w(fbLipSuckLB))
	set(tracking.LipFunnelUpperRight, w(fbLipFunnelerRT))
	set(tracking.LipFunnelUpperLeft, w(fbLipFunnelerLT))
	set(tracking.LipFunnelLowerRight, w(fbLipFunnelerRB))
	set(tracking.LipFunnelLowerLeft, w(fbLipFunnelerLB))
	set(tracking.LipPuckerUpperRight, w(fbLipPuckerR))
	set(tracking.LipPuckerUpperLeft, w(fbLipPuckerL))
	set(tracking.LipPuckerLowerRight, w(fbLipPuckerR))
	set(tracking.LipPuckerLowerLeft, w(fbLipPuckerL))

	// Upper lip raiser
	upperUpR := max(0, w(fbUpperLipRaiserR)-w(fbNoseWrinklerR))
	upperUpL := max(0, w(fbUpperLipRaiserL)-w(fbNoseWrinklerL))
	set(tracking.MouthUpperUpRight, upperUpR)
	set(tracking.MouthUpperUpLeft, upperUpL)
	set(tracking.MouthUpperDeepenRight, upperUpR)
	set(tracking.MouthUpperDeepenLeft, upperUpL)
	set(tracking.NoseSneerRight, w(fbNoseWrinklerR))
	set(tracking.NoseSneerLeft, w(fbNoseWrinklerL))

	// Lower lip depressor
	set(tracking.MouthLowerDownRight, w(fbLowerLipDepressorR))
	set(tracking.MouthLowerDownLeft, w(fbLowerLipDepressorL))

	// Mouth direction
	set(tracking.MouthUpperRight, w(fbMouthRight))
	set(tracking.MouthUpperLeft, w(fbMouthLeft))
	set(tracking.MouthLowerRight, w(fbMouthRight))
	set(tracking.MouthLowerLeft, w(fbMouthLeft))

	// Smile
	set(tracking.MouthCornerPullRight, w(fbLipCornerPullerR))
	set(tracking.MouthCornerPullLeft, w(fbLipCornerPullerL))
	set(tracking.MouthCornerSlantRight, w(fbLipCornerPullerR))
	set(tracking.MouthCornerSlantLeft, w(fbLipCornerPullerL))

	// Frown
	set(tracking.MouthFrownRight, w(fbLipCornerDepressorR))
	set(tracking.MouthFrownLeft, w(fbLipCornerDepressorL))
	set(tracking.MouthStretchRight, w(fbLipStretcherR))
	set(tracking.MouthStretchLeft, w(fbLipStretcherL))

	set(tracking.MouthDimpleLeft, w(fbDimplerL))
	set(tracking.MouthDimpleRight, w(fbDimplerR))

	set(tracking.MouthRaiserUpper, w(fbChinRaiserT))
	set(tracking.MouthRaiserLower, w(fbChinRaiserB))
	set(tracking.MouthPressRight, w(fbLipPressorR))
	set(tracking.MouthPressLeft, w(fbLipPressorL))
	set(tracking.MouthTightenerRight, w(fbLipTightenerR))
	set(tracking.MouthTightenerLeft, w(fbLipTightenerL))
}
