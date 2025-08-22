package main

import (
	"math"
	"math/rand"
	"sync/atomic"
)

var raysTraced atomic.Int64 = atomic.Int64{}
var recentRaysTraced atomic.Int64 = atomic.Int64{}

func TraceRay(ray Ray, stepSize float64, bvh *LinearBVH, maxSteps, bounces, scatterRays int, vnmu *VNMU, ambient float64, scene *Scene, bounceIndex int) Vec3 {
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
				diffuseComponent := HandleDiffuseMaterial(
					ray, stepSize, bvh, maxSteps, bounces, scatterRays, vnmu, ambient, scene, false, tri, intersection_point, rayPosition, normal, bounceIndex,
				)
				specularComponent := HandleReflectiveMaterial(ray.Origin, ray.Direction, stepSize, bvh, maxSteps, bounces, scatterRays, vnmu, ambient, scene, true, tri, intersection_point, rayPosition, normal, bounceIndex)
				return diffuseComponent.Add(specularComponent)
			default:
				return HandleDiffuseMaterial(
					ray, stepSize, bvh, maxSteps, bounces, scatterRays, vnmu, ambient, scene, true, tri, intersection_point, rayPosition, normal, bounceIndex,
				)
			}
		}

		rayPosition = rayPosition.Add(ray.Direction.Scale(stepSize))
	}

	return Vec3{}
	// return Vec3{X: 76, Y: 76, Z: 76}.Scale(1.0 / 255)
	// Hits the sky
	angle := ray.Direction.Dot(Vec3{Y: 1})
	if angle < 0 {
		return Vec3{X: 76, Y: 76, Z: 76}.Scale(1.0 / 255)
	}

	horizonColor := Vec3{X: 200, Y: 230, Z: 255}.Scale(1.0 / 255) // Light blue horizon
	zenithColor := Vec3{X: 50, Y: 120, Z: 255}.Scale(1.0 / 255)   // Deeper blue zenith

	skyColor := horizonColor.Scale(1.0 - angle).Add(zenithColor.Scale(angle))
	if bounceIndex == 0 {
		skyColor = skyColor.Scale(0.5) // Dim the sky color for indirect rays
	}
	return skyColor
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
) Vec3 {
	material := vnmu.Materials[tri.Index/3]
	emissiveColor := material.Emissive
	if bounceIndex > 0 { // This is an indirect ray
		if emissiveColor.R > 0 || emissiveColor.G > 0 || emissiveColor.B > 0 {
			return FromColor(emissiveColor)
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
		// diffuseColor.R = float32(math.Pow(float64(sampledColor.R)/255.0, 2.2))
		// diffuseColor.G = float32(math.Pow(float64(sampledColor.G)/255.0, 2.2))
		// diffuseColor.B = float32(math.Pow(float64(sampledColor.B)/255.0, 2.2))

		diffuseColor.R = float32(sampledColor.R) / 255.0
		diffuseColor.G = float32(sampledColor.G) / 255.0
		diffuseColor.B = float32(sampledColor.B) / 255.0
	}

	// Sample bump map if available
	if material.BumpImage != nil {
		bumpNormal := SampleBumpMap(material.BumpImage, x, y, 30.0)
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
		directContribution._Add(contribution)
	}
	// Now for emissive surfaces
	if len(vnmu.EmissiveTriangles) > 0 {
		rayOrigin := intersection_point.Add(normal.Scale(0.01))
		emissiveContribution := func() Vec3 {
			choice := rand.Intn(len(vnmu.EmissiveTriangles))
			i0 := vnmu.EmissiveTriangles[choice].VertexIndices[0]
			i1 := vnmu.EmissiveTriangles[choice].VertexIndices[1]
			i2 := vnmu.EmissiveTriangles[choice].VertexIndices[2]

			lightPoint := SampleTrianglePoint(vnmu.Vertices[i0], vnmu.Vertices[i1], vnmu.Vertices[i2])
			lightSurfaceNormal := InterpolateNormal(
				lightPoint, vnmu.Vertices[i0], vnmu.Vertices[i1], vnmu.Vertices[i2], vnmu.Normals[i0], vnmu.Normals[i1], vnmu.Normals[i2])
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
			sndorl := toLight.Dot(lightSurfaceNormal)
			if sndorl <= 0 {
				return Vec3{}
			}

			geometryTerm := ndotl * sndorl / (distance * distance)
			pdf := TriangleArea(vnmu.Vertices[i0], vnmu.Vertices[i1], vnmu.Vertices[i2])
			pdf = 1.0 / (pdf * float64(len(vnmu.EmissiveTriangles)))

			lightMaterial := FromColor(vnmu.Materials[vnmu.EmissiveTriangles[choice].MaterialIndex].Emissive).Scale(5) //) vnmu.Materials[vnmu.EmissiveTriangles[choice*3]]
			// lightEmission := FromColor(lightMaterial.Emissive)
			lightEmission := lightMaterial
			brdf := albedo.Scale(1.0 / math.Pi)

			return lightEmission.ComponentMul(brdf).Scale(geometryTerm / pdf)
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
			u1 := rand.Float64()
			u2 := rand.Float64()

			r := math.Sqrt(u1)
			theta := 2 * math.Pi * u2

			x := r * math.Cos(theta)
			y := r * math.Sin(theta)
			z := math.Sqrt(math.Max(0.0, 1.0-u1)) // This equals cos(phi)

			dir = tangent1.Scale(x)
			dir._Add(tangent2.Scale(y))
			dir._Add(normal.Scale(z))
			dir._Normalize()

			ray := NewRay(intersection_point.Add(normal.Scale(0.001)), dir)
			contribution := TraceRay(ray, stepSize, bvh, maxSteps, bounces-1, scatterRays, vnmu, ambient, scene, bounceIndex+1)
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
	return final
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
