package main

import (
	"math"
	"math/rand"
	"strings"
	"sync/atomic"
)

var raysTraced atomic.Int64 = atomic.Int64{}
var recentRaysTraced atomic.Int64 = atomic.Int64{}

func TraceRay(ray Ray, stepSize float64, bvh *LinearBVH, maxSteps, bounces, scatterRays int, vnmu *VNMU, ambient float64, scene *Scene, bounceIndex int, lastSuraceNormal Vec3, isSpecular bool, refractiveIndex *RefractiveIndexTracker, energy float64) Vec3 {
	if energy < 1e-2 || bounces < 0 {
		return Vec3{}
	}

	var refractionComponent Vec3
	var diffuseComponent Vec3
	var specularComponent Vec3
	var reflectiveComponent Vec3

	var rayState *RayState = nil

	if len(scene.BlackHoles) > 0 {
		rayState = GetInitialState(ray.Origin, ray.Direction, scene.BlackHoles[0])
	}

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
			if strings.HasPrefix(material.Name, "Glass") {
				ri := float64(material.Refraction)
				goingOut := false
				if normal.Dot(ray.Direction) > 0 {
					normal = normal.Scale(-1)
					goingOut = true
					ri = refractiveIndex.GetPreviousIndex()
				}

				refractedRayDir, tir := GetRefractedRay(ray.Direction, normal, refractiveIndex.GetCurrentIndex(), ri)
				if tir {
					refractionComponent = TraceRay(
						Ray{
							Origin:    intersection_point.Add(refractedRayDir.Scale(0.001)),
							Direction: refractedRayDir,
						},
						stepSize,
						bvh,
						maxSteps,
						bounces-1,
						scatterRays,
						vnmu,
						ambient,
						scene,
						bounceIndex+1,
						lastSuraceNormal,
						isSpecular,
						refractiveIndex,
						energy*0.95,
					).Scale(energy)
				} else {
					refractedRay := Ray{
						Origin:    intersection_point.Add(refractedRayDir.Scale(0.001)),
						Direction: refractedRayDir,
					}
					if goingOut {
						refractiveIndex.PopIndex()
					} else {
						refractiveIndex.UpdateIndex(ri)
					}
					refractionComponent = TraceRay(refractedRay, stepSize, bvh, maxSteps, bounces-1, scatterRays, vnmu, ambient, scene, bounceIndex, lastSuraceNormal, isSpecular, refractiveIndex, energy*0.95).Scale(energy)
				}
			}

			materialType := material.Illum
			reflectivity := (material.Specular.R + material.Specular.G + material.Specular.B) / 3.0
			switch materialType {
			default:
				switch {
				case reflectivity < 0.1:
					// Handle low reflectivity
					dc, isIndirectEmissive := HandleDiffuseMaterial(
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
						refractiveIndex,
						energy,
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
						dc._Scale(weight)
					}

					diffuseComponent = dc
				case reflectivity < 0.9:
					// Handle medium reflectivity
					isReflectiveRay := rand.Float64() < float64(reflectivity)
					if isReflectiveRay {
						reflectiveComponent = HandleReflectiveMaterial(ray.Origin, ray.Direction, stepSize, bvh, maxSteps, bounces, scatterRays, vnmu, ambient, scene, true, tri, intersection_point, rayPosition, normal, bounceIndex, refractiveIndex, energy).Add(refractionComponent)
					} else {
						dc, isIndirectEmissive := HandleDiffuseMaterial(
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
							refractiveIndex,
							energy,
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
							dc._Scale(weight)
						}

						diffuseComponent = dc
					}
				default:
					// os.Exit(1)
					// return Vec3{}
					// Handle high reflectivity\
					reflectiveComponent = HandleReflectiveMaterial(ray.Origin, ray.Direction, stepSize, bvh, maxSteps, bounces, scatterRays, vnmu, ambient, scene, true, tri, intersection_point, rayPosition, normal, bounceIndex, refractiveIndex, energy).Add(refractionComponent)
				}
				// case 4:
				// 	return refractiveComponent
			}

			// Combine all components
			diffuseScale := 1.0
			if refractionComponent.Length() > 0 {
				diffuseScale = 0.1
			}
			final := diffuseComponent.Scale(diffuseScale).Add(specularComponent).Add(reflectiveComponent).Add(refractionComponent)
			return final
		}

		if rayState == nil {
			rayPosition = rayPosition.Add(ray.Direction.Scale(stepSize))
			ray.Origin = rayPosition
		} else {
			// 1. Advance the ray one step in the Cartesian physics simulation.
			rayState = Step(rayState, stepSize, scene.BlackHoles[0])

			// 2. The state is already in relative Cartesian coordinates.
			//    Just create a Vec3 from the state's components.
			relativePos := Vec3{X: rayState.P_x, Y: rayState.P_y, Z: rayState.P_z}

			// 3. Translate to get the new world position.
			newRayPosition := relativePos.Add(scene.BlackHoles[0].Position)

			// 4. The velocity is also already Cartesian.
			newRayDirection := Vec3{X: rayState.V_x, Y: rayState.V_y, Z: rayState.V_z}.Normalize()

			// 5. Update the main ray object.
			ray.Origin = newRayPosition
			ray.Direction = newRayDirection
			rayPosition = newRayPosition

			// 6. Perform the termination check using Cartesian coordinates.
			//    It's more efficient to compare squared distances to avoid a sqrt().
			r_squared := relativePos.Dot(relativePos) // P_x² + P_y² + P_z²
			rs_squared := scene.BlackHoles[0].Rs * scene.BlackHoles[0].Rs

			if r_squared <= rs_squared {
				return Vec3{0, 0, 0} // Return black and STOP.
			}
		}
		// 1. Did the ray fall into the black hole?
	}

	if scene.Skybox == nil {
		return Vec3{}
	}
	return scene.Skybox.Sample(ray.Direction)
}

// func HandleRefractiveMaterial(
// 	ray Ray,
// 	stepSize float64,
// 	bvh *LinearBVH,
// 	maxSteps, bounces, scatterRays int,
// 	vnmu *VNMU,
// 	ambient float64,
// 	scene *Scene,
// 	indirectRay bool,
// 	tri *BVHTriangle,
// 	intersection_point, rayPosition, normal Vec3,
// 	bounceIndex int,
// 	lastSurfaceNormal Vec3,
// 	refractiveIndex, energy float64,
// ) (Vec3, bool) {
// 	material := vnmu.Materials[tri.Index/3]
// 	n2 := float64(material.Refraction)

// 	refractedRayDir, tir := GetRefractedRay(ray.Direction, normal, 1.0, n2)
// 	if tir { // Total Internal Reflection
// 		return TraceRay(Ray{
// 			Origin:    intersection_point.Add(refractedRayDir.Scale(0.001)),
// 			Direction: refractedRayDir,
// 		}, stepSize, bvh, maxSteps, bounces, scatterRays, vnmu, ambient, scene, bounceIndex, lastSurfaceNormal, true), false
// 	}

// 	var outgoingRay Ray
// 	var o_outgoingNormal Vec3
// 	var wentOut bool = false

// 	energy := 0.95
// 	// We are now inside the object
// 	for energy > 1e-4 {
// 		refractionPoint := intersection_point.Add(refractedRayDir.Scale(0.001))
// 		refractedRay := Ray{
// 			Origin:    refractionPoint,
// 			Direction: refractedRayDir,
// 		}
// 		intersects, t, intri := bvh.CheckIntersection(refractedRay, stepSize)
// 		if !intersects {
// 			return TraceRay(refractedRay, stepSize, bvh, maxSteps, bounces, scatterRays, vnmu, ambient, scene, bounceIndex, lastSurfaceNormal, true, refractiveIndex, energy), false
// 		}

// 		// Find the next point inside the object where the ray hits
// 		outgoingIntersectionPoint := refractionPoint.Add(refractedRayDir.Scale(t))
// 		outgoingNormal := InterpolateNormal(
// 			outgoingIntersectionPoint,
// 			intri.A,
// 			intri.B,
// 			intri.C,
// 			vnmu.Normals[intri.Index],
// 			vnmu.Normals[intri.Index+1],
// 			vnmu.Normals[intri.Index+2],
// 		).Normalize()

// 		outgoingRayDir, tir := GetRefractedRay(refractedRayDir, outgoingNormal.Scale(-1), n2, 1.0)
// 		if !tir {
// 			wentOut = true
// 			outgoingRay = Ray{
// 				Origin:    outgoingIntersectionPoint,
// 				Direction: outgoingRayDir,
// 			}
// 			o_outgoingNormal = outgoingNormal
// 			break
// 		}

// 		// Total internal reflection!
// 		outgoingRay = Ray{
// 			Origin:    outgoingIntersectionPoint,
// 			Direction: outgoingRayDir,
// 		}
// 		refractedRayDir = outgoingRayDir
// 		intersection_point = outgoingIntersectionPoint
// 		o_outgoingNormal = outgoingNormal
// 		energy = energy * 0.9 // Lose some energy on each internal reflection
// 	}

// 	if !wentOut {
// 		return Vec3{}, false
// 	}

// 	return TraceRay(outgoingRay, stepSize, bvh, maxSteps, bounces, scatterRays, vnmu, ambient, scene, bounceIndex, o_outgoingNormal, true).Scale(energy), false
// }

func HandleAmbientMaterial(
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

		x = tile(w*uv0_x + u*uv1_x + v*uv2_x)
		y = tile(w*uv0_y + u*uv1_y + v*uv2_y)
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

	// Calculate direct lighting
	var directContribution Vec3
	// rayOrigin := intersection_point.Add(normal.Scale(0.001))

	// From Skybox
	if scene.Skybox != nil {
		for range 1 {
			randomNormal := GetCosineWeighedHemisphereSampling(normal)
			// ray := Ray{
			// 	Origin:    rayOrigin,
			// 	Direction: randomNormal,
			// }
			// if !bvh.QuickCheckIntersection(ray, 100000.0) {
			directContribution._Add(scene.Skybox.Sample(randomNormal).ComponentMul(albedo))
			// }
		}
	}

	return directContribution, false
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
	ri *RefractiveIndexTracker,
	ni float64,
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

		x = tile(w*uv0_x + u*uv1_x + v*uv2_x)
		y = tile(w*uv0_y + u*uv1_y + v*uv2_y)
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
				directContribution = directContribution.Add(scene.Skybox.Sample(randomNormal).Scale(1.0 / 1.0).ComponentMul(albedo))
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
			contribution := TraceRay(ray, stepSize, bvh, maxSteps, bounces-1, scatterRays, vnmu, ambient, scene, bounceIndex+1, normal, false, ri, ni)
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
	ri *RefractiveIndexTracker,
	ni float64,
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
			ri, ni,
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
