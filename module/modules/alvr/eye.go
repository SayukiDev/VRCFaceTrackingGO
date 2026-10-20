package alvr

import (
	"math"

	"VRCFaceTracking/pkg/tracking"
)

// normalizedGaze converts an eye orientation quaternion (x, y, z, w) into
// gaze angles. Taken from
// https://github.com/regzo2/VRCFaceTracking-QuestProOpenXR/blob/9b1b15b4b74fc070784a2f0370da1b47b756aac6/VRCFT%20-%20Quest%20OpenXR/QuestOpenXRTrackingModule.cs#L117
func normalizedGaze(q []float32) tracking.Vector2 {
	x, y, z, w := float64(q[0]), float64(q[1]), float64(q[2]), float64(q[3])
	mag := math.Sqrt(x*x + y*y + z*z + w*w)
	x, y, z, w = x/mag, y/mag, z/mag, w/mag

	pitch := math.Asin(2 * (x*z - w*y))
	yaw := math.Atan2(2*(y*z+w*x), w*w-x*x-y*y+z*z)
	return tracking.Vector2{X: float32(pitch), Y: float32(yaw)}
}

// setFixedPupil reports an average pupil size; ALVR quaternion and HTC eye
// data carry none.
func setFixedPupil(eye *tracking.EyeData) {
	eye.Left.PupilDiameterMM = 5
	eye.Right.PupilDiameterMM = 5
	eye.MinDilation = 0
	eye.MaxDilation = 10
}

func setEyesQuat(p []float32, data *tracking.TrackingData) {
	data.Eye.Left.Gaze = normalizedGaze(p[0:4])
	data.Eye.Right.Gaze = normalizedGaze(p[4:8])
	setFixedPupil(&data.Eye)
}

func setCombEyesQuat(p []float32, data *tracking.TrackingData) {
	gaze := normalizedGaze(p[0:4])
	data.Eye.Left.Gaze = gaze
	data.Eye.Right.Gaze = gaze
	setFixedPupil(&data.Eye)
}

// openness derives eye openness from a blink weight. The lid tightener only
// counts while the eye is already closing, hence the multiplication.
func openness(closed, tightener float32) float32 {
	return 1 - clamp01(closed+closed*tightener)
}

func clamp01(v float32) float32 { return min(max(v, 0), 1) }
