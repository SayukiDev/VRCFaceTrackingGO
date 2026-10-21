package vrchat

import (
	"testing"

	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/tracking"
)

func TestPayloadLayout(t *testing.T) {
	if FacePayloadSize != 102 {
		t.Fatalf("payload size %d, want 102", FacePayloadSize)
	}
	// Spot checks against the documented key order.
	for _, c := range []struct {
		shape tracking.UnifiedExpression
		index int
	}{
		{tracking.EyeSquintRight, 14},
		{tracking.JawOpen, 36},
		{tracking.MouthClosed, 43},
		{tracking.TongueOut, 86},
		{tracking.NeckFlexLeft, 101},
	} {
		if got := shapeOffset + int(c.shape); got != c.index {
			t.Errorf("%v at %d, want %d", c.shape, got, c.index)
		}
	}
}

func TestEncode(t *testing.T) {
	d := tracking.NewTrackingData()
	d.Shapes[tracking.JawOpen].Weight = 1
	d.Shapes[tracking.TongueOut].Weight = 2 // clamped
	d.Eye.Left.Openness = 0
	d.Eye.Right.Gaze = tracking.Vector2{X: 1} // 45° outward
	d.Eye.Left.Gaze = tracking.Vector2{Y: -1} // 45° down

	p := EncodeFace(d)
	want := map[int]byte{
		eyeLookOutRight: 255,
		eyeLookDownLeft: 255,
		eyeClosedLeft:   255,
		36:              255,
		86:              255,
	}
	for i, b := range p {
		if b != want[i] {
			t.Errorf("byte %d = %d, want %d", i, b, want[i])
		}
	}
}

func TestEncodeDilation(t *testing.T) {
	d := tracking.NewTrackingData()
	d.Eye.MinDilation, d.Eye.MaxDilation = 2, 6
	d.Eye.Right.PupilDiameterMM = 6
	d.Eye.Left.PupilDiameterMM = 2

	p := EncodeFace(d)
	if p[eyeDilationRight] != 255 || p[eyeConstrictRight] != 0 {
		t.Errorf("right dilation/constrict = %d/%d", p[eyeDilationRight], p[eyeConstrictRight])
	}
	if p[eyeDilationLeft] != 0 || p[eyeConstrictLeft] != 255 {
		t.Errorf("left dilation/constrict = %d/%d", p[eyeDilationLeft], p[eyeConstrictLeft])
	}
}
