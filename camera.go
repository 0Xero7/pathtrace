package main

import "math"

type Camera struct {
	Position         Vec3
	Forward          Vec3
	Right            Vec3
	Up               Vec3
	FrustrumDistance float64
}

// Rotate vector around Y axis (global rotation)
func rotateAroundY(v Vec3, angle float64) Vec3 {
	cos := math.Cos(angle)
	sin := math.Sin(angle)

	return Vec3{
		X: v.X*cos + v.Z*sin,
		Y: v.Y,
		Z: -v.X*sin + v.Z*cos,
	}
}

// Rotate vector around an arbitrary axis
func rotateAroundAxis(v Vec3, axis Vec3, angle float64) Vec3 {
	// Rodrigues' rotation formula
	cos := math.Cos(angle)
	sin := math.Sin(angle)

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
func (c *Camera) ApplyRotation(rotY, rotX float64) {
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
func (c *Camera) SetRotationFromAngles(yaw, pitch float64) {
	// Reset to initial orientation then apply rotations
	c.Forward = Vec3{X: 0, Y: 0, Z: 1}
	c.Right = Vec3{X: 1, Y: 0, Z: 0}
	c.Up = Vec3{X: 0, Y: -1, Z: 0}

	c.ApplyRotation(yaw, pitch)
}
