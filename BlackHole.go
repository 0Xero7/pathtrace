package main

import (
	"math"

	"github.com/aquilax/go-perlin"
	"github.com/g3n/engine/math32"
)

type BlackHole struct {
	Position      Vec3
	Rs            float32
	AccretionDisk *AccretionDisk
}

type RayState struct {
	P_x, P_y, P_z float32 // 4D Position
	V_x, V_y, V_z float32 // 4D Velocity
	P_t, V_t      float32
}

// GetInitialState calculates the starting state of the ray in Cartesian coordinates.
// This function is now much simpler as no coordinate system conversion is needed.
func GetInitialState(
	position, direction Vec3,
	blackHole *BlackHole,
) *RayState {
	// The state's spatial components are simply the ray's properties
	// relative to the black hole's position.
	relativePos := position.Sub(blackHole.Position)
	r := relativePos.Length()

	// Handle the case where the ray starts exactly at the center.
	if r == 0 {
		return &RayState{}
	}

	// For a photon, the spacetime interval ds^2 is 0. We use this to find V_t.
	// This involves solving a quadratic equation derived from the Kerr-Schild metric.
	rs := blackHole.Rs
	p_dot_v := relativePos.Dot(direction)
	v_dot_v := direction.Dot(direction) // This should be 1 if direction is normalized.

	// Quadratic equation coefficients: A*v_t^2 + B*v_t + C = 0
	A := rs/r - 1.0
	B := 2.0 * rs * p_dot_v / (r * r)
	C := v_dot_v + rs*p_dot_v*p_dot_v/(r*r*r)

	// Solve for V_t using the quadratic formula. We take the positive root
	// as we are tracing forward in coordinate time.
	discriminant := B*B - 4*A*C
	if discriminant < 0 {
		// Should not happen for physical paths outside the event horizon.
		// Return a zero state to indicate an error.
		return &RayState{}
	}
	V_t := (-B + math32.Sqrt(discriminant)) / (2 * A)

	return &RayState{
		P_t: 0.0, // Start at coordinate time 0
		P_x: float32(relativePos.X),
		P_y: float32(relativePos.Y),
		P_z: float32(relativePos.Z),
		V_t: float32(V_t),
		V_x: float32(direction.X),
		V_y: float32(direction.Y),
		V_z: float32(direction.Z),
	}
}

// GetAcceleration calculates the 4D acceleration using the geodesic equations
// derived from the Schwarzschild metric in Kerr-Schild Cartesian coordinates.
// This form is free of polar singularities.
func GetAcceleration(state *RayState, blackHole *BlackHole) (float32, float32, float32, float32) {
	rs := blackHole.Rs

	// Cache state values
	px, py, pz := state.P_x, state.P_y, state.P_z
	vx, vy, vz := state.V_x, state.V_y, state.V_z

	r_sq := px*px + py*py + pz*pz

	// Early exit check without sqrt
	if r_sq <= rs*rs {
		return 0, 0, 0, 0
	}

	r := math32.Sqrt(r_sq)
	r_cubed := r_sq * r
	inv_r_cubed := 1.0 / r_cubed

	// Cache repeated calculations
	x_dot_v := px*vx + py*vy + pz*vz
	x_dot_v_sq := x_dot_v * x_dot_v
	rs_x_dot_v := rs * x_dot_v
	two_x_dot_v_inv_r_cubed := 2.0 * x_dot_v * inv_r_cubed

	// Time acceleration
	accel_t := -2.0 * rs_x_dot_v * inv_r_cubed

	// Spatial acceleration
	factor := (1.0 - 3.0*rs*x_dot_v_sq*inv_r_cubed) * inv_r_cubed
	rs_inv_factor := -rs * factor

	accel_x := px*rs_inv_factor + rs*two_x_dot_v_inv_r_cubed*vx
	accel_y := py*rs_inv_factor + rs*two_x_dot_v_inv_r_cubed*vy
	accel_z := pz*rs_inv_factor + rs*two_x_dot_v_inv_r_cubed*vz

	return accel_t, accel_x, accel_y, accel_z
}

// RK4 helper structs and functions remain the same, but now operate on Cartesian components.
type RK4Derivative struct {
	Accel_t, Accel_x, Accel_y, Accel_z float32
	Vel_t, Vel_x, Vel_y, Vel_z         float32
}

func evaluateRK4(initialState *RayState, dt float32, deriv *RK4Derivative, outState *RayState) {
	outState.P_t = initialState.P_t + deriv.Vel_t*dt
	outState.P_x = initialState.P_x + deriv.Vel_x*dt
	outState.P_y = initialState.P_y + deriv.Vel_y*dt
	outState.P_z = initialState.P_z + deriv.Vel_z*dt
	outState.V_t = initialState.V_t + deriv.Accel_t*dt
	outState.V_x = initialState.V_x + deriv.Accel_x*dt
	outState.V_y = initialState.V_y + deriv.Accel_y*dt
	outState.V_z = initialState.V_z + deriv.Accel_z*dt
}

// Step advances the ray's state using the RK4 integrator. Its structure is identical,
// but it now works with the Cartesian state and acceleration.
func Step(state *RayState, stepSize float32, blackHole *BlackHole) *RayState {
	// Stage 1
	k1_at, k1_ax, k1_ay, k1_az := GetAcceleration(state, blackHole)
	k1 := RK4Derivative{k1_at, k1_ax, k1_ay, k1_az, state.V_t, state.V_x, state.V_y, state.V_z}

	tempState := RayState{}
	// Stage 2
	evaluateRK4(state, stepSize*0.5, &k1, &tempState)
	k2_at, k2_ax, k2_ay, k2_az := GetAcceleration(&tempState, blackHole)
	k2 := RK4Derivative{k2_at, k2_ax, k2_ay, k2_az, tempState.V_t, tempState.V_x, tempState.V_y, tempState.V_z}

	// Stage 3
	evaluateRK4(state, stepSize*0.5, &k2, &tempState)
	k3_at, k3_ax, k3_ay, k3_az := GetAcceleration(&tempState, blackHole)
	k3 := RK4Derivative{k3_at, k3_ax, k3_ay, k3_az, tempState.V_t, tempState.V_x, tempState.V_y, tempState.V_z}

	// Stage 4
	evaluateRK4(state, stepSize, &k3, &tempState)
	k4_at, k4_ax, k4_ay, k4_az := GetAcceleration(&tempState, blackHole)
	k4 := RK4Derivative{k4_at, k4_ax, k4_ay, k4_az, tempState.V_t, tempState.V_x, tempState.V_y, tempState.V_z}

	// Combine
	tempState.P_t = state.P_t + (stepSize/6.0)*(k1.Vel_t+2*k2.Vel_t+2*k3.Vel_t+k4.Vel_t)
	tempState.P_x = state.P_x + (stepSize/6.0)*(k1.Vel_x+2*k2.Vel_x+2*k3.Vel_x+k4.Vel_x)
	tempState.P_y = state.P_y + (stepSize/6.0)*(k1.Vel_y+2*k2.Vel_y+2*k3.Vel_y+k4.Vel_y)
	tempState.P_z = state.P_z + (stepSize/6.0)*(k1.Vel_z+2*k2.Vel_z+2*k3.Vel_z+k4.Vel_z)
	tempState.V_t = state.V_t + (stepSize/6.0)*(k1.Accel_t+2*k2.Accel_t+2*k3.Accel_t+k4.Accel_t)
	tempState.V_x = state.V_x + (stepSize/6.0)*(k1.Accel_x+2*k2.Accel_x+2*k3.Accel_x+k4.Accel_x)
	tempState.V_y = state.V_y + (stepSize/6.0)*(k1.Accel_y+2*k2.Accel_y+2*k3.Accel_y+k4.Accel_y)
	tempState.V_z = state.V_z + (stepSize/6.0)*(k1.Accel_z+2*k2.Accel_z+2*k3.Accel_z+k4.Accel_z)

	return &tempState
}

// AccretionDisk represents the parameters for our procedural disk.
type AccretionDisk struct {
	InnerRadius float32        // The radius where the disk starts.
	OuterRadius float32        // The radius where the disk ends.
	NoiseGen    *perlin.Perlin // Assuming a Perlin noise generator.
}

// GetProceduralColor calculates the base emissive color of the accretion disk
// at a specific point in world space.
func (disk *AccretionDisk) GetProceduralColor(worldPosition Vec3, blackHolePosition Vec3) Vec3 {
	// 1. Get the point's position relative to the black hole.
	relativePos := worldPosition.Sub(blackHolePosition)

	// 2. Convert to 2D polar coordinates (radius and angle) for texturing.
	// We'll use the XZ plane for a Y-up renderer.
	radius := math32.Sqrt(relativePos.X*relativePos.X + relativePos.Z*relativePos.Z)
	angle := math32.Atan2(relativePos.Z, relativePos.X)

	// --- Layer 1: Temperature Gradient ---

	// Normalize the radius to a [0, 1] range based on the disk's bounds.
	// This `t` value is our gradient parameter.
	t := (radius - disk.InnerRadius) / (disk.OuterRadius - disk.InnerRadius)

	// Clamp t to prevent colors outside the gradient.
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}

	// Define our color gradient: hot white -> yellow -> orange -> deep red.
	hotColor := Vec3{X: 1.0, Y: 1.0, Z: 0.85} // Bright, slightly yellow white
	midColor := Vec3{X: 1.0, Y: 0.6, Z: 0.0}  // Orange
	coolColor := Vec3{X: 0.8, Y: 0.1, Z: 0.0} // Deep red

	var tempColor Vec3
	if t < 0.5 {
		// Lerp between hot and mid colors for the inner half.
		tempColor = hotColor.Lerp(midColor, t*2.0)
	} else {
		// Lerp between mid and cool colors for the outer half.
		tempColor = midColor.Lerp(coolColor, (t-0.5)*2.0)
	}

	// The intensity should also fall off dramatically with distance.
	// We can use an inverse square falloff for a more physical feel.
	intensity := 1.0 / (t*t + 0.1) // The +0.1 prevents division by zero and keeps the center bright.
	tempColor = tempColor.Scale(intensity)

	// --- Layer 2: Turbulent Noise ---

	// We sample the 2D noise function using polar coordinates.
	// Manipulating these coordinates is how we create the swirling effect.
	noiseScale := float32(3.0)    // Controls the "zoom" level of the noise. Higher is more detailed.
	stretchFactor := float32(2.0) // Stretches the noise along the angle to create a fibrous look.

	// Convert polar to distorted Cartesian coordinates for noise sampling.
	noiseX := (radius / disk.OuterRadius) * noiseScale * stretchFactor
	noiseY := (angle / (2 * math.Pi)) * noiseScale

	// Get a noise value, map it from [-1, 1] to [0, 1].
	noiseValue := float32(disk.NoiseGen.Noise2D(float64(noiseX), float64(noiseY))+1.0) / 2.0

	// We can raise the noise to a power to increase contrast and create sharper "filaments".
	noiseValue = math32.Pow(float32(noiseValue), float32(5.0))

	// Add a minimum brightness to the noise to ensure the disk is always visible.
	// noiseValue = noiseValue*0.95 + 0.05

	// --- Final Combination ---

	// Multiply the temperature color by the noise intensity.
	finalColor := tempColor.Scale(noiseValue)

	return finalColor
}
