package alvr

import "VRCFaceTracking/pkg/tracking"

// picoShape indexes the 72 ARKit-style weights of the PICO face tracking
// extension as forwarded by ALVR.
type picoShape int

const (
	picoEyeLookDownL picoShape = iota
	picoNoseSneerL
	picoEyeLookInL
	picoBrowInnerUp
	picoBrowDownR
	picoMouthClose
	picoMouthLowerDownR
	picoJawOpen
	picoMouthUpperUpR
	picoMouthShrugUpper
	picoMouthFunnel
	picoEyeLookInR
	picoEyeLookDownR
	picoNoseSneerR
	picoMouthRollUpper
	picoJawRight
	picoBrowDownL
	picoMouthShrugLower
	picoMouthRollLower
	picoMouthSmileL
	picoMouthPressL
	picoMouthSmileR
	picoMouthPressR
	picoMouthDimpleR
	picoMouthLeft
	picoJawForward
	picoEyeSquintL
	picoMouthFrownL
	picoEyeBlinkL
	picoCheekSquintL
	picoBrowOuterUpL
	picoEyeLookUpL
	picoJawLeft
	picoMouthStretchL
	picoMouthPucker
	picoEyeLookUpR
	picoBrowOuterUpR
	picoCheekSquintR
	picoEyeBlinkR
	picoMouthUpperUpL
	picoMouthFrownR
	picoEyeSquintR
	picoMouthStretchR
	picoCheekPuff
	picoEyeLookOutL
	picoEyeLookOutR
	picoEyeWideR
	picoEyeWideL
	picoMouthRight
	picoMouthDimpleL
	picoMouthLowerDownL
	picoTongueOut
	picoVisemePP
	picoVisemeCH
	picoVisemeOLower
	picoVisemeO
	picoVisemeI
	picoVisemeULower
	picoVisemeRR
	picoVisemeXX
	picoVisemeAA
	picoVisemeILower
	picoVisemeFF
	picoVisemeU
	picoVisemeTH
	picoVisemeKK
	picoVisemeSS
	picoVisemeELower
	picoVisemeDD
	picoVisemeE
	picoVisemeNN
	picoVisemeSil

	picoCount
)

func setFacePico(p []float32, data *tracking.TrackingData) {
	w := func(s picoShape) float32 { return p[s] }
	set := func(u tracking.UnifiedExpression, v float32) { data.Shapes[u].Weight = v }

	data.Eye.Right.Openness = openness(w(picoEyeBlinkR), w(picoEyeSquintR))
	data.Eye.Left.Openness = openness(w(picoEyeBlinkL), w(picoEyeSquintL))

	// Eye
	set(tracking.EyeSquintRight, w(picoEyeSquintR))
	set(tracking.EyeSquintLeft, w(picoEyeSquintL))
	set(tracking.EyeWideRight, w(picoEyeWideR))
	set(tracking.EyeWideLeft, w(picoEyeWideL))

	// Eyebrow
	set(tracking.BrowPinchRight, w(picoBrowDownR))
	set(tracking.BrowPinchLeft, w(picoBrowDownL))
	set(tracking.BrowLowererRight, w(picoBrowDownR))
	set(tracking.BrowLowererLeft, w(picoBrowDownL))
	set(tracking.BrowInnerUpRight, w(picoBrowInnerUp))
	set(tracking.BrowInnerUpLeft, w(picoBrowInnerUp))
	set(tracking.BrowOuterUpRight, w(picoBrowOuterUpR))
	set(tracking.BrowOuterUpLeft, w(picoBrowOuterUpL))

	// Cheek
	set(tracking.CheekSquintRight, w(picoCheekSquintR))
	set(tracking.CheekSquintLeft, w(picoCheekSquintL))
	set(tracking.CheekPuffRight, w(picoCheekPuff))
	set(tracking.CheekPuffLeft, w(picoCheekPuff))

	// Jaw
	set(tracking.JawOpen, w(picoJawOpen))
	set(tracking.JawRight, w(picoJawRight))
	set(tracking.JawLeft, w(picoJawLeft))
	set(tracking.JawForward, w(picoJawForward))
	set(tracking.MouthClosed, w(picoMouthClose))

	// Lip
	set(tracking.LipSuckUpperRight, w(picoMouthRollUpper))
	set(tracking.LipSuckUpperLeft, w(picoMouthRollUpper))
	set(tracking.LipSuckLowerRight, w(picoMouthRollLower))
	set(tracking.LipSuckLowerLeft, w(picoMouthRollLower))

	set(tracking.LipFunnelUpperRight, w(picoMouthFunnel))
	set(tracking.LipFunnelUpperLeft, w(picoMouthFunnel))
	set(tracking.LipFunnelLowerRight, w(picoMouthFunnel))
	set(tracking.LipFunnelLowerLeft, w(picoMouthFunnel))

	set(tracking.LipPuckerUpperRight, w(picoMouthPucker))
	set(tracking.LipPuckerUpperLeft, w(picoMouthPucker))
	set(tracking.LipPuckerLowerRight, w(picoMouthPucker))
	set(tracking.LipPuckerLowerLeft, w(picoMouthPucker))

	upperUpR := max(0, w(picoMouthUpperUpR)-w(picoNoseSneerR))
	upperUpL := max(0, w(picoMouthUpperUpL)-w(picoNoseSneerL))
	set(tracking.MouthUpperUpRight, upperUpR)
	set(tracking.MouthUpperUpLeft, upperUpL)
	set(tracking.MouthUpperDeepenRight, upperUpR)
	set(tracking.MouthUpperDeepenLeft, upperUpL)

	set(tracking.NoseSneerRight, w(picoNoseSneerR))
	set(tracking.NoseSneerLeft, w(picoNoseSneerL))

	set(tracking.MouthLowerDownRight, w(picoMouthLowerDownR))
	set(tracking.MouthLowerDownLeft, w(picoMouthLowerDownL))

	set(tracking.MouthUpperRight, w(picoMouthRight))
	set(tracking.MouthUpperLeft, w(picoMouthLeft))
	set(tracking.MouthLowerRight, w(picoMouthRight))
	set(tracking.MouthLowerLeft, w(picoMouthLeft))

	set(tracking.MouthCornerPullRight, w(picoMouthSmileR))
	set(tracking.MouthCornerPullLeft, w(picoMouthSmileL))
	set(tracking.MouthCornerSlantRight, w(picoMouthSmileR))
	set(tracking.MouthCornerSlantLeft, w(picoMouthSmileL))

	set(tracking.MouthFrownRight, w(picoMouthFrownR))
	set(tracking.MouthFrownLeft, w(picoMouthFrownL))
	set(tracking.MouthStretchRight, w(picoMouthStretchR))
	set(tracking.MouthStretchLeft, w(picoMouthStretchL))

	set(tracking.MouthDimpleRight, w(picoMouthDimpleR))
	set(tracking.MouthDimpleLeft, w(picoMouthDimpleL))

	set(tracking.MouthRaiserUpper, w(picoMouthShrugUpper))
	set(tracking.MouthRaiserLower, w(picoMouthShrugLower))
	set(tracking.MouthPressRight, w(picoMouthPressR))
	set(tracking.MouthPressLeft, w(picoMouthPressL))

	// Tongue
	set(tracking.TongueOut, w(picoTongueOut))
}
