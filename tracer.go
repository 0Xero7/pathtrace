package main

import (
	"image/color"
	"math"
)

func TraceRay(rayOrigin, rayDirection Vec3, stepSize float64, bvh *Box, maxSteps, bounces int, vertices, normals []Vec3, ambient float64, sunDirection Vec3) color.RGBA {
	if bounces < 0 {
		return color.RGBA{}
	}

	rayPosition := rayOrigin
	for range maxSteps {
		// if done {
		// 	break
		// }

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
			)

			ndotr := math.Min(1.0, math.Max(ambient, normal.Dot(sunDirection)))
			shadow, _, _ := bvh.CheckIntersection(intersection_point.Add(normal.Scale(0.01)), sunDirection, stepSize, vertices)
			if shadow {
				ndotr = ambient
			}

			// === REFLECTION RAY ===
			// Calculate reflection direction: R = D - 2(D·N)N
			dotProduct := rayDirection.Dot(normal)
			reflectionDirection := rayDirection.Sub(normal.Scale(2 * dotProduct)).Normalize()

			// Cast reflection ray
			reflectionOrigin := intersection_point.Add(normal.Scale(0.001))
			reflectionColor := TraceRay(reflectionOrigin, reflectionDirection, stepSize, bvh, maxSteps, bounces-1, vertices, normals, ambient, sunDirection)

			// === COMBINE DIRECT + REFLECTED LIGHT ===
			reflectivity := 0.3 // How reflective the surface is (0.0 = matte, 1.0 = mirror)

			finalR := uint8(float64(ndotr*255)*(1.0-reflectivity) + float64(reflectionColor.R)*reflectivity)
			finalG := uint8(float64(ndotr*255)*(1.0-reflectivity) + float64(reflectionColor.G)*reflectivity)
			finalB := uint8(float64(ndotr*255)*(1.0-reflectivity) + float64(reflectionColor.B)*reflectivity)

			return color.RGBA{R: finalR, G: finalG, B: finalB, A: 255}
		}

		rayPosition = rayPosition.Add(rayDirection.Scale(stepSize))
	}

	return color.RGBA{}
}
