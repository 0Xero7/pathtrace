// package main

// import "math"

// type BlackHole struct {
// 	Position Vec3
// 	Rs       float64
// }

// type RayState struct {
// 	P_t, P_r, P_theta, P_phi float64
// 	V_t, V_r, V_theta, V_phi float64
// }

// func GetInitialState2(
// 	position, direction Vec3,
// 	blackHole *BlackHole,
// ) *RayState {
// 	dist := position.Sub(blackHole.Position)
// 	dist.Y, dist.Z = dist.Z, dist.Y

// 	P_r := dist.Length()
// 	P_t := 0.0
// 	P_theta := math.Acos(dist.Z / P_r)
// 	P_phi := math.Atan2(dist.Y, dist.X)

// 	V_r := dist.Dot(direction) / P_r
// 	V_theta := math.Cos(P_theta)*((dist.X*direction.X)+(dist.Y*direction.Y)) - (dist.Z * direction.Z * math.Sin(P_theta))
// 	V_theta /= (P_r * P_r)

// 	V_phi := -dist.Y*direction.X + dist.X*direction.Y
// 	V_phi /= ((dist.X * dist.X) + (dist.Y * dist.Y))

// 	g_tt := -1.0 + blackHole.Rs/P_r
// 	g_rr := -1.0 / g_tt
// 	g_theta_theta := P_r * P_r
// 	g_phi_phi := g_theta_theta * math.Pow(math.Sin(P_theta), 2)
// 	D := (g_rr * V_r * V_r) + (g_theta_theta * V_theta * V_theta) + (g_phi_phi * V_phi * V_phi)
// 	D *= -1.0 / g_tt
// 	V_t := math.Sqrt(D)

// 	return &RayState{
// 		P_t:     P_t,
// 		P_r:     P_r,
// 		P_theta: P_theta,
// 		P_phi:   P_phi,

// 		V_t:     V_t,
// 		V_r:     V_r,
// 		V_theta: V_theta,
// 		V_phi:   V_phi,
// 	}
// }

// func GetInitialState(
// 	position, direction Vec3,
// 	blackHole *BlackHole,
// ) *RayState {
// 	// --- Position Conversion (Correct for Y-up) ---
// 	dist := position.Sub(blackHole.Position)
// 	P_r := dist.Length()
// 	P_t := 0.0
// 	// For Y-up, theta is the angle from the positive Y-axis.
// 	P_theta := math.Acos(dist.Y / P_r)
// 	// For Y-up, phi is the angle in the XZ plane.
// 	P_phi := math.Atan2(dist.Z, dist.X)

// 	// --- Velocity Conversion (Correct for Y-up) ---
// 	// V_r is coordinate-system independent and remains correct.
// 	V_r := dist.Dot(direction) / P_r

// 	// This check prevents division by zero if the ray is fired straight up/down the pole.
// 	if (dist.X*dist.X)+(dist.Z*dist.Z) == 0 {
// 		// If at the pole, angular velocities in the plane are zero.
// 		// We can return a state directly or handle as a special case.
// 		// For simplicity, we can proceed, as V_phi will be handled,
// 		// and V_theta's formula is stable.
// 	}

// 	// Correct Y-up formula for V_theta.
// 	V_theta := (math.Cos(P_theta)*(dist.X*direction.X+dist.Z*direction.Z) - math.Sin(P_theta)*dist.Y*direction.Y) / (P_r * P_r)

// 	// Correct Y-up formula for V_phi.
// 	var V_phi float64
// 	xz_dist_sq := (dist.X * dist.X) + (dist.Z * dist.Z)
// 	if xz_dist_sq == 0 {
// 		V_phi = 0
// 	} else {
// 		V_phi = (dist.X*direction.Z - dist.Z*direction.X) / xz_dist_sq
// 	}

// 	// --- Time Velocity Calculation (Correct) ---
// 	g_tt := -1.0 + blackHole.Rs/P_r
// 	g_rr := 1.0 / (1.0 - blackHole.Rs/P_r)
// 	g_theta_theta := P_r * P_r
// 	g_phi_phi := g_theta_theta * math.Pow(math.Sin(P_theta), 2)

// 	D := (g_rr * V_r * V_r) + (g_theta_theta * V_theta * V_theta) + (g_phi_phi * V_phi * V_phi)
// 	V_t := math.Sqrt(D / -g_tt)

// 	return &RayState{
// 		P_t: P_t, P_r: P_r, P_theta: P_theta, P_phi: P_phi,
// 		V_t: V_t, V_r: V_r, V_theta: V_theta, V_phi: V_phi,
// 	}
// }

// func GetAcceleration(
// 	state *RayState,
// 	blackHole *BlackHole,
// ) (float64, float64, float64, float64) {

// 	rs := blackHole.Rs
// 	r := state.P_r
// 	th := state.P_theta

// 	factor := 1.0 - rs/r
// 	factor2 := rs / (2 * r * r)

// 	C_r_tt := factor2 * factor
// 	C_r_rr := -factor2 / factor
// 	C_r_thth := -r * factor
// 	C_r_phph := -r * math.Pow(math.Sin(th), 2) * factor
// 	C_t_tr := factor2 / factor
// 	C_th_rth := 1 / r
// 	C_th_phph := -math.Sin(th) * math.Cos(th)
// 	C_ph_rph := 1 / r
// 	C_ph_thph := 1.0 / math.Tan(th)

// 	accel_t := 2 * C_t_tr * state.V_t * state.V_r

// 	accel_r := (C_r_phph * state.V_phi * state.V_phi) +
// 		(C_r_rr * state.V_r * state.V_r) +
// 		(C_r_thth * state.V_theta * state.V_theta) +
// 		(C_r_tt * state.V_t * state.V_t)

// 	accel_theta := (C_th_phph * state.V_phi * state.V_phi) +
// 		(2 * C_th_rth * state.V_r * state.V_theta)

// 	accel_phi := (2 * C_ph_rph * state.V_r * state.V_phi) +
// 		(2 * C_ph_thph * state.V_theta * state.V_phi)

// 	return -accel_t, -accel_r, -accel_theta, -accel_phi
// }

// func Step(state *RayState, stepSize float64, blackHole *BlackHole) *RayState {
// 	accel_t, accel_r, accel_theta, accel_phi := GetAcceleration(state, blackHole)

// 	vel_t := state.V_t + accel_t*stepSize
// 	vel_r := state.V_r + accel_r*stepSize
// 	vel_theta := state.V_theta + accel_theta*stepSize
// 	vel_phi := state.V_phi + accel_phi*stepSize

// 	p_t := state.P_t + vel_t*stepSize
// 	p_r := state.P_r + vel_r*stepSize
// 	p_theta := state.P_theta + vel_theta*stepSize
// 	p_phi := state.P_phi + vel_phi*stepSize

// 	return &RayState{
// 		P_t:     p_t,
// 		P_r:     p_r,
// 		P_theta: p_theta,
// 		P_phi:   p_phi,

// 		V_t:     vel_t,
// 		V_r:     vel_r,
// 		V_theta: vel_theta,
// 		V_phi:   vel_phi,
// 	}
// }

// // Assume Vec3 and BlackHole/RayState types are defined elsewhere
// // SphericalToWorld converts a ray's state back to Y-up Cartesian coordinates.
// // THIS FUNCTION HAS BEEN REPLACED WITH THE CORRECT Y-UP VERSION.
// func SphericalToWorld(state *RayState, blackHole *BlackHole) (Vec3, Vec3) {
// 	sin_theta := math.Sin(state.P_theta)
// 	cos_theta := math.Cos(state.P_theta)
// 	sin_phi := math.Sin(state.P_phi)
// 	cos_phi := math.Cos(state.P_phi)

// 	// --- Position Conversion (Correct for Y-up) ---
// 	relativePos := Vec3{
// 		X: state.P_r * sin_theta * cos_phi,
// 		Y: state.P_r * cos_theta,
// 		Z: state.P_r * sin_theta * sin_phi,
// 	}
// 	worldPos := relativePos.Add(blackHole.Position)

// 	// --- Direction Conversion (Correct for Y-up) ---
// 	vx := (sin_theta*cos_phi*state.V_r +
// 		cos_theta*cos_phi*state.P_r*state.V_theta -
// 		sin_phi*state.P_r*sin_theta*state.V_phi)

// 	vy := (cos_theta*state.V_r -
// 		sin_theta*state.P_r*state.V_theta)

// 	vz := (sin_theta*sin_phi*state.V_r +
// 		cos_theta*sin_phi*state.P_r*state.V_theta +
// 		cos_phi*state.P_r*sin_theta*state.V_phi)

// 	worldDir := Vec3{X: vx, Y: vy, Z: vz}.Normalize()

// 	return worldPos, worldDir
// }

// // SphericalToWorld converts a ray's state from spherical coordinates (relative to a black hole)
// // back to a 3D world position and a 3D direction vector.
// func SphericalToWorld2(state *RayState, blackHole *BlackHole) (Vec3, Vec3) {
// 	// Pre-calculate sin and cos of the angles for efficiency and readability.
// 	sin_theta := math.Sin(state.P_theta)
// 	cos_theta := math.Cos(state.P_theta)
// 	sin_phi := math.Sin(state.P_phi)
// 	cos_phi := math.Cos(state.P_phi)

// 	// --- 1. Calculate Position ---
// 	// Convert the spherical position to a Cartesian vector relative to the black hole.
// 	relativePos := Vec3{
// 		X: state.P_r * sin_theta * cos_phi,
// 		Y: state.P_r * cos_theta,
// 		Z: state.P_r * sin_theta * sin_phi,
// 	}
// 	// Translate the relative position to get the final world position.
// 	worldPos := relativePos.Add(blackHole.Position)

// 	// --- 2. Calculate Direction ---
// 	// Convert the spherical velocity to a Cartesian direction vector.
// 	// This uses the full transformation equations involving both position and velocity components.
// 	vx := (sin_theta * cos_phi * state.V_r) +
// 		(cos_theta * cos_phi * state.P_r * state.V_theta) -
// 		(sin_phi * state.P_r * sin_theta * state.V_phi)

// 	vy := (sin_theta * sin_phi * state.V_r) +
// 		(cos_theta * sin_phi * state.P_r * state.V_theta) +
// 		(cos_phi * state.P_r * sin_theta * state.V_phi)

// 	vz := (cos_theta * state.V_r) -
// 		(sin_theta * state.P_r * state.V_theta)

// 	worldDir := Vec3{X: vx, Y: vy, Z: vz}

// 	// Since this is a direction, it should be normalized for most uses in a ray tracer.
// 	// You might want to do this here or in the calling function.
// 	worldDir = worldDir.Normalize()

// 	return worldPos, worldDir
// }

// // func SphericalToWorldPos2(state *RayState, blackHole *BlackHole) (Vec3, Vec3) {
// // 	return Vec3{
// // 		X: state.P_r * math.Sin(state.P_theta) * math.Cos(state.P_phi),
// // 		Y: state.P_r * math.Sin(state.P_theta) * math.Sin(state.P_phi),
// // 		Z: state.P_r * math.Cos(state.P_theta),
// // 	}.Add(blackHole.Position)
// // }

package main

import "math"

// NOTE: Vec3 struct and its methods (Sub, Length, Dot, Scale, Add, Normalize)
// are assumed to be defined elsewhere.

// BlackHole struct remains the same.
type BlackHole struct {
	Position Vec3
	Rs       float64
}

// RayState is now fully Cartesian, eliminating the need for spherical coordinates.
type RayState struct {
	P_t, P_x, P_y, P_z float64 // 4D Position
	V_t, V_x, V_y, V_z float64 // 4D Velocity
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
	V_t := (-B + math.Sqrt(discriminant)) / (2 * A)

	return &RayState{
		P_t: 0.0, // Start at coordinate time 0
		P_x: relativePos.X,
		P_y: relativePos.Y,
		P_z: relativePos.Z,
		V_t: V_t,
		V_x: direction.X,
		V_y: direction.Y,
		V_z: direction.Z,
	}
}

// GetAcceleration calculates the 4D acceleration using the geodesic equations
// derived from the Schwarzschild metric in Kerr-Schild Cartesian coordinates.
// This form is free of polar singularities.
func GetAcceleration(
	state *RayState,
	blackHole *BlackHole,
) (float64, float64, float64, float64) {
	rs := blackHole.Rs

	// Calculate Euclidean radius from the state's Cartesian position.
	r_sq := state.P_x*state.P_x + state.P_y*state.P_y + state.P_z*state.P_z
	r := math.Sqrt(r_sq)

	if r <= rs {
		return 0, 0, 0, 0 // Inside or at the event horizon.
	}

	r_cubed := r_sq * r

	// Calculate dot product of position and velocity vectors.
	x_dot_v := state.P_x*state.V_x + state.P_y*state.V_y + state.P_z*state.V_z

	// --- Time Acceleration ---
	accel_t := -2.0 * rs * x_dot_v / r_cubed

	// --- Spatial Acceleration ---
	// This factor appears in the spatial acceleration equations.
	factor := (1.0 - 3.0*rs*x_dot_v*x_dot_v/(r_cubed)) / r_cubed
	term_x := state.P_x*factor - 2.0*x_dot_v*state.V_x/r_cubed
	term_y := state.P_y*factor - 2.0*x_dot_v*state.V_y/r_cubed
	term_z := state.P_z*factor - 2.0*x_dot_v*state.V_z/r_cubed

	accel_x := -rs * term_x
	accel_y := -rs * term_y
	accel_z := -rs * term_z

	return accel_t, accel_x, accel_y, accel_z
}

// RK4 helper structs and functions remain the same, but now operate on Cartesian components.
type RK4Derivative struct {
	Accel_t, Accel_x, Accel_y, Accel_z float64
	Vel_t, Vel_x, Vel_y, Vel_z         float64
}

func evaluateRK4(initialState *RayState, dt float64, deriv *RK4Derivative) *RayState {
	return &RayState{
		P_t: initialState.P_t + deriv.Vel_t*dt,
		P_x: initialState.P_x + deriv.Vel_x*dt,
		P_y: initialState.P_y + deriv.Vel_y*dt,
		P_z: initialState.P_z + deriv.Vel_z*dt,
		V_t: initialState.V_t + deriv.Accel_t*dt,
		V_x: initialState.V_x + deriv.Accel_x*dt,
		V_y: initialState.V_y + deriv.Accel_y*dt,
		V_z: initialState.V_z + deriv.Accel_z*dt,
	}
}

// Step advances the ray's state using the RK4 integrator. Its structure is identical,
// but it now works with the Cartesian state and acceleration.
func Step(state *RayState, stepSize float64, blackHole *BlackHole) *RayState {
	// Stage 1
	k1_at, k1_ax, k1_ay, k1_az := GetAcceleration(state, blackHole)
	k1 := RK4Derivative{k1_at, k1_ax, k1_ay, k1_az, state.V_t, state.V_x, state.V_y, state.V_z}

	// Stage 2
	midState1 := evaluateRK4(state, stepSize*0.5, &k1)
	k2_at, k2_ax, k2_ay, k2_az := GetAcceleration(midState1, blackHole)
	k2 := RK4Derivative{k2_at, k2_ax, k2_ay, k2_az, midState1.V_t, midState1.V_x, midState1.V_y, midState1.V_z}

	// Stage 3
	midState2 := evaluateRK4(state, stepSize*0.5, &k2)
	k3_at, k3_ax, k3_ay, k3_az := GetAcceleration(midState2, blackHole)
	k3 := RK4Derivative{k3_at, k3_ax, k3_ay, k3_az, midState2.V_t, midState2.V_x, midState2.V_y, midState2.V_z}

	// Stage 4
	endState := evaluateRK4(state, stepSize, &k3)
	k4_at, k4_ax, k4_ay, k4_az := GetAcceleration(endState, blackHole)
	k4 := RK4Derivative{k4_at, k4_ax, k4_ay, k4_az, endState.V_t, endState.V_x, endState.V_y, endState.V_z}

	// Combine
	newState := &RayState{}
	newState.P_t = state.P_t + (stepSize/6.0)*(k1.Vel_t+2*k2.Vel_t+2*k3.Vel_t+k4.Vel_t)
	newState.P_x = state.P_x + (stepSize/6.0)*(k1.Vel_x+2*k2.Vel_x+2*k3.Vel_x+k4.Vel_x)
	newState.P_y = state.P_y + (stepSize/6.0)*(k1.Vel_y+2*k2.Vel_y+2*k3.Vel_y+k4.Vel_y)
	newState.P_z = state.P_z + (stepSize/6.0)*(k1.Vel_z+2*k2.Vel_z+2*k3.Vel_z+k4.Vel_z)
	newState.V_t = state.V_t + (stepSize/6.0)*(k1.Accel_t+2*k2.Accel_t+2*k3.Accel_t+k4.Accel_t)
	newState.V_x = state.V_x + (stepSize/6.0)*(k1.Accel_x+2*k2.Accel_x+2*k3.Accel_x+k4.Accel_x)
	newState.V_y = state.V_y + (stepSize/6.0)*(k1.Accel_y+2*k2.Accel_y+2*k3.Accel_y+k4.Accel_y)
	newState.V_z = state.V_z + (stepSize/6.0)*(k1.Accel_z+2*k2.Accel_z+2*k3.Accel_z+k4.Accel_z)

	return newState
}
