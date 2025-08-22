package main

import (
	"fmt"
	"math"
	"math/rand"
)

func Humanize[T int | int32 | int64 | float64 | float32](val T) string {
	if val >= 1e9 {
		return fmt.Sprintf("%.1fG", float64(val)/1e9)
	}
	if val >= 1e6 {
		return fmt.Sprintf("%.1fM", float64(val)/1e6)
	}
	if val >= 1e3 {
		return fmt.Sprintf("%.1fK", float64(val)/1e3)
	}
	return fmt.Sprintf("%.1f", float64(val))
}

func Between(val, l, r float64) bool {
	return l <= val && r >= val
}

func IntersectSegmentTriangle1(origin, direction Vec3, stepSize float64, A, B, C Vec3) (bool, float64) {
	const epsilon = 1e-6

	edge1 := B.Sub(A)
	edge2 := C.Sub(A)
	ray_cross_e2 := direction.Cross(edge2)
	det := edge1.Dot(ray_cross_e2)

	if det > -epsilon && det < epsilon {
		return false, 0
	}

	inv_det := 1.0 / det
	s := origin.Sub(A)
	u := inv_det * s.Dot(ray_cross_e2)

	if (u < 0 && math.Abs(u) > epsilon) || (u > 1 && math.Abs(u-1) > epsilon) {
		return false, 0
	}

	s_cross_e1 := s.Cross(edge1)
	v := inv_det * direction.Dot(s_cross_e1)

	if (v < 0 && math.Abs(v) > epsilon) || (u+v > 1 && math.Abs(u+v-1) > epsilon) {
		return false, 0
	}

	t := inv_det * edge2.Dot(s_cross_e1)
	if t > epsilon && t <= stepSize {
		return true, t
	}

	return false, 0
}

func IntersectSegmentTriangle(origin, direction Vec3, stepSize float64, A, B, C Vec3) (bool, float64) {
	const EPSILON = 1e-6 // Increased precision for better accuracy

	// Normalize direction vector to ensure consistent distance calculations
	// direction := dir

	// Find vectors for two edges sharing vertex A.
	edge1 := B.Sub(A)
	edge2 := C.Sub(A)

	// Step 1: Calculate the determinant.
	// This involves a vector triple product. If the determinant is near zero,
	// the ray is parallel to the plane of the triangle.
	// pvec := direction.Cross(edge2)
	pvec := Vec3{
		X: direction.Y*edge2.Z - direction.Z*edge2.Y,
		Y: direction.Z*edge2.X - direction.X*edge2.Z,
		Z: direction.X*edge2.Y - direction.Y*edge2.X,
	}
	determinant := edge1.Dot(pvec)

	// If the determinant is close to 0, the ray lies in the plane of the triangle or is parallel to it.
	if math.Abs(determinant) < EPSILON {
		return false, 0
	}
	invDeterminant := 1.0 / determinant

	// Step 2: Calculate the first barycentric coordinate (u).
	// This checks if the intersection point is between the C-A and A-B edges.
	tvec := origin.Sub(A)
	u := tvec.Dot(pvec) * invDeterminant

	// Check the u-bound with small tolerance for floating-point errors
	if u < -EPSILON || u > 1.0+EPSILON {
		return false, 0
	}

	// Step 3: Calculate the second barycentric coordinate (v).
	// This checks if the intersection point is between the A-B and B-C edges.
	// qvec := tvec.Cross(edge1)
	qvec := Vec3{
		X: tvec.Y*edge1.Z - tvec.Z*edge1.Y,
		Y: tvec.Z*edge1.X - tvec.X*edge1.Z,
		Z: tvec.X*edge1.Y - tvec.Y*edge1.X,
	}
	v := direction.Dot(qvec) * invDeterminant

	// Check the v-bound and the u+v bound with tolerance
	if v < -EPSILON || u+v > 1.0+EPSILON {
		return false, 0
	}

	// Step 4: Calculate t, the distance from the ray origin to the intersection point.
	t := edge2.Dot(qvec) * invDeterminant

	// Step 5: The final and crucial check for a LINE SEGMENT.
	// We confirm that the intersection point 't' lies within the segment's length.
	// It must be a forward intersection (t > EPSILON) and within the stepSize.
	if t <= EPSILON || t > stepSize {
		return false, 0
	}
	return true, t
}

func InterpolateNormal(p, a, b, c Vec3, nA, nB, nC Vec3) Vec3 {
	const EPSILON = 1e-9

	// Calculate the vectors that form the sides of the triangle from vertex A
	v0 := b.Sub(a)
	v1 := c.Sub(a)
	// Calculate the vector from vertex A to the point P
	v2 := p.Sub(a)

	// --- Calculate Barycentric Coordinates (u, v, w) ---
	// This technique uses the dot products of the edge vectors to find the weights.
	d00 := v0.Dot(v0)
	d01 := v0.Dot(v1)
	d11 := v1.Dot(v1)
	d20 := v2.Dot(v0)
	d21 := v2.Dot(v1)

	// Calculate the denominator, which is related to the area of the triangle
	denom := d00*d11 - d01*d01

	// Use epsilon comparison instead of exact equality for floating-point safety
	if math.Abs(denom) < EPSILON {
		// The triangle is degenerate (a line or a point), return the normal of the first vertex.
		return nA.Normalize()
	}

	// v is the weight for vertex B
	v := (d11*d20 - d01*d21) / denom
	// w is the weight for vertex C
	w := (d00*d21 - d01*d20) / denom
	// u is the weight for vertex A
	u := 1.0 - v - w

	// Clamp barycentric coordinates to handle floating-point precision issues
	u = math.Max(0.0, math.Min(1.0, u))
	v = math.Max(0.0, math.Min(1.0, v))
	w = math.Max(0.0, math.Min(1.0, w))

	// Renormalize to ensure they sum to 1.0
	sum := u + v + w
	if sum > EPSILON {
		u /= sum
		v /= sum
		w /= sum
	}

	// --- Interpolate the Normals ---
	// Ensure input normals are normalized
	nA = nA.Normalize()
	nB = nB.Normalize()
	nC = nC.Normalize()

	// Multiply each vertex normal by its corresponding barycentric weight.
	interpNA := nA.Scale(u)
	interpNB := nB.Scale(v)
	interpNC := nC.Scale(w)

	// Sum the weighted normals to get the final interpolated normal.
	interpolatedNormal := interpNA.Add(interpNB).Add(interpNC)

	// It's crucial to re-normalize the result to ensure it's a valid unit vector.
	return interpolatedNormal.Normalize()
}

func SampleTrianglePoint(A, B, C Vec3) Vec3 {
	u := rand.Float64()
	v := rand.Float64()

	if u+v > 1 {
		u = 1 - u
		v = 1 - v
	}

	w := 1 - u - v
	return A.Scale(w).Add(B.Scale(u)).Add(C.Scale(v))
}

func TriangleArea(A, B, C Vec3) float64 {
	// Calculate the area using the cross product
	AB := B.Sub(A)
	AC := C.Sub(A)
	return 0.5 * AB.Cross(AC).Length()
}

func Clamp01(val float64) float64 {
	return math.Max(0, math.Min(1, val))
}
