package mixer

import "github.com/SayukiDev/VRCFaceTrackingGO/pkg/tracking"

// AdjustmentConfig is the persisted state of Adjustment. groups of shapes from a user chosen
// Every range is remapped from (floor, ceil) onto 0..1.
type AdjustmentConfig struct {
	activeState
	EyeBrowRaiser  Range `json:"eyeBrowRaiser"`
	EyeBrowLower   Range `json:"eyeBrowLower"`
	EyeSquint      Range `json:"eyeSquint"`
	EyeWide        Range `json:"eyeWide"`
	CheekPuffSuck  Range `json:"cheekPuffSuck"`
	CheekSquint    Range `json:"cheekSquint"`
	JawOpen        Range `json:"jawOpen"`
	MouthClosed    Range `json:"mouthClosed"`
	JawX           Range `json:"jawX"`
	JawZ           Range `json:"jawZ"`
	LipFunnel      Range `json:"lipFunnel"`
	LipSuck        Range `json:"lipSuck"`
	LipPucker      Range `json:"lipPucker"`
	MouthOpener    Range `json:"mouthOpener"`
	MouthSmile     Range `json:"mouthSmile"`
	MouthFrown     Range `json:"mouthFrown"`
	MouthStretch   Range `json:"mouthStretch"`
	MouthTightener Range `json:"mouthTightener"`
	MouthPress     Range `json:"mouthPress"`
	MouthX         Range `json:"mouthX"`
	MouthRaiser    Range `json:"mouthRaiser"`
	Nose           Range `json:"nose"`
	NoseSneer      Range `json:"noseSneer"`
	Neck           Range `json:"neck"`
	TongueOut      Range `json:"tongueOut"`
	TongueMove     Range `json:"tongueMove"`
	TongueOther    Range `json:"tongueOther"`

	HeadRotationYaw   Range `json:"headRotationYaw"`
	HeadRotationPitch Range `json:"headRotationPitch"`
	HeadRotationRoll  Range `json:"headRotationRoll"`
	HeadPositionX     Range `json:"headPositionX"`
	HeadPositionY     Range `json:"headPositionY"`
	HeadPositionZ     Range `json:"headPositionZ"`
}

// DefaultAdjustmentConfig maps every range onto itself (no adjustment).
func DefaultAdjustmentConfig() AdjustmentConfig {
	var c AdjustmentConfig
	c.reset()
	return c
}

// Adjustment remaps groups of shapes from a user chosen (floor, ceil) range
// onto 0..1.
type Adjustment struct {
	AdjustmentConfig
}

func (p *AdjustmentConfig) reset() {
	unit := Range{Floor: 0, Ceil: 1}
	signed := Range{Floor: -1, Ceil: 1}
	for _, r := range []*Range{
		&p.EyeBrowRaiser, &p.EyeBrowLower, &p.EyeSquint, &p.EyeWide, &p.CheekPuffSuck, &p.CheekSquint,
		&p.JawOpen, &p.MouthClosed, &p.JawX, &p.JawZ, &p.LipFunnel, &p.LipSuck, &p.LipPucker,
		&p.MouthOpener, &p.MouthSmile, &p.MouthFrown, &p.MouthStretch, &p.MouthTightener, &p.MouthPress,
		&p.MouthX, &p.MouthRaiser, &p.Nose, &p.NoseSneer, &p.Neck, &p.TongueOut, &p.TongueMove, &p.TongueOther,
	} {
		*r = unit
	}
	for _, r := range []*Range{
		&p.HeadRotationYaw, &p.HeadRotationPitch, &p.HeadRotationRoll,
		&p.HeadPositionX, &p.HeadPositionY, &p.HeadPositionZ,
	} {
		*r = signed
	}
}

func (p *Adjustment) Name() string        { return "Parameter Adjustment" }
func (p *Adjustment) Description() string { return "Adjust VRCFaceTracking Parameters." }
func (p *Adjustment) Priority() Priority  { return None }

func remap(v *float32, r Range) { *v = (*v - r.Floor) / (r.Ceil - r.Floor) }

func (p *Adjustment) Mutate(d *tracking.TrackingData) {
	set := func(r Range, shapes ...tracking.UnifiedExpression) {
		for _, s := range shapes {
			remap(&d.Shapes[s].Weight, r)
		}
	}
	set(p.EyeBrowRaiser, tracking.BrowInnerUpLeft, tracking.BrowInnerUpRight, tracking.BrowOuterUpLeft, tracking.BrowOuterUpRight)
	set(p.EyeBrowLower, tracking.BrowLowererLeft, tracking.BrowLowererRight, tracking.BrowPinchLeft, tracking.BrowPinchRight)
	set(p.CheekPuffSuck, tracking.CheekPuffLeft, tracking.CheekPuffRight, tracking.CheekSuckLeft, tracking.CheekSuckRight)
	set(p.CheekSquint, tracking.CheekSquintLeft, tracking.CheekSquintRight)
	set(p.EyeSquint, tracking.EyeSquintLeft, tracking.EyeSquintRight)
	set(p.EyeWide, tracking.EyeWideLeft, tracking.EyeWideRight)
	set(p.JawZ, tracking.JawBackward, tracking.JawForward)
	set(p.JawOpen, tracking.JawClench, tracking.JawMandibleRaise, tracking.JawOpen)
	set(p.MouthClosed, tracking.MouthClosed)
	set(p.JawX, tracking.JawLeft, tracking.JawRight)
	set(p.LipFunnel, tracking.LipFunnelLowerLeft, tracking.LipFunnelLowerRight, tracking.LipFunnelUpperLeft, tracking.LipFunnelUpperRight)
	set(p.LipPucker, tracking.LipPuckerLowerLeft, tracking.LipPuckerLowerRight, tracking.LipPuckerUpperLeft, tracking.LipPuckerUpperRight)
	set(p.LipSuck, tracking.LipSuckCornerLeft, tracking.LipSuckCornerRight, tracking.LipSuckUpperLeft, tracking.LipSuckUpperRight, tracking.LipSuckLowerLeft, tracking.LipSuckLowerRight)
	set(p.MouthOpener, tracking.MouthUpperDeepenLeft, tracking.MouthUpperDeepenRight, tracking.MouthUpperUpLeft, tracking.MouthUpperUpRight, tracking.MouthLowerDownLeft, tracking.MouthLowerDownRight)
	set(p.MouthSmile, tracking.MouthCornerPullLeft, tracking.MouthCornerPullRight, tracking.MouthCornerSlantLeft, tracking.MouthCornerSlantRight)
	set(p.MouthFrown, tracking.MouthFrownLeft, tracking.MouthFrownRight)
	set(p.MouthStretch, tracking.MouthStretchLeft, tracking.MouthStretchRight)
	set(p.MouthTightener, tracking.MouthTightenerLeft, tracking.MouthTightenerRight)
	set(p.MouthPress, tracking.MouthPressRight, tracking.MouthPressLeft)
	set(p.MouthX, tracking.MouthUpperLeft, tracking.MouthUpperRight, tracking.MouthLowerLeft, tracking.MouthLowerRight)
	set(p.MouthRaiser, tracking.MouthRaiserLower, tracking.MouthRaiserUpper)
	set(p.Nose, tracking.NasalConstrictLeft, tracking.NasalConstrictRight, tracking.NasalDilationLeft, tracking.NasalDilationRight)
	set(p.NoseSneer, tracking.NoseSneerLeft, tracking.NoseSneerRight)
	set(p.Neck, tracking.NeckFlexLeft, tracking.NeckFlexRight, tracking.SoftPalateClose, tracking.ThroatSwallow)
	set(p.TongueMove, tracking.TongueBendDown, tracking.TongueCurlUp, tracking.TongueDown, tracking.TongueUp, tracking.TongueLeft, tracking.TongueRight)
	set(p.TongueOther, tracking.TongueTwistLeft, tracking.TongueTwistRight, tracking.TongueFlat, tracking.TongueSquish, tracking.TongueRoll)
	set(p.TongueOut, tracking.TongueOut)

	remap(&d.Head.Yaw, p.HeadRotationYaw)
	remap(&d.Head.Pitch, p.HeadRotationPitch)
	remap(&d.Head.Roll, p.HeadRotationRoll)
	remap(&d.Head.PosX, p.HeadPositionX)
	remap(&d.Head.PosY, p.HeadPositionY)
	remap(&d.Head.PosZ, p.HeadPositionZ)
}

func (p *Adjustment) Components() []Component {
	return []Component{
		button("Reset Values", p.reset),
		rangeSlider("Eyebrow Raiser", 0, 1, &p.EyeBrowRaiser),
		rangeSlider("Eyebrow Lowerer", 0, 1, &p.EyeBrowLower),
		rangeSlider("Eye Squint", 0, 1, &p.EyeSquint),
		rangeSlider("Eye Wide", 0, 1, &p.EyeWide),
		rangeSlider("Cheek", 0, 1, &p.CheekPuffSuck),
		rangeSlider("Cheek Squint", 0, 1, &p.CheekSquint),
		rangeSlider("Jaw", 0, 1, &p.JawOpen),
		rangeSlider("MouthClosed", 0, 1, &p.MouthClosed),
		rangeSlider("Jaw Sideways", 0, 1, &p.JawX),
		rangeSlider("Jaw Forward / Backward", 0, 1, &p.JawZ),
		rangeSlider("Lip Funnel", 0, 1, &p.LipFunnel),
		rangeSlider("Lip Suck", 0, 1, &p.LipSuck),
		rangeSlider("Lip Pucker", 0, 1, &p.LipPucker),
		rangeSlider("Mouth Open", 0, 1, &p.MouthOpener),
		rangeSlider("Mouth Smile", 0, 1, &p.MouthSmile),
		rangeSlider("Mouth Frown", 0, 1, &p.MouthFrown),
		rangeSlider("Mouth Stretch", 0, 1, &p.MouthStretch),
		rangeSlider("Mouth Tightener", 0, 1, &p.MouthTightener),
		rangeSlider("Mouth Press", 0, 1, &p.MouthPress),
		rangeSlider("Mouth Sideways", 0, 1, &p.MouthX),
		rangeSlider("Mouth Raiser", 0, 1, &p.MouthRaiser),
		rangeSlider("Nose", 0, 1, &p.Nose),
		rangeSlider("Nose Sneer", 0, 1, &p.NoseSneer),
		rangeSlider("Neck", 0, 1, &p.Neck),
		rangeSlider("Tongue Out", 0, 1, &p.TongueOut),
		rangeSlider("Tongue Directions", 0, 1, &p.TongueMove),
		rangeSlider("Tongue Miscellaneous", 0, 1, &p.TongueOther),
		rangeSlider("Head Rotation (Side-to-Side)", -1, 1, &p.HeadRotationYaw),
		rangeSlider("Head Rotation (Up-Down Tilt)", -1, 1, &p.HeadRotationPitch),
		rangeSlider("Head Rotation (Side Tilt)", -1, 1, &p.HeadRotationRoll),
		rangeSlider("Head Position (Side-to-Side)", -1, 1, &p.HeadPositionX),
		rangeSlider("Head Position (Up-Down)", -1, 1, &p.HeadPositionY),
		rangeSlider("Head Position (Forward-Back)", -1, 1, &p.HeadPositionZ),
	}
}
