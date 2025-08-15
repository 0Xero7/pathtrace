package main

import "math"

func IntersectSegmentTriangle(origin, direction Vec3, stepSize float64, A, B, C Vec3) (bool, float64) {
	const EPSILON = 1e-9 // Increased precision for better accuracy

	// Normalize direction vector to ensure consistent distance calculations
	direction = direction.Normalize()

	// Find vectors for two edges sharing vertex A.
	edge1 := B.Sub(A)
	edge2 := C.Sub(A)

	// Step 1: Calculate the determinant.
	// This involves a vector triple product. If the determinant is near zero,
	// the ray is parallel to the plane of the triangle.
	pvec := direction.Cross(edge2)
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
	qvec := tvec.Cross(edge1)
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
	if t > EPSILON && t <= stepSize {
		// An intersection has been found.
		return true, t
	}

	// The intersection is on the infinite ray but not on the segment.
	return false, 0
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
