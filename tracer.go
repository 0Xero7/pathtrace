package main

import (
	"fmt"
	"math"
	"math/rand"
	"sync/atomic"
)

var raysTraced atomic.Int64 = atomic.Int64{}
var recentRaysTraced atomic.Int64 = atomic.Int64{}

func TraceRay(ray Ray, stepSize float64, bvh *LinearBVH, maxSteps, bounces, scatterRays int, vnmu *VNMU, ambient float64, scene *Scene, bounceIndex int, lastSuraceNormal Vec3, isSpecular bool) Vec3 {
	rayPosition := ray.Origin
	for range maxSteps {
		intersects, t, tri := bvh.CheckIntersection(ray, stepSize)
		if intersects {
			intersection_point := rayPosition.Add(ray.Direction.Scale(t))
			normal := InterpolateNormal(
				intersection_point,
				tri.A,
				tri.B,
				tri.C,
				vnmu.Normals[tri.Index],
				vnmu.Normals[tri.Index+1],
				vnmu.Normals[tri.Index+2],
			).Normalize()

			material := vnmu.Materials[tri.Index/3]
			materialType := material.Illum

			switch materialType {
			case 3, 5, 7:
				diffuseComponent, isIndirectEmissive := HandleDiffuseMaterial(
					ray,
					stepSize,
					bvh,
					maxSteps,
					bounces,
					scatterRays,
					vnmu,
					ambient,
					scene,
					false,
					tri,
					intersection_point,
					rayPosition,
					normal,
					bounceIndex,
					lastSuraceNormal,
				)

				if isIndirectEmissive && !isSpecular {
					// Do MIS
					pdf_brdf := ray.Direction.Dot(lastSuraceNormal) / math.Pi

					triangle_area := TriangleArea(tri.A, tri.B, tri.C)
					pdf_NEE_area := 1.0 / (float64(len(vnmu.EmissiveTriangles)) * triangle_area)

					lightNormal := normal
					cosLight := max(0, ray.Direction.Dot(lightNormal))
					distance := intersection_point.Sub(ray.Origin).Length()
					pdf_NEE_solidAngle := pdf_NEE_area * distance * distance / cosLight

					// 4. Calculate MIS weight
					weight := MISWeight(pdf_brdf, pdf_NEE_solidAngle)
					diffuseComponent._Scale(weight)

					fmt.Printf("MIS: pdf_brdf=%.3f; pdf_nee=%.3f; weight=%.3f\n", pdf_brdf, pdf_NEE_solidAngle, weight)
				}

				specularComponent := HandleReflectiveMaterial(ray.Origin, ray.Direction, stepSize, bvh, maxSteps, bounces, scatterRays, vnmu, ambient, scene, true, tri, intersection_point, rayPosition, normal, bounceIndex)
				return diffuseComponent.Add(specularComponent)
			default:
				dc, isIndirectEmissive := HandleDiffuseMaterial(
					ray, stepSize, bvh, maxSteps, bounces, scatterRays, vnmu, ambient, scene, true, tri, intersection_point, rayPosition, normal, bounceIndex, lastSuraceNormal,
				)

				if isIndirectEmissive && !isSpecular {
					// Do MIS
					pdf_brdf := ray.Direction.Dot(lastSuraceNormal) / math.Pi

					triangle_area := TriangleArea(tri.A, tri.B, tri.C)
					// total_light_area := float64(len(vnmu.EmissiveTriangles)) * triangle_area
					// fmt.Printf("DEBUG: Triangle Area = %.4f, Total Light Area = %.4f\n", triangle_area, total_light_area)

					pdf_NEE_area := 1.0 / (float64(len(vnmu.EmissiveTriangles)) * triangle_area)

					lightNormal := normal
					cosLight := max(0, ray.Direction.Dot(lightNormal))
					distance := intersection_point.Sub(ray.Origin).Length()
					pdf_NEE_solidAngle := pdf_NEE_area * distance * distance / cosLight

					// 4. Calculate MIS weight
					weight := MISWeight(pdf_brdf, pdf_NEE_solidAngle)
					dc._Scale(weight)

					// fmt.Printf("MIS: pdf_brdf=%.3f; pdf_nee=%.3f; weight=%.3f\n", pdf_brdf, pdf_NEE_solidAngle, weight)
				}
				return dc
			}
		}

		rayPosition = rayPosition.Add(ray.Direction.Scale(stepSize))
	}

	if scene.Skybox == nil {
		return Vec3{}
	}
	return scene.Skybox.Sample(ray.Direction)
}

func HandleDiffuseMaterial(
	ray Ray,
	stepSize float64,
	bvh *LinearBVH,
	maxSteps, bounces, scatterRays int,
	vnmu *VNMU,
	ambient float64,
	scene *Scene,
	indirectRay bool,
	tri *BVHTriangle,
	intersection_point, rayPosition, normal Vec3,
	bounceIndex int,
	lastSurfaceNormal Vec3,
) (Vec3, bool) {
	material := vnmu.Materials[tri.Index/3]
	emissiveColor := material.Emissive
	if bounceIndex > 0 { // This is an indirect ray
		if emissiveColor.R > 0 || emissiveColor.G > 0 || emissiveColor.B > 0 {
			return FromColor(emissiveColor), true
		}
	}

	// Get base diffuse color (albedo)
	diffuseColor := material.Diffuse

	// Calculate UV coordinates
	var x, y float64
	if material.HasImage {
		triangleIndex := tri.Index / 3
		baseUVIndex := triangleIndex * 6

		uv0_x, uv0_y := vnmu.UVs[baseUVIndex], vnmu.UVs[baseUVIndex+1]
		uv1_x, uv1_y := vnmu.UVs[baseUVIndex+2], vnmu.UVs[baseUVIndex+3]
		uv2_x, uv2_y := vnmu.UVs[baseUVIndex+4], vnmu.UVs[baseUVIndex+5]

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

		x = (w*uv0_x + u*uv1_x + v*uv2_x)
		y = (w*uv0_y + u*uv1_y + v*uv2_y)
	}

	// Sample texture if available
	if material.DiffuseImage != nil {
		sampledColor := SampleDiffuseMap(material.DiffuseImage, x, y)

		// Convert from sRGB to linear space for lighting calculations
		diffuseColor.R = float32(math.Pow(float64(sampledColor.R)/255.0, 2.2))
		diffuseColor.G = float32(math.Pow(float64(sampledColor.G)/255.0, 2.2))
		diffuseColor.B = float32(math.Pow(float64(sampledColor.B)/255.0, 2.2))
	}

	// Sample bump map if available
	if material.BumpImage != nil {
		bumpNormal := SampleBumpMap(material.BumpImage, x, y, 1.0)
		normal = TransformNormalToWorldSpace(bumpNormal, normal, tri, intersection_point, vnmu.UVs).Normalize()
	}

	// Create albedo vector
	albedo := Vec3{
		X: float64(diffuseColor.R),
		Y: float64(diffuseColor.G),
		Z: float64(diffuseColor.B),
	}

	// Ambient contribution
	ambientContribution := albedo.Scale(ambient)

	// Calculate direct lighting
	var directContribution Vec3
	rayOrigin := intersection_point.Add(normal.Scale(0.001))

	// From Skybox
	if scene.Skybox != nil {
		for range 1 {
			randomNormal := GetCosineWeighedHemisphereSampling(normal)
			ray := Ray{
				Origin:    rayOrigin,
				Direction: randomNormal,
			}
			if !bvh.QuickCheckIntersection(ray, 100000.0) {
				directContribution.Add(scene.Skybox.Sample(randomNormal).Scale(1.0 / 1.0).ComponentMul(albedo))
			}
		}
	}

	// From lights
	for _, light := range scene.Lights {
		var lightDirection Vec3
		sun, isSun := light.Object.(*Sun)
		if isSun {
			lightDirection = sun.Direction
		} else {
			lightDirection = light.Position.Sub(rayOrigin).Normalize()
		}
		lightRay := Ray{
			Origin:    rayOrigin,
			Direction: lightDirection,
		}
		contribution := light.Object.Sample(lightRay, normal, bvh, stepSize, light.Position)
		// Apply lighting to albedo (not as multiplication but as proper lighting)
		directContribution._Add(albedo.ComponentMul(contribution))
	}
	// Now for emissive surfaces
	if len(vnmu.EmissiveTriangles) > 0 {
		rayOrigin := intersection_point.Add(normal.Scale(0.01))
		emissiveContribution := func() Vec3 {
			choice := rand.Intn(len(vnmu.EmissiveTriangles))
			i0 := vnmu.EmissiveTriangles[choice].VertexIndices[0]
			i1 := vnmu.EmissiveTriangles[choice].VertexIndices[1]
			i2 := vnmu.EmissiveTriangles[choice].VertexIndices[2]

			n0 := vnmu.EmissiveTriangles[choice].NormalIndices[0]
			n1 := vnmu.EmissiveTriangles[choice].NormalIndices[1]
			n2 := vnmu.EmissiveTriangles[choice].NormalIndices[2]

			lightPoint := SampleTrianglePoint(vnmu.Vertices[i0], vnmu.Vertices[i1], vnmu.Vertices[i2])
			lightSurfaceNormal := InterpolateNormal(
				lightPoint, vnmu.Vertices[i0], vnmu.Vertices[i1], vnmu.Vertices[i2], vnmu.Normals[n0], vnmu.Normals[n1], vnmu.Normals[n2])
			toLight := lightPoint.Sub(rayOrigin)
			distance := toLight.Length()
			toLight._Normalize()

			ndotl := toLight.Dot(normal)
			if ndotl <= 0 {
				return Vec3{}
			}

			shadowRay := Ray{
				Origin:    rayOrigin,
				Direction: toLight,
			}
			shadow := bvh.QuickCheckIntersection(shadowRay, distance-0.01)
			if shadow {
				return Vec3{}
			}
			sndorl := -toLight.Dot(lightSurfaceNormal)
			if sndorl <= 0 {
				return Vec3{}
			}

			geometryTerm := ndotl * sndorl / (distance * distance)
			pdf := TriangleArea(vnmu.Vertices[i0], vnmu.Vertices[i1], vnmu.Vertices[i2])
			pdf = 1.0 / (pdf * float64(len(vnmu.EmissiveTriangles)))

			pdf_brdf := normal.Dot(toLight) / math.Pi
			pdf_solidAngle := pdf * (distance * distance) / sndorl

			// Calculate MIS weight
			weight := MISWeight(pdf_solidAngle, pdf_brdf)

			lightMaterial := FromColor(vnmu.Materials[vnmu.EmissiveTriangles[choice].MaterialIndex].Emissive) //) vnmu.Materials[vnmu.EmissiveTriangles[choice*3]]
			// lightEmission := FromColor(lightMaterial.Emissive)
			lightEmission := lightMaterial
			brdf := albedo.Scale(1.0 / math.Pi)
			finalValue := lightEmission.ComponentMul(brdf).Scale(geometryTerm * weight / pdf)

			return finalValue
		}()
		directContribution._Add(emissiveContribution)
	}

	// GI Rays
	var indirectContribution Vec3
	if bounces > 0 {
		var dir Vec3
		var up Vec3
		if math.Abs(normal.Y) < 0.999 {
			up = Vec3{X: 0, Y: 1, Z: 0}
		} else {
			up = Vec3{X: 1, Y: 0, Z: 0}
		}

		tangent1 := normal.Cross(up)
		tangent1._Normalize()

		tangent2 := normal.Cross(tangent1)

		for range scatterRays {
			dir = GetCosineWeighedHemisphereSampling2(normal, tangent1, tangent2)

			ray := NewRay(intersection_point.Add(normal.Scale(0.001)), dir)
			contribution := TraceRay(ray, stepSize, bvh, maxSteps, bounces-1, scatterRays, vnmu, ambient, scene, bounceIndex+1, normal, false)
			// lambert := dir.Dot(normal)

			// Apply albedo to incoming light, not as multiplication
			contribution._ComponentMul(albedo)
			indirectContribution._Add(contribution)
		}
		indirectContribution = indirectContribution.Scale(1.0 / float64(scatterRays))
	}

	// // Emissive contribution (only add once)
	// emissiveContribution := Vec3{
	// 	X: float64(emissiveColor.R),
	// 	Y: float64(emissiveColor.G),
	// 	Z: float64(emissiveColor.B),
	// }

	// Combine all lighting
	final := directContribution.Clone()
	final._Add(ambientContribution)
	final._Add(indirectContribution.Scale(1))
	// final._Add(emissiveContribution) // Scale down GI to prevent overbright
	if bounceIndex == 0 && (emissiveColor.R > 0 || emissiveColor.G > 0 || emissiveColor.B > 0) {
		final._Add(FromColor(emissiveColor))
	}

	raysTraced.Add(1)
	return final, false
}

func HandleReflectiveMaterial(
	rayOrigin, rayDirection Vec3,
	stepSize float64,
	bvh *LinearBVH,
	maxSteps, bounces, scatterRays int,
	vnmu *VNMU,
	ambient float64,
	scene *Scene,
	indirectRay bool,
	tri *BVHTriangle,
	intersection_point, rayPosition, normal Vec3,
	bounceIndex int,
) Vec3 {
	material := vnmu.Materials[tri.Index/3]

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

	for range scatterRays {
		sampledDir := SampleGlossyReflection(reflectionDirection, normal, float64(roughness))
		ray := NewRay(intersection_point.Add(normal.Scale(0.001)), sampledDir)
		contribution := TraceRay(
			ray,
			stepSize, bvh, maxSteps, bounces-1, scatterRays,
			vnmu,
			ambient, scene, bounceIndex+1,
			normal,
			true,
		)
		reflectionContribution._Add(contribution)
	}
	reflectionContribution._Scale(1.0 / float64(scatterRays))

	// Mix based on Fresnel and specular color
	reflectionContribution._ComponentMul(specularColor)
	return reflectionContribution
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
	theta := rand.Float64() * 2 * math.Pi
	alpha := roughness * roughness
	u := rand.Float64()
	phi := math.Atan(alpha * math.Sqrt(u) / math.Sqrt(1.0-u))

	x := math.Cos(theta) * math.Sin(phi)
	y := math.Sin(theta) * math.Sin(phi)
	z := math.Cos(phi)

	return tangent1.Scale(x).Add(tangent2.Scale(y)).Add(reflectionDir.Scale(z)).Normalize()
}
