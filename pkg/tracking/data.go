package tracking

import "math"

// Shape holds the raw weight of a single Unified Expression.
type Shape struct {
	Weight float32
}

// SingleEyeData describes one eye.
type SingleEyeData struct {
	Gaze            Vector2
	PupilDiameterMM float32
	Openness        float32
}

// EyeData holds both eyes plus the running pupil dilation range used to
// normalise dilation into 0..1.
type EyeData struct {
	Left, Right SingleEyeData

	MaxDilation, MinDilation    float32
	leftDiameter, rightDiameter float32
}

// Combined merges the left and right eye into a single eye. Pupil dilation is
// averaged and normalised against the observed min/max range, which this call
// updates as a side effect.
func (e *EyeData) Combined() SingleEyeData {
	averageDilation := (e.Left.PupilDiameterMM + e.Right.PupilDiameterMM) / 2
	leftDiff := e.leftDiameter != e.Left.PupilDiameterMM
	rightDiff := e.rightDiameter != e.Right.PupilDiameterMM

	if leftDiff || rightDiff {
		if averageDilation > e.MaxDilation {
			e.MaxDilation = averageDilation
		} else if averageDilation < e.MinDilation {
			e.MinDilation = averageDilation
		}
	}
	if leftDiff {
		e.leftDiameter = e.Left.PupilDiameterMM
	}
	if rightDiff {
		e.rightDiameter = e.Right.PupilDiameterMM
	}

	normalized := (averageDilation - e.MinDilation) / (e.MaxDilation - e.MinDilation)
	if math.IsNaN(float64(normalized)) {
		normalized = 0.5
	}

	return SingleEyeData{
		Gaze:            e.Left.Gaze.Add(e.Right.Gaze).Div(2),
		Openness:        (e.Left.Openness + e.Right.Openness) / 2,
		PupilDiameterMM: normalized,
	}
}

// CopyFrom copies every field of src, including the dilation bookkeeping.
func (e *EyeData) CopyFrom(src *EyeData) {
	src.Combined()
	*e = *src
}

// HeadData holds head pose. Rotation is normalised to [-1, 1] representing
// -90..90 degrees; position is normalised to [-1, 1] over roughly a 1 m cube
// around the user's origin. Coordinate system is left handed, Y up.
type HeadData struct {
	Yaw, Pitch, Roll float32
	PosX, PosY, PosZ float32
}

// TrackingData is the full set of data written by modules and consumed by
// output parameters.
type TrackingData struct {
	Eye    EyeData
	Shapes [ExpressionCount]Shape
	Head   HeadData
}

// NewTrackingData returns data with the same defaults as a freshly started
// tracker: eyes open, dilation range unset.
func NewTrackingData() *TrackingData {
	d := &TrackingData{}
	d.Eye.Left.Openness = 1
	d.Eye.Right.Openness = 1
	d.Eye.MinDilation = 999
	return d
}

// CopyFrom copies src into d.
func (d *TrackingData) CopyFrom(src *TrackingData) {
	d.Eye.CopyFrom(&src.Eye)
	d.Shapes = src.Shapes
	d.Head = src.Head
}
