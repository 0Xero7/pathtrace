package main

type Light interface {
	isLight()
	Sample(ray Ray, normal Vec3, bvh *LinearBVH, stepSize float64, vertices []Vec3) (Vec3, float64)
	// Sample(origin, direction, normal Vec3) Vec3
}

// -------------------------------------------

type Sun struct {
	Color     Vec3
	Direction Vec3
	Intensity float64
}

func (s *Sun) isLight() {}
func (s *Sun) Sample(ray Ray, normal Vec3, bvh *LinearBVH, stepSize float64, vertices []Vec3) (Vec3, float64) {
	ndotr := ray.Direction.Dot(normal)
	if ndotr < 0 {
		return Vec3{}, 0.0
	}
	shadow, _, _ := bvh.CheckIntersection(ray, stepSize, vertices)
	if shadow {
		return Vec3{}, 0.0
	}
	return s.Color.Scale(ndotr * s.Intensity), ndotr
}

// -------------------------------------------
