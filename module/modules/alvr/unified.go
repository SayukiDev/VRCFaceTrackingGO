package alvr

import "VRCFaceTracking/pkg/tracking"

// uniExprTail indexes the eye values ALVR appends after the raw Unified
// Expressions weights in a "UniExpr" section.
type uniExprTail int

const (
	uniVersion uniExprTail = iota
	uniLeftOpenness
	uniRightOpenness
	uniLeftPupilDiameter
	uniRightPupilDiameter
	uniMinDilation
	uniMaxDilation
	uniLeftGazePitch
	uniLeftGazeYaw
	uniRightGazePitch
	uniRightGazeYaw

	uniTailCount
)

const uniExprCount = int(tracking.ExpressionCount) + int(uniTailCount)

func setUnifiedExpr(p []float32, data *tracking.TrackingData) {
	for i := range data.Shapes {
		data.Shapes[i].Weight = p[i]
	}

	t := p[tracking.ExpressionCount:]
	eye := &data.Eye
	eye.Left.Openness = t[uniLeftOpenness]
	eye.Right.Openness = t[uniRightOpenness]
	eye.Left.PupilDiameterMM = t[uniLeftPupilDiameter]
	eye.Right.PupilDiameterMM = t[uniRightPupilDiameter]
	eye.MinDilation = t[uniMinDilation]
	eye.MaxDilation = t[uniMaxDilation]
	eye.Left.Gaze = tracking.Vector2{X: t[uniLeftGazePitch], Y: t[uniLeftGazeYaw]}
	eye.Right.Gaze = tracking.Vector2{X: t[uniRightGazePitch], Y: t[uniRightGazeYaw]}
}
