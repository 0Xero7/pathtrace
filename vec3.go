package main

import (
	"image/color"
	"math"

	"github.com/g3n/engine/math32"
)

type Vec3 struct {
	X, Y, Z float64
}

func (v Vec3) Ones() Vec3 {
	v.X, v.Y, v.Z = 1, 1, 1
	return v
}

func (v Vec3) Clone() Vec3 {
	return v
}

func (v Vec3) Add(other Vec3) Vec3 {
	v._Add(other)
	return v
}
func (v *Vec3) _Add(other Vec3) {
	v.X += other.X
	v.Y += other.Y
	v.Z += other.Z
}

func (v Vec3) Sub(other Vec3) Vec3 {
	v._Sub(other)
	return v
}
func (v *Vec3) _Sub(other Vec3) {
	v.X -= other.X
	v.Y -= other.Y
	v.Z -= other.Z
}

func (v Vec3) Scale(scalar float64) Vec3 {
	v._Scale(scalar)
	return v
}
func (v *Vec3) _Scale(scalar float64) {
	v.X *= scalar
	v.Y *= scalar
	v.Z *= scalar
}

func (v *Vec3) Dot(other Vec3) float64 {
	return v.X*other.X + v.Y*other.Y + v.Z*other.Z
}

func (v Vec3) Length() float64 {
	return math.Sqrt(v.Dot(v))
}

func (v Vec3) Normalize() Vec3 {
	length := v.Length()
	if length == 0 {
		return Vec3{}
	}
	return v.Scale(1.0 / length)
}
func (v *Vec3) _Normalize() {
	length := v.Length()
	if length == 0 {
		v.X, v.Y, v.Z = 0, 0, 0
		return
	}
	scale := 1.0 / length
	v.X *= scale
	v.Y *= scale
	v.Z *= scale
}

func (v Vec3) Cross(other Vec3) Vec3 {
	return Vec3{
		X: v.Y*other.Z - v.Z*other.Y,
		Y: v.Z*other.X - v.X*other.Z,
		Z: v.X*other.Y - v.Y*other.X,
	}
}
func (v *Vec3) _Cross(other Vec3) {
	newX := v.Y*other.Z - v.Z*other.Y
	newY := v.Z*other.X - v.X*other.Z
	newZ := v.X*other.Y - v.Y*other.X
	v.X = newX
	v.Y = newY
	v.Z = newZ
}

func (v Vec3) ToRGBA() color.RGBA {
	r := Clamp01(math.Sqrt(max(0.0, v.X))) * 255
	g := Clamp01(math.Sqrt(max(0.0, v.Y))) * 255
	b := Clamp01(math.Sqrt(max(0.0, v.Z))) * 255

	return color.RGBA{
		R: uint8(r),
		G: uint8(g),
		B: uint8(b),
		A: 255,
	}
}

func (v Vec3) ComponentMul(other Vec3) Vec3 {
	v._ComponentMul(other)
	return v
}
func (v *Vec3) _ComponentMul(other Vec3) {
	v.X *= other.X
	v.Y *= other.Y
	v.Z *= other.Z
}

func (v *Vec3) Inverse() Vec3 {
	return Vec3{
		X: 1.0 / v.X,
		Y: 1.0 / v.Y,
		Z: 1.0 / v.Z,
	}
}

func FromColor(col math32.Color) Vec3 {
	return Vec3{
		X: float64(col.R),
		Y: float64(col.G),
		Z: float64(col.B),
	}
}

func (v1 Vec3) Lerp(v2 Vec3, t float64) Vec3 {
	return v1.Add(v2.Sub(v1).Scale(t))
}
