package tracking

import "math"

// Vector2 is a 2D float vector. Gaze vectors store X (yaw axis) and Y (pitch
// axis) in a normalised, tangent-like space.
type Vector2 struct {
	X, Y float32
}

func (v Vector2) Add(o Vector2) Vector2 { return Vector2{v.X + o.X, v.Y + o.Y} }
func (v Vector2) Sub(o Vector2) Vector2 { return Vector2{v.X - o.X, v.Y - o.Y} }
func (v Vector2) Mul(s float32) Vector2 { return Vector2{v.X * s, v.Y * s} }
func (v Vector2) Div(s float32) Vector2 { return Vector2{v.X / s, v.Y / s} }

// Vector4 is a 4-component float vector, used for combined gaze output.
type Vector4 struct {
	X, Y, Z, W float32
}

// ToYaw converts the X component into degrees.
func (v Vector2) ToYaw() float32 { return float32(math.Atan(float64(v.X)) * (180 / math.Pi)) }

// ToPitch converts the Y component into degrees (positive = down).
func (v Vector2) ToPitch() float32 { return -float32(math.Atan(float64(v.Y)) * (180 / math.Pi)) }
