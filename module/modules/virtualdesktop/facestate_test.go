package virtualdesktop

import (
	"math"
	"testing"
	"time"
	"unsafe"

	"go.uber.org/zap"

	"VRCFaceTracking/module/adapter"
	"VRCFaceTracking/pkg/tracking"
)

// The C# FaceState struct (v1.5, StructLayout.Sequential) is 360 bytes:
// 4 (flags+bool+pad) + 70*4 + 2*4 + 4 (bools+pad) + 2*28 (poses) + 2*4.
func TestFaceStateLayout(t *testing.T) {
	var s faceState
	checks := map[string][2]uintptr{
		"size":                           {unsafe.Sizeof(s), 360},
		"IsEyeFollowingBlendshapesValid": {unsafe.Offsetof(s.IsEyeFollowingBlendshapesValid), 1},
		"ExpressionWeights":              {unsafe.Offsetof(s.ExpressionWeights), 4},
		"ExpressionConfidences":          {unsafe.Offsetof(s.ExpressionConfidences), 284},
		"LeftEyeIsValid":                 {unsafe.Offsetof(s.LeftEyeIsValid), 292},
		"RightEyeIsValid":                {unsafe.Offsetof(s.RightEyeIsValid), 293},
		"LeftEyePose":                    {unsafe.Offsetof(s.LeftEyePose), 296},
		"RightEyePose":                   {unsafe.Offsetof(s.RightEyePose), 324},
		"LeftEyeConfidence":              {unsafe.Offsetof(s.LeftEyeConfidence), 352},
	}
	for name, c := range checks {
		if c[0] != c[1] {
			t.Errorf("%s = %d, want %d", name, c[0], c[1])
		}
	}
}

func TestFaceFlags(t *testing.T) {
	s := faceState{FaceFlags: faceFlagValid | faceFlagGXR}
	if !s.faceValid() || !s.gxr() || !s.anyValid() {
		t.Fatalf("flags not decoded: %+v", s)
	}
	var zero faceState
	if zero.anyValid() {
		t.Fatal("zero state must not be valid")
	}
}

func TestQuaternionCartesianIdentity(t *testing.T) {
	x, y := quaternion{0, 0, 0, 1}.cartesian()
	if x != 0 || y != 0 {
		t.Fatalf("identity gaze = (%v, %v), want (0, 0)", x, y)
	}
}

func TestUpdateEyeExpressionsOpenness(t *testing.T) {
	var e weights
	e[eyesClosedL] = 0.5
	e[lidTightenerL] = 1 // 0.5 + 0.5*1 = 1 -> fully closed
	e[eyesClosedR] = 0.25
	e[lidTightenerR] = 0 // 0.25 -> 75% open

	data := tracking.NewTrackingData()
	updateEyeExpressions(&data.Eye, &data.Shapes, &e)
	if data.Eye.Left.Openness != 0 {
		t.Errorf("left openness = %v, want 0", data.Eye.Left.Openness)
	}
	if data.Eye.Right.Openness != 0.75 {
		t.Errorf("right openness = %v, want 0.75", data.Eye.Right.Openness)
	}
}

func TestUpdateMouthExpressions(t *testing.T) {
	var e weights
	e[jawDrop] = 0.5
	e[upperLipRaiserL] = 0.8
	e[noseWrinklerL] = 0.3
	e[tongueOut] = 1

	var shapes [tracking.ExpressionCount]tracking.Shape
	updateMouthExpressions(&shapes, &e, false)

	if got := shapes[tracking.JawOpen].Weight; got != 0.5 {
		t.Errorf("JawOpen = %v, want 0.5", got)
	}
	if got := shapes[tracking.MouthUpperUpLeft].Weight; math.Abs(float64(got-0.5)) > 1e-6 {
		t.Errorf("MouthUpperUpLeft = %v, want 0.5", got)
	}
	if got := shapes[tracking.TongueOut].Weight; got != 1 {
		t.Errorf("TongueOut = %v, want 1", got)
	}

	// GXR layout reads the tongue from the dorsal indices instead.
	var gxr [tracking.ExpressionCount]tracking.Shape
	e[tongueTipInterdental] = 0.4
	updateMouthExpressions(&gxr, &e, true)
	if got := gxr[tracking.TongueOut].Weight; got != 0.4 {
		t.Errorf("GXR TongueOut = %v, want 0.4", got)
	}
}

func TestConvertRespectsGrantedKinds(t *testing.T) {
	m := &Module{logger: zap.NewNop(), granted: adapter.Support{Expression: true}}
	state := &faceState{FaceFlags: faceFlagValid, LeftEyeIsValid: true, IsEyeFollowingBlendshapesValid: true}
	state.ExpressionWeights[jawDrop] = 0.7
	state.ExpressionWeights[eyesClosedL] = 1

	data := tracking.NewTrackingData()
	if !m.convert(state, data) {
		t.Fatal("frame with a valid face must count as tracked")
	}
	if data.Shapes[tracking.JawOpen].Weight != 0.7 {
		t.Fatalf("JawOpen = %v", data.Shapes[tracking.JawOpen].Weight)
	}
	if data.Eye.Left.Openness != 1 {
		t.Fatalf("eye data must be untouched when eye was not granted, got openness %v", data.Eye.Left.Openness)
	}
	if state.ExpressionWeights[jawDrop] != 0.7 {
		t.Fatal("convert must not modify the frame")
	}
}

func TestSetTrackingDelaysSilenceWarning(t *testing.T) {
	m := &Module{logger: zap.NewNop(), lastFrameAt: time.Now()}
	m.setTracking(false)
	if m.silenceAlert {
		t.Fatal("warning must not fire immediately")
	}
	m.lastFrameAt = time.Now().Add(-silenceWarnAfter - time.Second)
	m.setTracking(false)
	if !m.silenceAlert {
		t.Fatal("warning expected after the silence threshold")
	}
	m.setTracking(true)
	if !m.tracking || m.silenceAlert {
		t.Fatal("tracking must clear the alert")
	}
}

// Without the Streamer the module must fail fast with a clear error.
func TestInitializeWithoutStreamerFailsFast(t *testing.T) {
	m := New()
	start := time.Now()
	_, err := m.Initialize(t.Context(), zap.NewNop(), adapter.Support{Eye: true, Expression: true})
	if err == nil {
		t.Skip("Virtual Desktop Streamer is running on this machine")
	}
	if time.Since(start) > time.Second {
		t.Fatalf("Initialize took %s", time.Since(start))
	}
	if err := m.Teardown(); err != nil {
		t.Fatalf("teardown after failed init: %v", err)
	}
}
