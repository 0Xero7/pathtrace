package main

type Light interface {
	isLight()
	Sample(ray Ray, normal Vec3, bvh *LinearBVH, stepSize float32, lightPos Vec3) Vec3
	// Sample(origin, direction, normal Vec3) Vec3
}

// -------------------------------------------

type Sun struct {
	Color     Vec3
	Direction Vec3
	Intensity float32
}

func (s *Sun) isLight() {}
func (s *Sun) Sample(ray Ray, normal Vec3, bvh *LinearBVH, stepSize float32, lightPos Vec3) Vec3 {
	ndotr := ray.Direction.Dot(normal)
	if ndotr < 0 {
		return Vec3{}
	}
	shadow := bvh.QuickCheckIntersection(ray, stepSize)
	if shadow {
		return Vec3{}
	}
	return s.Color.Scale(ndotr * s.Intensity)
}

// -------------------------------------------

type PointLight struct {
	Color     Vec3
	Intensity float32
}

func (s *PointLight) isLight() {}
func (s *PointLight) Sample(ray Ray, normal Vec3, bvh *LinearBVH, stepSize float32, lightPos Vec3) Vec3 {
	toLight := lightPos.Sub(ray.Origin)
	distance := toLight.Length()
	toLight._Normalize()

	ndotl := normal.Dot(toLight)
	if ndotl <= 0 {
		return Vec3{}
	}
	shadow := bvh.QuickCheckIntersection(ray, distance)
	if shadow {
		return Vec3{}
	}

	attenuation := 1.0 / (distance * distance)
	return s.Color.Scale(ndotl * s.Intensity * attenuation)
}

func (s *PointLight) Sample2(ray Ray, normal Vec3, bvh *LinearBVH, stepSize float32, lightPos Vec3) Vec3 {
	// 1. Create direction FROM hit point TO light
	toLight := lightPos.Sub(ray.Origin)
	distance := toLight.Length()
	lightDir := toLight.Scale(1.0 / distance) // Normalize

	// 2. Check if light is on the correct side of surface
	ndotl := normal.Dot(lightDir)
	if ndotl <= 0 {
		return Vec3{} // Light is behind surface
	}

	// 3. Shadow ray - from hit point toward light
	// IMPORTANT: Offset the origin slightly to avoid self-intersection
	shadowRay := Ray{
		Origin:    ray.Origin.Add(normal.Scale(0.001)), // Small epsilon offset
		Direction: lightDir,
	}

	// 4. Check occlusion up to the light distance (minus epsilon)
	if bvh.QuickCheckIntersection(shadowRay, distance-0.001) {
		return Vec3{} // Something blocks the light
	}

	// 5. Calculate contribution
	// Iₒᵤₜ = (Iₗᵢ𝓰ₕₜ * BRDF * cos(θ)) / distance²
	// For Lambertian BRDF = albedo/π, but that's handled elsewhere
	attenuation := s.Intensity / (distance * distance)

	return s.Color.Scale(ndotl * attenuation)
}
