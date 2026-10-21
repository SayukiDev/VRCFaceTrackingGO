package params

import (
	"strings"

	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/module/adapter"
	"github.com/SayukiDev/VRCFaceTrackingGO/pkg/tracking"
)

// StatusProvider reports the router's tracking state, used by the
// EyeTrackingActive / ExpressionTrackingActive parameters.
type StatusProvider interface {
	EyeStatus() adapter.State
	ExpressionStatus() adapter.State
}

type getter = func(*tracking.TrackingData) float32

func shape(u tracking.UnifiedExpression) getter {
	return func(d *tracking.TrackingData) float32 { return d.Shapes[u].Weight }
}

func simple(s tracking.SimpleExpression) getter {
	return func(d *tracking.TrackingData) float32 { return d.Simple(s) }
}

func avg2(a, b tracking.UnifiedExpression) getter {
	return func(d *tracking.TrackingData) float32 { return (d.Shapes[a].Weight + d.Shapes[b].Weight) / 2 }
}

func avg4(a, b, c, e tracking.UnifiedExpression) getter {
	return func(d *tracking.TrackingData) float32 {
		return (d.Shapes[a].Weight + d.Shapes[b].Weight + d.Shapes[c].Weight + d.Shapes[e].Weight) / 4
	}
}

func diff(a, b tracking.UnifiedExpression) getter {
	return func(d *tracking.TrackingData) float32 { return d.Shapes[a].Weight - d.Shapes[b].Weight }
}

func maxOf(a, b tracking.UnifiedExpression) getter {
	return func(d *tracking.TrackingData) float32 { return max(d.Shapes[a].Weight, d.Shapes[b].Weight) }
}

// VRChat's built-in eye tracking inputs. They drive bone-rotation gaze and
// the default eyelids independently of avatar parameters.
const (
	NativeGazeAddress       = "/tracking/eye/LeftRightPitchYaw"
	NativeEyesClosedAddress = "/tracking/eye/EyesClosedAmount"
)

// NativeGaze returns both eyes as pitch/yaw degrees for NativeGazeAddress.
func NativeGaze(d *tracking.TrackingData) tracking.Vector4 {
	return tracking.Vector4{
		X: d.Eye.Left.Gaze.ToPitch(), Y: d.Eye.Left.Gaze.ToYaw(),
		Z: d.Eye.Right.Gaze.ToPitch(), W: d.Eye.Right.Gaze.ToYaw(),
	}
}

// NativeEyesClosed returns the combined eyelid closure for
// NativeEyesClosedAddress.
func NativeEyesClosed(d *tracking.TrackingData) float32 { return 1 - d.Eye.Combined().Openness }

func v2(name string, get getter) Parameter { return NewEParam("v2/"+name, get, 0.5) }

// AllParameters returns every output parameter: one per Unified Expression,
// one per simple expression, the combined/compacted shapes, the native
// tracking endpoints and head parameters.
func AllParameters(status StatusProvider) []Parameter {
	var all []Parameter
	for u := tracking.UnifiedExpression(0); u < tracking.ExpressionCount; u++ {
		all = append(all, NewEParam("v2/"+u.String(), shape(u), 0))
	}
	for s := tracking.SimpleExpression(0); s < tracking.SimpleExpressionCount; s++ {
		all = append(all, NewEParam("v2/"+s.String(), simple(s), 0))
	}
	all = append(all, combinedShapes(status)...)
	all = append(all, headParameters()...)
	return all
}

// hasAnyParameterNamed reports whether one of defs addresses one of the known
// parameter names. Used to decide whether native tracking endpoints should
// be used instead of avatar parameters.
func hasAnyParameterNamed(defs []Definition, names []string) bool {
	for _, def := range defs {
		for _, n := range names {
			if strings.HasSuffix(def.Address, n) {
				return true
			}
		}
	}
	return false
}

func filterDefs(defs []Definition, pred func(Definition) bool) []Definition {
	var out []Definition
	for _, d := range defs {
		if pred(d) {
			out = append(out, d)
		}
	}
	return out
}

func combinedShapes(status StatusProvider) []Parameter {
	// Native endpoints check the avatar for eye parameters. Collect the names
	// lazily from the full parameter set (only needed on Reset, which avoids
	// infinite recursion while constructing the set).
	var knownNames []string
	knownNamesOf := func() []string {
		if knownNames == nil {
			for _, p := range AllParameters(status) {
				knownNames = append(knownNames, p.Names()...)
			}
		}
		return knownNames
	}

	eyeStatusActive := func(*tracking.TrackingData) (bool, bool) {
		s := status.EyeStatus()
		return s == adapter.Active, s != adapter.Uninitialized
	}
	expressionStatusActive := func(*tracking.TrackingData) (bool, bool) {
		s := status.ExpressionStatus()
		return s == adapter.Active, s != adapter.Uninitialized
	}

	return []Parameter{
		// Eye gaze
		NewEParamVector2("v2/Eye", func(d *tracking.TrackingData) tracking.Vector2 { return d.Eye.Combined().Gaze }, 0.5),
		NewEParamVector2("v2/EyeLeft", func(d *tracking.TrackingData) tracking.Vector2 { return d.Eye.Left.Gaze }, 0.5),
		NewEParamVector2("v2/EyeRight", func(d *tracking.TrackingData) tracking.Vector2 { return d.Eye.Right.Gaze }, 0.5),

		// Native gaze, used when the avatar has no eye X/Y parameters.
		NewNative(NativeGazeAddress, NativeGaze,
			func(defs []Definition) bool {
				eye := filterDefs(defs, func(d Definition) bool {
					return strings.Contains(d.Name, "Eye") && (strings.Contains(d.Name, "X") || strings.Contains(d.Name, "Y"))
				})
				return !hasAnyParameterNamed(eye, knownNamesOf())
			}),
		NewNative(NativeEyesClosedAddress, NativeEyesClosed,
			func(defs []Definition) bool {
				eye := filterDefs(defs, func(d Definition) bool {
					return strings.Contains(d.Name, "Eye") && (strings.Contains(d.Name, "Open") || strings.Contains(d.Name, "Lid"))
				})
				return !hasAnyParameterNamed(eye, knownNamesOf())
			}),

		// Pupils
		v2("PupilDilation", func(d *tracking.TrackingData) float32 { return d.Eye.Combined().PupilDiameterMM }),
		v2("PupilDiameterLeft", func(d *tracking.TrackingData) float32 { return d.Eye.Left.PupilDiameterMM * 0.1 }),
		v2("PupilDiameterRight", func(d *tracking.TrackingData) float32 { return d.Eye.Right.PupilDiameterMM * 0.1 }),
		v2("PupilDiameter", func(d *tracking.TrackingData) float32 {
			return (d.Eye.Left.PupilDiameterMM + d.Eye.Right.PupilDiameterMM) * .05
		}),

		// Openness
		v2("EyeOpenLeft", func(d *tracking.TrackingData) float32 { return d.Eye.Left.Openness }),
		v2("EyeOpenRight", func(d *tracking.TrackingData) float32 { return d.Eye.Right.Openness }),
		v2("EyeOpen", func(d *tracking.TrackingData) float32 { return (d.Eye.Left.Openness + d.Eye.Right.Openness) / 2 }),
		v2("EyeClosedLeft", func(d *tracking.TrackingData) float32 { return 1 - d.Eye.Left.Openness }),
		v2("EyeClosedRight", func(d *tracking.TrackingData) float32 { return 1 - d.Eye.Right.Openness }),
		v2("EyeClosed", func(d *tracking.TrackingData) float32 { return 1 - (d.Eye.Left.Openness+d.Eye.Right.Openness)/2 }),

		// Widen
		v2("EyeWide", maxOf(tracking.EyeWideLeft, tracking.EyeWideRight)),

		// Eyelids combined
		v2("EyeLidLeft", func(d *tracking.TrackingData) float32 {
			return d.Eye.Left.Openness*.75 + d.Shapes[tracking.EyeWideLeft].Weight*.25
		}),
		v2("EyeLidRight", func(d *tracking.TrackingData) float32 {
			return d.Eye.Right.Openness*.75 + d.Shapes[tracking.EyeWideRight].Weight*.25
		}),
		v2("EyeLid", func(d *tracking.TrackingData) float32 {
			return ((d.Eye.Left.Openness+d.Eye.Right.Openness)/2)*.75 +
				((d.Shapes[tracking.EyeWideRight].Weight+d.Shapes[tracking.EyeWideLeft].Weight)/2)*.25
		}),

		// Squint
		v2("EyeSquint", maxOf(tracking.EyeSquintLeft, tracking.EyeSquintRight)),
		v2("EyesSquint", maxOf(tracking.EyeSquintLeft, tracking.EyeSquintRight)),

		// Eyebrows
		v2("BrowUp", func(d *tracking.TrackingData) float32 {
			return (d.Simple(tracking.BrowUpRight) + d.Simple(tracking.BrowUpLeft)) * .5
		}),
		v2("BrowDown", func(d *tracking.TrackingData) float32 {
			return (d.Simple(tracking.BrowDownRight) + d.Simple(tracking.BrowDownLeft)) * .5
		}),
		v2("BrowInnerUp", avg2(tracking.BrowInnerUpLeft, tracking.BrowInnerUpRight)),
		v2("BrowOuterUp", avg2(tracking.BrowOuterUpLeft, tracking.BrowOuterUpRight)),
		// -1 = angry, +1 = worried
		v2("BrowExpressionRight", func(d *tracking.TrackingData) float32 {
			return min(1, d.Shapes[tracking.BrowInnerUpRight].Weight*.5+d.Shapes[tracking.BrowOuterUpRight].Weight*.5) -
				d.Simple(tracking.BrowDownRight)
		}),
		v2("BrowExpressionLeft", func(d *tracking.TrackingData) float32 {
			return min(1, d.Shapes[tracking.BrowInnerUpLeft].Weight*.5+d.Shapes[tracking.BrowOuterUpLeft].Weight*.5) -
				d.Simple(tracking.BrowDownLeft)
		}),
		v2("BrowExpression", func(d *tracking.TrackingData) float32 {
			return (min(1, (d.Shapes[tracking.BrowInnerUpRight].Weight+d.Shapes[tracking.BrowOuterUpRight].Weight)*.5) -
				d.Simple(tracking.BrowDownRight) +
				min(1, (d.Shapes[tracking.BrowInnerUpLeft].Weight+d.Shapes[tracking.BrowOuterUpLeft].Weight)*.5) -
				d.Simple(tracking.BrowDownLeft)) * .5
		}),

		// Jaw
		v2("JawX", diff(tracking.JawRight, tracking.JawLeft)),
		v2("JawZ", diff(tracking.JawForward, tracking.JawBackward)),

		// Cheeks
		v2("CheekSquint", avg2(tracking.CheekSquintLeft, tracking.CheekSquintRight)),
		v2("CheekPuffSuckLeft", diff(tracking.CheekPuffLeft, tracking.CheekSuckLeft)),
		v2("CheekPuffSuckRight", diff(tracking.CheekPuffRight, tracking.CheekSuckRight)),
		v2("CheekPuffSuck", func(d *tracking.TrackingData) float32 {
			return (d.Shapes[tracking.CheekPuffRight].Weight+d.Shapes[tracking.CheekPuffLeft].Weight)/2 -
				(d.Shapes[tracking.CheekSuckRight].Weight+d.Shapes[tracking.CheekSuckLeft].Weight)/2
		}),
		v2("CheekSuck", avg2(tracking.CheekSuckLeft, tracking.CheekSuckRight)),

		// Mouth direction
		v2("MouthUpperX", diff(tracking.MouthUpperRight, tracking.MouthUpperLeft)),
		v2("MouthLowerX", diff(tracking.MouthLowerRight, tracking.MouthLowerLeft)),
		v2("MouthX", func(d *tracking.TrackingData) float32 {
			return (d.Shapes[tracking.MouthUpperRight].Weight+d.Shapes[tracking.MouthLowerRight].Weight)/2 -
				(d.Shapes[tracking.MouthUpperLeft].Weight+d.Shapes[tracking.MouthLowerLeft].Weight)/2
		}),

		// Lips
		v2("LipSuckUpper", avg2(tracking.LipSuckUpperRight, tracking.LipSuckUpperLeft)),
		v2("LipSuckLower", avg2(tracking.LipSuckLowerRight, tracking.LipSuckLowerLeft)),
		v2("LipSuck", avg4(tracking.LipSuckUpperRight, tracking.LipSuckUpperLeft, tracking.LipSuckLowerRight, tracking.LipSuckLowerLeft)),
		v2("LipFunnelUpper", avg2(tracking.LipFunnelUpperRight, tracking.LipFunnelUpperLeft)),
		v2("LipFunnelLower", avg2(tracking.LipFunnelLowerRight, tracking.LipFunnelLowerLeft)),
		v2("LipFunnel", avg4(tracking.LipFunnelUpperRight, tracking.LipFunnelUpperLeft, tracking.LipFunnelLowerRight, tracking.LipFunnelLowerLeft)),
		v2("LipPuckerUpper", avg2(tracking.LipPuckerUpperRight, tracking.LipPuckerUpperLeft)),
		v2("LipPuckerLower", avg2(tracking.LipPuckerLowerRight, tracking.LipPuckerLowerLeft)),
		v2("LipPuckerRight", avg2(tracking.LipPuckerUpperRight, tracking.LipPuckerLowerRight)),
		v2("LipPuckerLeft", avg2(tracking.LipPuckerUpperLeft, tracking.LipPuckerLowerLeft)),
		v2("LipPucker", avg4(tracking.LipPuckerUpperRight, tracking.LipPuckerUpperLeft, tracking.LipPuckerLowerRight, tracking.LipPuckerLowerLeft)),

		v2("LipSuckFunnelUpper", func(d *tracking.TrackingData) float32 {
			return (d.Shapes[tracking.LipSuckUpperRight].Weight+d.Shapes[tracking.LipSuckUpperLeft].Weight)/2 -
				(d.Shapes[tracking.LipFunnelUpperRight].Weight+d.Shapes[tracking.LipFunnelUpperLeft].Weight)/2
		}),
		v2("LipSuckFunnelLower", func(d *tracking.TrackingData) float32 {
			return (d.Shapes[tracking.LipSuckLowerRight].Weight+d.Shapes[tracking.LipSuckLowerLeft].Weight)/2 -
				(d.Shapes[tracking.LipFunnelLowerRight].Weight+d.Shapes[tracking.LipFunnelLowerLeft].Weight)/2
		}),
		v2("LipSuckFunnelLowerLeft", diff(tracking.LipSuckLowerLeft, tracking.LipFunnelLowerLeft)),
		v2("LipSuckFunnelLowerRight", diff(tracking.LipSuckLowerRight, tracking.LipFunnelLowerRight)),
		v2("LipSuckFunnelUpperLeft", diff(tracking.LipSuckUpperLeft, tracking.LipFunnelUpperLeft)),
		v2("LipSuckFunnelUpperRight", diff(tracking.LipSuckUpperRight, tracking.LipFunnelUpperRight)),

		// Mouth combined
		v2("MouthUpperUp", avg2(tracking.MouthUpperUpRight, tracking.MouthUpperUpLeft)),
		v2("MouthLowerDown", avg2(tracking.MouthLowerDownRight, tracking.MouthLowerDownLeft)),
		v2("MouthOpen", avg4(tracking.MouthUpperUpRight, tracking.MouthUpperUpLeft, tracking.MouthLowerDownRight, tracking.MouthLowerDownLeft)),
		v2("MouthStretch", avg2(tracking.MouthStretchRight, tracking.MouthStretchLeft)),
		v2("MouthTightener", avg2(tracking.MouthTightenerRight, tracking.MouthTightenerLeft)),
		v2("MouthPress", avg2(tracking.MouthPressRight, tracking.MouthPressLeft)),
		v2("MouthDimple", avg2(tracking.MouthDimpleRight, tracking.MouthDimpleLeft)),
		v2("NoseSneer", avg2(tracking.NoseSneerRight, tracking.NoseSneerLeft)),

		v2("MouthTightenerStretch", func(d *tracking.TrackingData) float32 {
			return (d.Shapes[tracking.MouthTightenerRight].Weight+d.Shapes[tracking.MouthTightenerLeft].Weight)/2 -
				(d.Shapes[tracking.MouthStretchRight].Weight+d.Shapes[tracking.MouthStretchLeft].Weight)/2
		}),
		v2("MouthTightenerStretchLeft", diff(tracking.MouthTightenerLeft, tracking.MouthStretchLeft)),
		v2("MouthTightenerStretchRight", diff(tracking.MouthTightenerRight, tracking.MouthStretchRight)),

		// Lip corners
		v2("MouthCornerYLeft", diff(tracking.MouthCornerSlantLeft, tracking.MouthFrownLeft)),
		v2("MouthCornerYRight", diff(tracking.MouthCornerSlantRight, tracking.MouthFrownRight)),
		v2("MouthCornerY", func(d *tracking.TrackingData) float32 {
			return (d.Shapes[tracking.MouthCornerSlantLeft].Weight - d.Shapes[tracking.MouthFrownLeft].Weight +
				d.Shapes[tracking.MouthCornerSlantRight].Weight - d.Shapes[tracking.MouthFrownRight].Weight) * .5
		}),
		v2("SmileFrownRight", func(d *tracking.TrackingData) float32 {
			return d.Simple(tracking.MouthSmileRight) - d.Shapes[tracking.MouthFrownRight].Weight
		}),
		v2("SmileFrownLeft", func(d *tracking.TrackingData) float32 {
			return d.Simple(tracking.MouthSmileLeft) - d.Shapes[tracking.MouthFrownLeft].Weight
		}),
		v2("SmileFrown", func(d *tracking.TrackingData) float32 {
			return d.Simple(tracking.MouthSmileRight)*.5 + d.Simple(tracking.MouthSmileLeft)*.5 -
				d.Shapes[tracking.MouthFrownRight].Weight*.5 - d.Shapes[tracking.MouthFrownLeft].Weight*.5
		}),
		v2("SmileSadRight", func(d *tracking.TrackingData) float32 {
			return d.Simple(tracking.MouthSmileRight) - d.Simple(tracking.MouthSadRight)
		}),
		v2("SmileSadLeft", func(d *tracking.TrackingData) float32 {
			return d.Simple(tracking.MouthSmileLeft) - d.Simple(tracking.MouthSadLeft)
		}),
		v2("SmileSad", func(d *tracking.TrackingData) float32 {
			return (d.Simple(tracking.MouthSmileLeft)+d.Simple(tracking.MouthSmileRight))/2 -
				(d.Simple(tracking.MouthSadLeft)+d.Simple(tracking.MouthSadRight))/2
		}),

		// Tongue
		v2("TongueX", diff(tracking.TongueRight, tracking.TongueLeft)),
		v2("TongueY", diff(tracking.TongueUp, tracking.TongueDown)),
		v2("TongueArchY", diff(tracking.TongueCurlUp, tracking.TongueBendDown)),
		v2("TongueShape", diff(tracking.TongueFlat, tracking.TongueSquish)),

		NewConditionalBool("EyeTrackingActive", eyeStatusActive),
		NewConditionalBool("ExpressionTrackingActive", expressionStatusActive),
		NewConditionalBool("LipTrackingActive", expressionStatusActive),
	}
}

func headParameters() []Parameter {
	return []Parameter{
		v2("Head/Yaw", func(d *tracking.TrackingData) float32 { return d.Head.Yaw }),
		v2("Head/Pitch", func(d *tracking.TrackingData) float32 { return d.Head.Pitch }),
		v2("Head/Roll", func(d *tracking.TrackingData) float32 { return d.Head.Roll }),
		v2("Head/PosX", func(d *tracking.TrackingData) float32 { return d.Head.PosX }),
		v2("Head/PosY", func(d *tracking.TrackingData) float32 { return d.Head.PosY }),
		v2("Head/PosZ", func(d *tracking.TrackingData) float32 { return d.Head.PosZ }),
	}
}
