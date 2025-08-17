package main

import (
	"math"
	"math/rand"
	"sync/atomic"

	"github.com/g3n/engine/loader/obj"
)

var raysTraced atomic.Int64 = atomic.Int64{}

func TraceRay(rayOrigin, rayDirection Vec3, stepSize float64, bvh *Box, maxSteps, bounces, scatterRays int, vertices, normals []Vec3, materials []*obj.Material, uvs []float64, ambient float64, sunDirection Vec3, indirectRay bool) Vec3 {
	rayPosition := rayOrigin
	for range maxSteps {
		intersects, t, tri := bvh.CheckIntersection(rayPosition, rayDirection, stepSize, vertices, false)
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

			materialType := materials[tri.Index/3].Illum
			switch materialType {
			case 3, 5, 7:
				diffuseComponent := HandleDiffuseMaterial(
					rayOrigin, rayDirection, stepSize, bvh, maxSteps, bounces, scatterRays, vertices, normals, materials, uvs, ambient, sunDirection, true, tri, intersection_point, rayPosition, normal,
				)
				specularComponent := HandleReflectiveMaterial(rayOrigin, rayDirection, stepSize, bvh, maxSteps, bounces, scatterRays, vertices, normals, materials, uvs, ambient, sunDirection, true, tri, intersection_point, rayPosition, normal)
				return diffuseComponent.Add(specularComponent)
			default:
				return HandleDiffuseMaterial(
					rayOrigin, rayDirection, stepSize, bvh, maxSteps, bounces, scatterRays, vertices, normals, materials, uvs, ambient, sunDirection, true, tri, intersection_point, rayPosition, normal,
				)
			}
		}

		rayPosition = rayPosition.Add(rayDirection.Scale(stepSize))
	}

	// return Vec3{X: 76, Y: 76, Z: 76}.Scale(1.0 / 255)
	// Hits the sky
	angle := rayDirection.Dot(Vec3{Y: 1})
	if angle < 0 {
		return Vec3{X: 76, Y: 76, Z: 76}.Scale(1.0 / 255)
	}

	horizonColor := Vec3{X: 200, Y: 230, Z: 255}.Scale(1.0 / 255) // Light blue horizon
	zenithColor := Vec3{X: 50, Y: 120, Z: 255}.Scale(1.0 / 255)   // Deeper blue zenith

	skyColor := horizonColor.Scale(1.0 - angle).Add(zenithColor.Scale(angle))
	return skyColor
}

func HandleDiffuseMaterial(
	rayOrigin, rayDirection Vec3,
	stepSize float64,
	bvh *Box,
	maxSteps, bounces, scatterRays int,
	vertices, normals []Vec3,
	materials []*obj.Material,
	uvs []float64,
	ambient float64,
	sunDirection Vec3,
	indirectRay bool,
	tri *BVHTriangle,
	intersection_point, rayPosition, normal Vec3,
) Vec3 {
	emissiveColor := materials[tri.Index/3].Emissive
	if indirectRay { // This is an indirect ray
		if emissiveColor.R > 0 || emissiveColor.G > 0 || emissiveColor.B > 0 {
			distance := intersection_point.Sub(rayPosition).Length()
			attenuation := 1.0 / (1.0 + distance*distance*0.01)
			lightIntensity := attenuation

			// Return emissive light directly for GI
			return Vec3{
				X: float64(emissiveColor.R) * lightIntensity,
				Y: float64(emissiveColor.G) * lightIntensity,
				Z: float64(emissiveColor.B) * lightIntensity,
			}
		}
	}

	// Get base diffuse color (albedo)
	diffuseColor := materials[tri.Index/3].Diffuse

	// Sample texture if available
	if materials[tri.Index/3].MapKd != "" {
		triangleIndex := tri.Index / 3
		baseUVIndex := triangleIndex * 6

		uv0_x, uv0_y := uvs[baseUVIndex], uvs[baseUVIndex+1]
		uv1_x, uv1_y := uvs[baseUVIndex+2], uvs[baseUVIndex+3]
		uv2_x, uv2_y := uvs[baseUVIndex+4], uvs[baseUVIndex+5]

		// Calculate barycentric coordinates
		v0 := tri.B.Sub(tri.A)
		v1 := tri.C.Sub(tri.A)
		v2 := intersection_point.Sub(tri.A)

		dot00 := v0.Dot(v0)
		dot01 := v0.Dot(v1)
		dot02 := v0.Dot(v2)
		dot11 := v1.Dot(v1)
		dot12 := v1.Dot(v2)

		invDenom := 1.0 / (dot00*dot11 - dot01*dot01)
		u := (dot11*dot02 - dot01*dot12) * invDenom
		v := (dot00*dot12 - dot01*dot02) * invDenom
		w := 1.0 - u - v

		x := w*uv0_x + u*uv1_x + v*uv2_x
		y := w*uv0_y + u*uv1_y + v*uv2_y

		sampledColor := Sample(materials[tri.Index/3].MapKd, x, y)

		// Convert from sRGB to linear space for lighting calculations
		diffuseColor.R = float32(math.Pow(float64(sampledColor.R)/255.0, 1.0))
		diffuseColor.G = float32(math.Pow(float64(sampledColor.G)/255.0, 1.0))
		diffuseColor.B = float32(math.Pow(float64(sampledColor.B)/255.0, 1.0))
	}

	// Create albedo vector
	albedo := Vec3{
		X: float64(diffuseColor.R),
		Y: float64(diffuseColor.G),
		Z: float64(diffuseColor.B),
	}

	// Calculate direct lighting
	ndotr := math.Min(1.0, math.Max(ambient, normal.Dot(sunDirection.Normalize())))
	shadow, _, _ := bvh.CheckIntersection(intersection_point.Add(normal.Scale(0.001)), sunDirection, stepSize, vertices, false)
	if shadow {
		ndotr = ambient
	}

	// Apply lighting to albedo (not as multiplication but as proper lighting)
	directLight := Vec3{X: ndotr, Y: ndotr, Z: ndotr}
	directContribution := albedo.ComponentMul(directLight)

	// GI Rays
	var indirectContribution Vec3
	if bounces > 0 {
		for range scatterRays {
			var dir Vec3
			var up Vec3
			if math.Abs(normal.Y) < 0.9 {
				up = Vec3{X: 0, Y: 1, Z: 0}
			} else {
				up = Vec3{X: 1, Y: 0, Z: 0}
			}

			tangent1 := normal.Cross(up).Normalize()
			tangent2 := normal.Cross(tangent1).Normalize()
			theta := rand.Float64() * 2 * math.Pi
			phi := rand.Float64() * math.Pi / 2
			x := math.Cos(theta) * math.Sin(phi)
			y := math.Sin(theta) * math.Sin(phi)
			z := math.Cos(phi)

			dir = tangent1.Scale(x).
				Add(tangent2.Scale(y)).
				Add(normal.Scale(z)).
				Normalize()

			contribution := TraceRay(intersection_point.Add(normal.Scale(0.001)), dir, stepSize, bvh, maxSteps, bounces-1, scatterRays, vertices, normals, materials, uvs, ambient, sunDirection, true)
			lambert := dir.Dot(normal)

			// Apply albedo to incoming light, not as multiplication
			lightContribution := contribution.ComponentMul(albedo).Scale(lambert)
			indirectContribution = indirectContribution.Add(lightContribution)
		}
		indirectContribution = indirectContribution.Scale(1.0 / float64(scatterRays))
	}

	// Emissive contribution (only add once)
	emissiveContribution := Vec3{
		X: float64(emissiveColor.R),
		Y: float64(emissiveColor.G),
		Z: float64(emissiveColor.B),
	}

	// Combine all lighting
	final := directContribution.Add(indirectContribution.Scale(1)).Add(emissiveContribution) // Scale down GI to prevent overbright

	raysTraced.Add(1)
	return final
}

func HandleDiffuseMaterial2(
	rayOrigin, rayDirection Vec3,
	stepSize float64,
	bvh *Box,
	maxSteps, bounces, scatterRays int,
	vertices, normals []Vec3,
	materials []*obj.Material,
	uvs []float64,
	ambient float64,
	sunDirection Vec3,
	indirectRay bool,
	tri *BVHTriangle,
	intersection_point, rayPosition, normal Vec3,
) Vec3 {
	emissiveColor := materials[tri.Index/3].Emissive
	if indirectRay { // This is an indirect ray
		if emissiveColor.R > 0 || emissiveColor.G > 0 || emissiveColor.B > 0 {
			distance := intersection_point.Sub(rayPosition).Length()
			attenuation := 1.0 / (1.0 + distance*distance*0.01)
			lightIntensity := attenuation

			// Return emissive light directly for GI
			return Vec3{
				X: float64(emissiveColor.R) * lightIntensity,
				Y: float64(emissiveColor.G) * lightIntensity,
				Z: float64(emissiveColor.B) * lightIntensity,
			}
		}
	}

	// Emissive surfaces glow regardless of lighting/shadows
	emissiveContribution := Vec3{
		X: float64(emissiveColor.R),
		Y: float64(emissiveColor.G),
		Z: float64(emissiveColor.B),
	}

	diffuseColor := materials[tri.Index/3].Diffuse

	if materials[tri.Index/3].MapKd != "" {
		triangleIndex := tri.Index / 3
		baseUVIndex := triangleIndex * 6

		uv0_x, uv0_y := uvs[baseUVIndex], uvs[baseUVIndex+1]
		uv1_x, uv1_y := uvs[baseUVIndex+2], uvs[baseUVIndex+3]
		uv2_x, uv2_y := uvs[baseUVIndex+4], uvs[baseUVIndex+5]

		// Calculate barycentric coordinates
		v0 := tri.B.Sub(tri.A)
		v1 := tri.C.Sub(tri.A)
		v2 := intersection_point.Sub(tri.A)

		dot00 := v0.Dot(v0)
		dot01 := v0.Dot(v1)
		dot02 := v0.Dot(v2)
		dot11 := v1.Dot(v1)
		dot12 := v1.Dot(v2)

		invDenom := 1.0 / (dot00*dot11 - dot01*dot01)
		u := (dot11*dot02 - dot01*dot12) * invDenom
		v := (dot00*dot12 - dot01*dot02) * invDenom
		w := 1.0 - u - v

		x := w*uv0_x + u*uv1_x + v*uv2_x
		y := w*uv0_y + u*uv1_y + v*uv2_y

		sampledColor := Sample(materials[tri.Index/3].MapKd, x, y)
		diffuseColor.R = float32(sampledColor.R) / 255.0
		diffuseColor.G = float32(sampledColor.G) / 255.0
		diffuseColor.B = float32(sampledColor.B) / 255.0
	}

	// Create albedo vector
	albedo := Vec3{
		X: float64(diffuseColor.R),
		Y: float64(diffuseColor.G),
		Z: float64(diffuseColor.B),
	}

	ndotr := math.Min(1.0, math.Max(ambient, normal.Dot(sunDirection.Normalize())))
	shadow, _, _ := bvh.CheckIntersection(intersection_point.Add(normal.Scale(0.001)), sunDirection, stepSize, vertices, false)
	if shadow {
		ndotr = ambient
	}

	// Apply lighting to albedo (not as multiplication but as proper lighting)
	directLight := Vec3{}.Ones().Scale(ndotr)
	directContribution := albedo.ComponentMul(directLight)

	final := Vec3{
		X: ndotr * float64(diffuseColor.R),
		Y: ndotr * float64(diffuseColor.G),
		Z: ndotr * float64(diffuseColor.B),
	}

	// GI Rays
	if bounces > 0 {
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
				Add(normal.Scale(z)).
				Normalize()

			contribution := TraceRay(intersection_point.Add(normal.Scale(0.001)), dir, stepSize, bvh, maxSteps, bounces-1, scatterRays, vertices, normals, materials, uvs, ambient, sunDirection, true)
			lambert := dir.Dot(normal)

			// Apply albedo to incoming light, not as multiplication
			lightContribution := contribution.ComponentMul(albedo).Scale(lambert)
			indirectContribution = indirectContribution.Add(lightContribution)

			// r := float64(contribution.X)
			// g := float64(contribution.Y)
			// b := float64(contribution.Z)

			// r = r * albedo * lambert
			// g = g * albedo * lambert
			// b = b * albedo * lambert

			// indirectContribution = indirectContribution.Add(Vec3{X: r, Y: g, Z: b})
		}
		indirectContribution = indirectContribution.Scale(1.0 / float64(scatterRays))
		final = directContribution.Add(indirectContribution.Scale(1)).Add(emissiveContribution)
	}

	raysTraced.Add(1)
	return final
}

func HandleReflectiveMaterial(
	rayOrigin, rayDirection Vec3,
	stepSize float64,
	bvh *Box,
	maxSteps, bounces, scatterRays int,
	vertices, normals []Vec3,
	materials []*obj.Material,
	uvs []float64,
	ambient float64,
	sunDirection Vec3,
	indirectRay bool,
	tri *BVHTriangle,
	intersection_point, rayPosition, normal Vec3,
) Vec3 {
	material := materials[tri.Index/3]

	// Convert shininess to roughness (higher shininess = lower roughness)
	roughness := 1.0 / (1.0 + material.Shininess/100.0)

	// Get specular color and intensity
	specularColor := Vec3{
		X: float64(material.Specular.R),
		Y: float64(material.Specular.G),
		Z: float64(material.Specular.B),
	}

	// Calculate perfect reflection direction
	dotProduct := rayDirection.Dot(normal)
	reflectionDirection := rayDirection.Sub(normal.Scale(2 * dotProduct)).Normalize()

	var reflectionContribution Vec3

	if roughness < 0.1 { // Very smooth - perfect reflection
		contribution := TraceRay(
			intersection_point.Add(normal.Scale(0.001)),
			reflectionDirection,
			stepSize, bvh, maxSteps, bounces-1, 1,
			vertices, normals, materials, uvs, ambient, sunDirection, true,
		)
		reflectionContribution = contribution

	} else { // Glossy - sample around reflection direction
		for range scatterRays {
			sampledDir := SampleGlossyReflection(reflectionDirection, normal, float64(roughness))
			contribution := TraceRay(
				intersection_point.Add(normal.Scale(0.001)),
				sampledDir,
				stepSize, bvh, maxSteps, bounces-1, scatterRays,
				vertices, normals, materials, uvs, ambient, sunDirection, true,
			)
			reflectionContribution = reflectionContribution.Add(contribution)
		}
		reflectionContribution = reflectionContribution.Scale(1.0 / float64(scatterRays))
	}

	// Mix based on Fresnel and specular color
	final := reflectionContribution.ComponentMul(specularColor)

	return final
}

func SampleGlossyReflection(reflectionDir, normal Vec3, roughness float64) Vec3 {
	// Build coordinate system around reflection direction
	var up Vec3
	if math.Abs(reflectionDir.Y) < 0.9 {
		up = Vec3{X: 0, Y: 1, Z: 0}
	} else {
		up = Vec3{X: 1, Y: 0, Z: 0}
	}

	tangent1 := reflectionDir.Cross(up).Normalize()
	tangent2 := reflectionDir.Cross(tangent1).Normalize()

	// Sample within cone - tighter cone for smoother materials
	alpha := roughness * roughness
	theta := rand.Float64() * 2 * math.Pi
	phi := math.Acos(math.Pow(rand.Float64(), 1.0/(alpha*4.0+1.0)))

	x := math.Cos(theta) * math.Sin(phi)
	y := math.Sin(theta) * math.Sin(phi)
	z := math.Cos(phi)

	return tangent1.Scale(x).Add(tangent2.Scale(y)).Add(reflectionDir.Scale(z)).Normalize()
}
