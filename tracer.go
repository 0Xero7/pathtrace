package main

import (
	"math"
	"math/rand"
	"sync/atomic"
)

var raysTraced atomic.Int64 = atomic.Int64{}

func TraceRay(ray *Ray, stepSize float64, bvh *LinearBVH, maxSteps, bounces, scatterRays int, vertices, normals []Vec3, materials []*Material, uvs []float64, ambient float64, sunDirection Vec3, indirectRay bool) Vec3 {
	rayPosition := ray.Origin
	for range maxSteps {
		intersects, t, tri := bvh.CheckIntersection(ray, stepSize, vertices)
		if intersects {
			intersection_point := rayPosition.Add(ray.Direction.Scale(t))
			normal := InterpolateNormal(
				intersection_point,
				tri.A,
				tri.B,
				tri.C,
				normals[tri.Index],
				normals[tri.Index+1],
				normals[tri.Index+2],
			).Normalize()

			material := materials[tri.Index/3]
			materialType := material.Illum
			switch materialType {
			case 3, 5, 7:
				diffuseComponent := HandleDiffuseMaterial(
					ray, stepSize, bvh, maxSteps, bounces, scatterRays, vertices, normals, materials, uvs, ambient, sunDirection, true, tri, intersection_point, rayPosition, normal,
				)
				specularComponent := HandleReflectiveMaterial(ray.Origin, ray.Direction, stepSize, bvh, maxSteps, bounces, scatterRays, vertices, normals, materials, uvs, ambient, sunDirection, true, tri, intersection_point, rayPosition, normal)
				return diffuseComponent.Add(specularComponent)
			default:
				return HandleDiffuseMaterial(
					ray, stepSize, bvh, maxSteps, bounces, scatterRays, vertices, normals, materials, uvs, ambient, sunDirection, true, tri, intersection_point, rayPosition, normal,
				)
			}
		}

		rayPosition = rayPosition.Add(ray.Direction.Scale(stepSize))
	}

	// return Vec3{X: 76, Y: 76, Z: 76}.Scale(1.0 / 255)
	// Hits the sky
	angle := ray.Direction.Dot(Vec3{Y: 1})
	if angle < 0 {
		return Vec3{X: 76, Y: 76, Z: 76}.Scale(1.0 / 255)
	}

	horizonColor := Vec3{X: 200, Y: 230, Z: 255}.Scale(1.0 / 255) // Light blue horizon
	zenithColor := Vec3{X: 50, Y: 120, Z: 255}.Scale(1.0 / 255)   // Deeper blue zenith

	skyColor := horizonColor.Scale(1.0 - angle).Add(zenithColor.Scale(angle))
	if indirectRay {
		skyColor = skyColor.Scale(0.5) // Dim the sky color for indirect rays
	}
	return skyColor
}

func HandleDiffuseMaterial(
	ray *Ray,
	stepSize float64,
	bvh *LinearBVH,
	maxSteps, bounces, scatterRays int,
	vertices, normals []Vec3,
	materials []*Material,
	uvs []float64,
	ambient float64,
	sunDirection Vec3,
	indirectRay bool,
	tri *BVHTriangle,
	intersection_point, rayPosition, normal Vec3,
) Vec3 {
	material := materials[tri.Index/3]
	emissiveColor := material.Emissive
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
	diffuseColor := material.Diffuse

	// Calculate UV coordinates
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

	x := tile(w*uv0_x + u*uv1_x + v*uv2_x)
	y := tile(w*uv0_y + u*uv1_y + v*uv2_y)

	// Sample texture if available
	if material.MapKd != "" {
		sampledColor := SampleDiffuseMap(material.MapKd, x, y)

		// Convert from sRGB to linear space for lighting calculations
		// diffuseColor.R = float32(math.Pow(float64(sampledColor.R)/255.0, 2.2))
		// diffuseColor.G = float32(math.Pow(float64(sampledColor.G)/255.0, 2.2))
		// diffuseColor.B = float32(math.Pow(float64(sampledColor.B)/255.0, 2.2))

		diffuseColor.R = float32(sampledColor.R) / 255.0
		diffuseColor.G = float32(sampledColor.G) / 255.0
		diffuseColor.B = float32(sampledColor.B) / 255.0
	}

	// Sample bump map if available
	if material.MapBump != "" {
		bumpNormal := SampleBumpMap(material.MapBump, x, y, 3.0)
		normal = TransformNormalToWorldSpace(bumpNormal, normal, tri, intersection_point, uvs).Normalize()
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
	ndotr := math.Max(ambient, normal.Dot(sunDirection.Normalize()))
	if ndotr > 0 {
		ray := NewRay(intersection_point.Add(normal.Scale(0.001)), sunDirection)
		shadow, _, _ := bvh.CheckIntersection(ray, stepSize, vertices)
		if shadow {
			ndotr = 0.0
		}
	}

	// Apply lighting to albedo (not as multiplication but as proper lighting)
	directContribution := albedo.Scale(ndotr)

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

			tangent1 := normal.Cross(up)
			tangent1._Normalize()

			tangent2 := normal.Cross(tangent1)
			tangent2._Normalize()

			theta := rand.Float64() * 2 * math.Pi
			// phi := rand.Float64() * math.Pi / 2
			phi := math.Acos(math.Sqrt(rand.Float64()))
			x := math.Cos(theta) * math.Sin(phi)
			y := math.Sin(theta) * math.Sin(phi)
			z := math.Cos(phi)

			dir = tangent1.Scale(x)
			dir._Add(tangent2.Scale(y))
			dir._Add(normal.Scale(z))
			dir._Normalize()

			ray := NewRay(intersection_point.Add(normal.Scale(0.001)), dir)
			contribution := TraceRay(ray, stepSize, bvh, maxSteps, bounces-1, scatterRays, vertices, normals, materials, uvs, ambient, sunDirection, true)
			// lambert := dir.Dot(normal)

			// Apply albedo to incoming light, not as multiplication
			contribution._ComponentMul(albedo)
			contribution._Scale(1.0) // <--- this was pi
			indirectContribution._Add(contribution)
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
	final := directContribution.Clone()
	final._Add(ambientContribution)
	final._Add(indirectContribution.Scale(1))
	final._Add(emissiveContribution) // Scale down GI to prevent overbright

	raysTraced.Add(1)
	return final
}

func HandleReflectiveMaterial(
	rayOrigin, rayDirection Vec3,
	stepSize float64,
	bvh *LinearBVH,
	maxSteps, bounces, scatterRays int,
	vertices, normals []Vec3,
	materials []*Material,
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
		ray := NewRay(intersection_point.Add(normal.Scale(0.001)), reflectionDirection)
		contribution := TraceRay(
			ray,
			stepSize, bvh, maxSteps, bounces-1, 1,
			vertices, normals, materials, uvs, ambient, sunDirection, true,
		)
		reflectionContribution = contribution

	} else { // Glossy - sample around reflection direction
		for range scatterRays {
			sampledDir := SampleGlossyReflection(reflectionDirection, normal, float64(roughness))
			ray := NewRay(intersection_point.Add(normal.Scale(0.001)), sampledDir)
			contribution := TraceRay(
				ray,
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
