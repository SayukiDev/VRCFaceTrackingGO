package mutation

import (
	"math"

	"VRCFaceTracking/pkg/tracking"
)

const filterHz = 10

// euroFilter is a One Euro filter (Casiez et al.) for a single channel.
type euroFilter struct {
	xPrev, dxPrev float32
}

func alpha(hz, cutoff float32) float32 {
	tau := 1 / (2 * math.Pi * cutoff)
	te := 1 / hz
	return 1 / (1 + tau/te)
}

func lowPass(prev *float32, x, a float32) float32 {
	*prev = a*x + (1-a)**prev
	return *prev
}

func (f *euroFilter) filter(x, minCutoff, beta, dCutoff float32) float32 {
	if math.IsNaN(float64(x)) {
		return 0
	}
	dx := (x - f.xPrev) * filterHz
	edx := lowPass(&f.dxPrev, dx, alpha(filterHz, dCutoff))
	cutoff := minCutoff + beta*float32(math.Abs(float64(edx)))
	return lowPass(&f.xPrev, x, alpha(filterHz, cutoff))
}

// Filter smooths every channel with a One Euro filter.
type Filter struct {
	activeState
	MinCutoff float32 `json:"minCutoff"`
	Beta      float32 `json:"beta"`
	DCutoff   float32 `json:"dCutoff"`

	shapes                       [tracking.ExpressionCount]euroFilter
	gazeLeftX, gazeLeftY         euroFilter
	gazeRightX, gazeRightY       euroFilter
	pupilLeft, pupilRight        euroFilter
	opennessLeft, opennessRight  euroFilter
	headYaw, headPitch, headRoll euroFilter
	headPosX, headPosY, headPosZ euroFilter
}

func NewFilter() *Filter {
	return &Filter{MinCutoff: 1, Beta: 0.5, DCutoff: 0.1}
}

func (f *Filter) Name() string { return "Data Filter" }
func (f *Filter) Description() string {
	return "Default data filtering for VRCFaceTracking expressions."
}
func (f *Filter) Priority() Priority                { return Postprocessor }
func (f *Filter) Persisted() bool                   { return true }
func (f *Filter) Initialize(*tracking.TrackingData) {}

func (f *Filter) apply(ef *euroFilter, v *float32) {
	*v = ef.filter(*v, f.MinCutoff, f.Beta, f.DCutoff)
}

func (f *Filter) Mutate(d *tracking.TrackingData) {
	for i := range d.Shapes {
		f.apply(&f.shapes[i], &d.Shapes[i].Weight)
	}
	f.apply(&f.opennessLeft, &d.Eye.Left.Openness)
	f.apply(&f.pupilLeft, &d.Eye.Left.PupilDiameterMM)
	f.apply(&f.gazeLeftX, &d.Eye.Left.Gaze.X)
	f.apply(&f.gazeLeftY, &d.Eye.Left.Gaze.Y)

	f.apply(&f.opennessRight, &d.Eye.Right.Openness)
	f.apply(&f.pupilRight, &d.Eye.Right.PupilDiameterMM)
	f.apply(&f.gazeRightX, &d.Eye.Right.Gaze.X)
	f.apply(&f.gazeRightY, &d.Eye.Right.Gaze.Y)

	f.apply(&f.headYaw, &d.Head.Yaw)
	f.apply(&f.headPitch, &d.Head.Pitch)
	f.apply(&f.headRoll, &d.Head.Roll)
	f.apply(&f.headPosX, &d.Head.PosX)
	f.apply(&f.headPosY, &d.Head.PosY)
	f.apply(&f.headPosZ, &d.Head.PosZ)
}

func (f *Filter) Components() []Component {
	return []Component{
		slider("Minimum Cutoff", true, 0, 2, &f.MinCutoff),
		slider("Beta", true, 0, 1, &f.Beta),
		slider("Derivative Cutoff", true, 0, 1, &f.DCutoff),
	}
}
