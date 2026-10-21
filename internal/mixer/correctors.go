package mixer

import "github.com/SayukiDev/VRCFaceTrackingGO/pkg/tracking"

// CorrectorsConfig is the persisted state of Correctors.
type CorrectorsConfig struct {
	activeState
	MouthClosedFix bool `json:"mouthClosedFix"`
	LipSuckFix     bool `json:"lipSuckFix"`
	// EyeLidBlend is how much influence the opposite eye has on openness:
	// 0 = none, 1 = both eyelids become the median.
	EyeLidBlend       float32 `json:"eyeLidBlend"`
	EyeLookSymmetrize bool    `json:"eyeLookSymmetrize"`
}

func DefaultCorrectorsConfig() CorrectorsConfig {
	return CorrectorsConfig{
		activeState:    activeState{IsActive: true},
		MouthClosedFix: true,
		LipSuckFix:     true,
	}
}

// Correctors nudges data into shapes that better conform to Unified
// Expressions.
type Correctors struct {
	CorrectorsConfig
}

func (c *Correctors) Name() string        { return "Unified Correctors" }
func (c *Correctors) Description() string { return "Processes data to conform to Unified Expressions." }
func (c *Correctors) Priority() Priority  { return Postprocessor }

func (c *Correctors) blend(current, influencer float32) float32 {
	return max(0, min(1, current*(1-c.EyeLidBlend*0.5)+influencer*(c.EyeLidBlend*0.5)))
}

func (c *Correctors) blendPair(left, right *float32) {
	*left = c.blend(*left, *right)
	*right = c.blend(*right, *left)
}

func (c *Correctors) blendShapes(d *tracking.TrackingData, left, right tracking.UnifiedExpression) {
	c.blendPair(&d.Shapes[left].Weight, &d.Shapes[right].Weight)
}

func (c *Correctors) Mutate(d *tracking.TrackingData) {
	if c.MouthClosedFix {
		d.Shapes[tracking.MouthClosed].Weight = min(d.Shapes[tracking.MouthClosed].Weight, d.Shapes[tracking.JawOpen].Weight)
	}
	if c.LipSuckFix {
		d.Shapes[tracking.LipSuckLowerLeft].Weight *= 1 - d.Shapes[tracking.MouthLowerDownLeft].Weight
		d.Shapes[tracking.LipSuckLowerRight].Weight *= 1 - d.Shapes[tracking.MouthLowerDownRight].Weight
		d.Shapes[tracking.LipSuckUpperLeft].Weight *= 1 - d.Shapes[tracking.MouthUpperUpLeft].Weight
		d.Shapes[tracking.LipSuckUpperRight].Weight *= 1 - d.Shapes[tracking.MouthUpperUpRight].Weight
	}
	if c.EyeLidBlend > 0 {
		c.blendPair(&d.Eye.Left.Openness, &d.Eye.Right.Openness)
		c.blendShapes(d, tracking.EyeWideLeft, tracking.EyeWideRight)
		c.blendShapes(d, tracking.EyeSquintLeft, tracking.EyeSquintRight)
		c.blendPair(&d.Eye.Left.PupilDiameterMM, &d.Eye.Right.PupilDiameterMM)
		c.blendShapes(d, tracking.BrowPinchLeft, tracking.BrowPinchRight)
		c.blendShapes(d, tracking.BrowLowererLeft, tracking.BrowLowererRight)
		c.blendShapes(d, tracking.BrowInnerUpLeft, tracking.BrowInnerUpRight)
		c.blendShapes(d, tracking.BrowOuterUpLeft, tracking.BrowOuterUpRight)
	}
	if c.EyeLookSymmetrize {
		// Human eyes cannot diverge vertically, so the median is safe.
		y := (d.Eye.Right.Gaze.Y + d.Eye.Left.Gaze.Y) / 2
		d.Eye.Right.Gaze.Y = y
		d.Eye.Left.Gaze.Y = y
	}
}

func (c *Correctors) Components() []Component {
	return []Component{
		checkbox("MouthClosed/JawOpen Clamp", &c.MouthClosedFix),
		checkbox("LipSuck Limiter", &c.LipSuckFix),
		slider("EyeLid Blend", 0, 1, &c.EyeLidBlend),
		checkbox("EyeLook Symmetrize", &c.EyeLookSymmetrize),
	}
}
