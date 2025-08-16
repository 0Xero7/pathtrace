package main

import (
	"context"
	"image/color"
	"math"
	"math/rand"
)

func TraceRay(ctx context.Context, rayOrigin, rayDirection Vec3, stepSize float64, bvh *Box, maxSteps, bounces, scatterRays int, vertices, normals []Vec3, ambient float64, sunDirection Vec3) color.RGBA {
	done := false
	go func() {
		<-ctx.Done()
		done = true
	}()

	if bounces < 0 {
		return color.RGBA{}
	}

	rayPosition := rayOrigin
	for range maxSteps {
		if done {
			break
		}

		intersects, t, tri := bvh.CheckIntersection(rayPosition, rayDirection, stepSize, vertices)
		if intersects {
			intersection_point := rayPosition.Add(rayDirection.Scale(t))
			normal := InterpolateNormal(
				intersection_point,
				tri.A,
				tri.B,
				tri.C,
				normals[tri.Index],
				normals[tri.Index+1],
				normals[tri.Index+2],
			).Normalize()

			ndotr := math.Min(1.0, math.Max(0.0, normal.Dot(sunDirection)))
			shadow, _, _ := bvh.CheckIntersection(intersection_point.Add(normal.Scale(0.001)), sunDirection, stepSize, vertices)
			if shadow {
				ndotr = ambient
			}

			// GI Rays
			var indirectContribution Vec3
			for range scatterRays {
				var dir Vec3
				var up Vec3
				if math.Abs(normal.Y) < 0.9 {
					up = Vec3{X: 0, Y: 1, Z: 0} // Use Y if normal isn't mostly Y
				} else {
					up = Vec3{X: 1, Y: 0, Z: 0} // Use X if normal is mostly Y
				}
				// Build the basis
				tangent1 := normal.Cross(up).Normalize()
				tangent2 := normal.Cross(tangent1).Normalize() // Generate hemisphere direction (Z still means "along normal")
				theta := rand.Float64() * 2 * math.Pi
				phi := rand.Float64() * math.Pi / 2
				x := math.Cos(theta) * math.Sin(phi)
				y := math.Sin(theta) * math.Sin(phi)
				z := math.Cos(phi)

				// Transform to world space
				dir = tangent1.Scale(x).
					Add(tangent2.Scale(y)).
					Add(normal.Scale(z))

				contribution := TraceRay(ctx, intersection_point.Add(normal.Scale(0.001)), dir, stepSize, bvh, maxSteps, bounces-1, scatterRays, vertices, normals, ambient, sunDirection)
				albedo := 1.0
				lambert := dir.Dot(normal)

				r := float64(contribution.R)
				g := float64(contribution.G)
				b := float64(contribution.B)

				r = r * albedo * lambert
				g = g * albedo * lambert
				b = b * albedo * lambert

				indirectContribution = indirectContribution.Add(Vec3{X: r, Y: g, Z: b})
			}
			indirectContribution = indirectContribution.Scale(1.0 / float64(scatterRays))
			// indirectContribution = indirectContribution.Scale(al)

			final := Vec3{X: ndotr * 255, Y: ndotr * 255, Z: ndotr * 255}.Add(indirectContribution)
			final.X = math.Min(255, math.Max(0, final.X))
			final.Y = math.Min(255, math.Max(0, final.Y))
			final.Z = math.Min(255, math.Max(0, final.Z))
			return color.RGBA{
				R: uint8(final.X),
				G: uint8(final.Y),
				B: uint8(final.Z),
				A: 255,
			}

			// // === REFLECTION RAY ===
			// // Calculate reflection direction: R = D - 2(D·N)N
			// dotProduct := rayDirection.Dot(normal)
			// reflectionDirection := rayDirection.Sub(normal.Scale(2 * dotProduct)).Normalize()

			// // Cast reflection ray
			// reflectionOrigin := intersection_point.Add(normal.Scale(0.001))
			// reflectionColor := TraceRay(reflectionOrigin, reflectionDirection, stepSize, bvh, maxSteps, bounces-1, scatterRays, vertices, normals, ambient, sunDirection)

			// // === COMBINE DIRECT + REFLECTED LIGHT ===
			// reflectivity := 0.3 // How reflective the surface is (0.0 = matte, 1.0 = mirror)

			// finalR := uint8(float64(ndotr*255)*(1.0-reflectivity) + float64(reflectionColor.R)*reflectivity)
			// finalG := uint8(float64(ndotr*255)*(1.0-reflectivity) + float64(reflectionColor.G)*reflectivity)
			// finalB := uint8(float64(ndotr*255)*(1.0-reflectivity) + float64(reflectionColor.B)*reflectivity)

			// return color.RGBA{R: finalR, G: finalG, B: finalB, A: 255}
		}

		rayPosition = rayPosition.Add(rayDirection.Scale(stepSize))
	}

	return color.RGBA{}
}
