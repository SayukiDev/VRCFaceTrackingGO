package virtualdesktop

import "math"

const (
	expressionCount = 70
	confidenceCount = 2
)

// The structs below mirror the memory layout Virtual Desktop Streamer writes
// into the "VirtualDesktop.BodyState" file mapping (C# StructLayout.Sequential,
// default packing; module v1.5). Field order and types must not change.

type vector3 struct {
	X, Y, Z float32
}

type quaternion struct {
	X, Y, Z, W float32
}

// cartesian converts the eye orientation into the (pitch, yaw) gaze pair that
// tracking.Vector2 expects.
func (q quaternion) cartesian() (x, y float32) {
	mag := float32(math.Sqrt(float64(q.X*q.X + q.Y*q.Y + q.Z*q.Z + q.W*q.W)))
	xm, ym, zm, wm := q.X/mag, q.Y/mag, q.Z/mag, q.W/mag

	pitch := float32(math.Asin(float64(2 * (xm*zm - wm*ym))))
	yaw := float32(math.Atan2(float64(2*(ym*zm+wm*xm)), float64(wm*wm-xm*xm-ym*ym+zm*zm)))
	return pitch, yaw
}

type pose struct {
	Orientation quaternion
	Position    vector3
}

type faceState struct {
	FaceFlags                      uint8
	IsEyeFollowingBlendshapesValid bool
	_                              [2]uint8
	ExpressionWeights              [expressionCount]float32
	ExpressionConfidences          [confidenceCount]float32
	LeftEyeIsValid                 bool
	RightEyeIsValid                bool
	_                              [2]uint8
	LeftEyePose                    pose
	RightEyePose                   pose
	LeftEyeConfidence              float32
	RightEyeConfidence             float32
}

// FaceFlags bits.
const (
	// faceFlagValid is set while face (mouth) tracking data is valid.
	faceFlagValid = 0b0001
	// faceFlagGXR is set when the tongue data follows the Quest 3/GXR layout.
	faceFlagGXR = 0b0010
)

func (f *faceState) faceValid() bool { return f.FaceFlags&faceFlagValid != 0 }
func (f *faceState) gxr() bool       { return f.FaceFlags&faceFlagGXR != 0 }

func (f *faceState) anyValid() bool {
	return f.LeftEyeIsValid || f.RightEyeIsValid || f.IsEyeFollowingBlendshapesValid || f.faceValid()
}
