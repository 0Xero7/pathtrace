package main

import (
	"fmt"

	"github.com/chewxy/math32"
)

type Camera struct {
	Position         Vec3
	Forward          Vec3
	Right            Vec3
	Up               Vec3
	FrustrumDistance float32
}

func (c *Camera) SphericalAround(center Vec3, radius, phi, theta float32) {
	fmt.Println("SphericalAround called with", center, radius, phi, theta)
	c.Position = Vec3{
		X: center.X + radius*math32.Sin(theta)*math32.Cos(phi),
		Y: center.Y + radius*math32.Cos(theta),
		Z: center.Z + radius*math32.Sin(theta)*math32.Sin(phi),
	}

	c.Forward = center.Sub(c.Position).Normalize()

	// Define world up vector (assuming Y-up coordinate system)
	worldUp := Vec3{X: 0, Y: 1, Z: 0}

	// Calculate right vector: worldUp × forward
	c.Right = worldUp.Cross(c.Forward).Normalize()

	// Handle the case where forward is parallel to world up (looking straight up/down)
	if c.Right.Length() < 1e-6 {
		// Use an arbitrary perpendicular vector
		c.Right = Vec3{X: 1, Y: 0, Z: 0}
	}

	// Calculate up vector: right × forward
	c.Up = c.Right.Cross(c.Forward).Normalize()
}

// Rotate vector around Y axis (global rotation)
func rotateAroundY(v Vec3, angle float32) Vec3 {
	cos := math32.Cos(angle)
	sin := math32.Sin(angle)

	return Vec3{
		X: v.X*cos + v.Z*sin,
		Y: v.Y,
		Z: -v.X*sin + v.Z*cos,
	}
}

// Rotate vector around an arbitrary axis
func rotateAroundAxis(v Vec3, axis Vec3, angle float32) Vec3 {
	// Rodrigues' rotation formula
	cos := math32.Cos(angle)
	sin := math32.Sin(angle)

	// Ensure axis is normalized
	axis = axis.Normalize()

	// v_rot = v*cos(θ) + (k × v)*sin(θ) + k*(k·v)*(1-cos(θ))
	// where k is the rotation axis

	dot := axis.X*v.X + axis.Y*v.Y + axis.Z*v.Z
	cross := axis.Cross(v)

	return Vec3{
		X: v.X*cos + cross.X*sin + axis.X*dot*(1-cos),
		Y: v.Y*cos + cross.Y*sin + axis.Y*dot*(1-cos),
		Z: v.Z*cos + cross.Z*sin + axis.Z*dot*(1-cos),
	}
}

// Apply camera rotations
func (c *Camera) ApplyRotation(rotY, rotX float32) {
	// Step 1: Rotate around global Y axis (yaw)
	if rotY != 0 {
		c.Forward = rotateAroundY(c.Forward, rotY)
		c.Right = rotateAroundY(c.Right, rotY)
		// Up vector stays the same for global Y rotation in most cases
		// but we'll recalculate it to maintain orthogonality
	}

	// Step 2: Rotate around local X axis (pitch) - use the current Right vector
	if rotX != 0 {
		c.Forward = rotateAroundAxis(c.Forward, c.Right, rotX)
		c.Up = rotateAroundAxis(c.Up, c.Right, rotX)
	}

	// Ensure vectors are normalized and orthogonal
	c.Forward = c.Forward.Normalize()
	c.Right = c.Right.Normalize()
	c.Up = c.Up.Normalize()

	// Recalculate Up to ensure perfect orthogonality
	// Up = Right × Forward
	c.Up = c.Right.Cross(c.Forward).Normalize()
}

// Alternative: Set absolute rotation from angles
func (c *Camera) SetRotationFromAngles(yaw, pitch float32) {
	// Reset to initial orientation then apply rotations
	c.Forward = Vec3{X: 0, Y: 0, Z: 1}
	c.Right = Vec3{X: 1, Y: 0, Z: 0}
	c.Up = Vec3{X: 0, Y: -1, Z: 0}

	c.ApplyRotation(yaw, pitch)
}
