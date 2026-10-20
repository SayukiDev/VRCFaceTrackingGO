package virtualdesktop

import (
	"math"

	"VRCFaceTracking/pkg/tracking"
)

// Conversion from OpenXR weights to Unified Expressions. Credit to
// https://github.com/regzo2/VRCFaceTracking-QuestProOpenXR for the mapping.

type weights [expressionCount]float32

func clamp01(v float32) float32 { return max(0, min(1, v)) }

// updateEyeData writes gaze from the eye poses. Openness comes from the
// blend shapes (see updateEyeExpressions), so it is left alone here.
func updateEyeData(eye *tracking.EyeData, left, right quaternion) {
	eye.Right.Gaze.X, eye.Right.Gaze.Y = right.cartesian()
	eye.Left.Gaze.X, eye.Left.Gaze.Y = left.cartesian()

	// Virtual Desktop does not report pupil size; force a fixed diameter and
	// a normalisation range that matches average pupil values.
	eye.Left.PupilDiameterMM = 5
	eye.Right.PupilDiameterMM = 5
	eye.MinDilation = 0
	eye.MaxDilation = 10
}

func updateEyeExpressions(eye *tracking.EyeData, s *[tracking.ExpressionCount]tracking.Shape, e *weights) {
	// The lid tightener only counts while the eye is already closing, hence
	// the multiplication with EyesClosed.
	eye.Left.Openness = 1 - clamp01(e[eyesClosedL]+e[eyesClosedL]*e[lidTightenerL])
	eye.Right.Openness = 1 - clamp01(e[eyesClosedR]+e[eyesClosedR]*e[lidTightenerR])

	set := func(u tracking.UnifiedExpression, v float32) { s[u].Weight = v }

	set(tracking.EyeWideLeft, e[upperLidRaiserL])
	set(tracking.EyeWideRight, e[upperLidRaiserR])

	set(tracking.EyeSquintLeft, e[lidTightenerL])
	set(tracking.EyeSquintRight, e[lidTightenerR])

	set(tracking.BrowInnerUpLeft, e[innerBrowRaiserL])
	set(tracking.BrowInnerUpRight, e[innerBrowRaiserR])
	set(tracking.BrowOuterUpLeft, e[outerBrowRaiserL])
	set(tracking.BrowOuterUpRight, e[outerBrowRaiserR])

	set(tracking.BrowPinchLeft, e[browLowererL])
	set(tracking.BrowLowererLeft, e[browLowererL])
	set(tracking.BrowPinchRight, e[browLowererR])
	set(tracking.BrowLowererRight, e[browLowererR])
}

func updateMouthExpressions(s *[tracking.ExpressionCount]tracking.Shape, e *weights, gxr bool) {
	set := func(u tracking.UnifiedExpression, v float32) { s[u].Weight = v }

	// Jaw
	set(tracking.JawOpen, e[jawDrop])
	set(tracking.JawLeft, e[jawSidewaysLeft])
	set(tracking.JawRight, e[jawSidewaysRight])
	set(tracking.JawForward, e[jawThrust])

	// Mouth
	set(tracking.MouthClosed, e[lipsToward])

	set(tracking.MouthUpperLeft, e[mouthLeft])
	set(tracking.MouthLowerLeft, e[mouthLeft])
	set(tracking.MouthUpperRight, e[mouthRight])
	set(tracking.MouthLowerRight, e[mouthRight])

	// Slant (sharp corner raiser) is baked into corner puller.
	set(tracking.MouthCornerPullLeft, e[lipCornerPullerL])
	set(tracking.MouthCornerSlantLeft, e[lipCornerPullerL])
	set(tracking.MouthCornerPullRight, e[lipCornerPullerR])
	set(tracking.MouthCornerSlantRight, e[lipCornerPullerR])
	set(tracking.MouthFrownLeft, e[lipCornerDepressorL])
	set(tracking.MouthFrownRight, e[lipCornerDepressorR])

	set(tracking.MouthLowerDownLeft, e[lowerLipDepressorL])
	set(tracking.MouthLowerDownRight, e[lowerLipDepressorR])

	// Subtracting the nose wrinkler works around an upper-lip tracking quirk.
	upperUpL := max(0, e[upperLipRaiserL]-e[noseWrinklerL])
	upperUpR := max(0, e[upperLipRaiserR]-e[noseWrinklerR])
	set(tracking.MouthUpperUpLeft, upperUpL)
	set(tracking.MouthUpperDeepenLeft, upperUpL)
	set(tracking.MouthUpperUpRight, upperUpR)
	set(tracking.MouthUpperDeepenRight, upperUpR)

	set(tracking.MouthRaiserUpper, e[chinRaiserT])
	set(tracking.MouthRaiserLower, e[chinRaiserB])

	set(tracking.MouthDimpleLeft, e[dimplerL])
	set(tracking.MouthDimpleRight, e[dimplerR])

	set(tracking.MouthTightenerLeft, e[lipTightenerL])
	set(tracking.MouthTightenerRight, e[lipTightenerR])

	set(tracking.MouthPressLeft, e[lipPressorL])
	set(tracking.MouthPressRight, e[lipPressorR])

	set(tracking.MouthStretchLeft, e[lipStretcherL])
	set(tracking.MouthStretchRight, e[lipStretcherR])

	// Lip
	set(tracking.LipPuckerUpperRight, e[lipPuckerR])
	set(tracking.LipPuckerLowerRight, e[lipPuckerR])
	set(tracking.LipPuckerUpperLeft, e[lipPuckerL])
	set(tracking.LipPuckerLowerLeft, e[lipPuckerL])

	set(tracking.LipFunnelUpperLeft, e[lipFunnelerLT])
	set(tracking.LipFunnelUpperRight, e[lipFunnelerRT])
	set(tracking.LipFunnelLowerLeft, e[lipFunnelerLB])
	set(tracking.LipFunnelLowerRight, e[lipFunnelerRB])

	set(tracking.LipSuckUpperLeft, min(1-float32(math.Pow(float64(e[upperLipRaiserL]), 1.0/6)), e[lipSuckLT]))
	set(tracking.LipSuckUpperRight, min(1-float32(math.Pow(float64(e[upperLipRaiserR]), 1.0/6)), e[lipSuckRT]))
	set(tracking.LipSuckLowerLeft, e[lipSuckLB])
	set(tracking.LipSuckLowerRight, e[lipSuckRB])

	// Cheek
	set(tracking.CheekPuffLeft, e[cheekPuffL])
	set(tracking.CheekPuffRight, e[cheekPuffR])
	set(tracking.CheekSuckLeft, e[cheekSuckL])
	set(tracking.CheekSuckRight, e[cheekSuckR])
	set(tracking.CheekSquintLeft, e[cheekRaiserL])
	set(tracking.CheekSquintRight, e[cheekRaiserR])

	// Nose
	set(tracking.NoseSneerLeft, e[noseWrinklerL])
	set(tracking.NoseSneerRight, e[noseWrinklerR])

	// Tongue
	if gxr {
		set(tracking.TongueOut, e[tongueTipInterdental])
		set(tracking.TongueLeft, e[tongueTipAlveolar])
		set(tracking.TongueRight, e[tongueFrontDorsalPalate])
		set(tracking.TongueUp, e[tongueMidDorsalPalate])
		set(tracking.TongueDown, e[tongueBackDorsalVelar])
	} else {
		set(tracking.TongueOut, e[tongueOut])
		set(tracking.TongueCurlUp, e[tongueTipAlveolar])
	}
}
