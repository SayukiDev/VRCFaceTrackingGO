// Package tracking defines the data types shared between the host application
// and tracking modules: Unified Expressions shape indices, eye/head data and
// the container passed to modules on every update.
package tracking

// UnifiedExpression indexes a single shape inside TrackingData.Shapes.
// The order is significant and mirrors the upstream Unified Expressions
// specification; do not reorder.
type UnifiedExpression int

//go:generate go run golang.org/x/tools/cmd/stringer@latest -type=UnifiedExpression -trimprefix=

const (
	// Eye
	EyeSquintRight UnifiedExpression = iota
	EyeSquintLeft
	EyeWideRight
	EyeWideLeft

	// Eyebrow
	BrowPinchRight
	BrowPinchLeft
	BrowLowererRight
	BrowLowererLeft
	BrowInnerUpRight
	BrowInnerUpLeft
	BrowOuterUpRight
	BrowOuterUpLeft

	// Nose
	NasalDilationRight
	NasalDilationLeft
	NasalConstrictRight
	NasalConstrictLeft

	// Cheek
	CheekSquintRight
	CheekSquintLeft
	CheekPuffRight
	CheekPuffLeft
	CheekSuckRight
	CheekSuckLeft

	// Jaw
	JawOpen
	JawRight
	JawLeft
	JawForward
	JawBackward
	JawClench
	JawMandibleRaise
	MouthClosed

	// Lip push/pull
	LipSuckUpperRight
	LipSuckUpperLeft
	LipSuckLowerRight
	LipSuckLowerLeft
	LipSuckCornerRight
	LipSuckCornerLeft
	LipFunnelUpperRight
	LipFunnelUpperLeft
	LipFunnelLowerRight
	LipFunnelLowerLeft
	LipPuckerUpperRight
	LipPuckerUpperLeft
	LipPuckerLowerRight
	LipPuckerLowerLeft

	// Upper lip raiser
	MouthUpperUpRight
	MouthUpperUpLeft
	MouthUpperDeepenRight
	MouthUpperDeepenLeft
	NoseSneerRight
	NoseSneerLeft

	// Lower lip depressor
	MouthLowerDownRight
	MouthLowerDownLeft

	// Mouth direction
	MouthUpperRight
	MouthUpperLeft
	MouthLowerRight
	MouthLowerLeft

	// Smile
	MouthCornerPullRight
	MouthCornerPullLeft
	MouthCornerSlantRight
	MouthCornerSlantLeft

	// Sad
	MouthFrownRight
	MouthFrownLeft
	MouthStretchRight
	MouthStretchLeft

	MouthDimpleRight
	MouthDimpleLeft

	MouthRaiserUpper
	MouthRaiserLower
	MouthPressRight
	MouthPressLeft
	MouthTightenerRight
	MouthTightenerLeft

	// Tongue
	TongueOut
	TongueUp
	TongueDown
	TongueRight
	TongueLeft
	TongueRoll
	TongueBendDown
	TongueCurlUp
	TongueSquish
	TongueFlat
	TongueTwistRight
	TongueTwistLeft

	// Throat / neck
	SoftPalateClose
	ThroatSwallow
	NeckFlexRight
	NeckFlexLeft

	// ExpressionCount is the number of Unified Expression shapes.
	ExpressionCount
)

// SimpleExpression is a derived, coarser shape computed from several Unified
// Expressions. It exists for compatibility with tracking standards that only
// expose e.g. "smile" or "brow down".
type SimpleExpression int

//go:generate go run golang.org/x/tools/cmd/stringer@latest -type=SimpleExpression -trimprefix=

const (
	BrowUpRight SimpleExpression = iota
	BrowUpLeft
	BrowDownRight
	BrowDownLeft
	MouthSmileRight
	MouthSmileLeft
	MouthSadRight
	MouthSadLeft

	SimpleExpressionCount
)

// Simple evaluates a SimpleExpression against the given data.
func (d *TrackingData) Simple(e SimpleExpression) float32 {
	w := func(u UnifiedExpression) float32 { return d.Shapes[u].Weight }
	switch e {
	case BrowUpRight:
		return w(BrowOuterUpRight)*.60 + w(BrowInnerUpRight)*.40
	case BrowUpLeft:
		return w(BrowOuterUpLeft)*.60 + w(BrowInnerUpLeft)*.40
	case BrowDownRight:
		return w(BrowLowererRight)*.75 + w(BrowPinchRight)*.25
	case BrowDownLeft:
		return w(BrowLowererLeft)*.75 + w(BrowPinchLeft)*.25
	case MouthSmileRight:
		return w(MouthCornerPullRight)*.8 + w(MouthCornerSlantRight)*.2
	case MouthSmileLeft:
		return w(MouthCornerPullLeft)*.8 + w(MouthCornerSlantLeft)*.2
	case MouthSadRight:
		return max(w(MouthFrownRight), w(MouthStretchRight))
	case MouthSadLeft:
		return max(w(MouthFrownLeft), w(MouthStretchLeft))
	}
	return 0
}
