package main

type Skybox interface {
	isSkybox()
	Sample(direction Vec3) Vec3
}

// ------------------------------------------------------------

type SolidColorSkybox struct {
	Color Vec3
}

func (s *SolidColorSkybox) isSkybox() {}

func (s *SolidColorSkybox) Sample(direction Vec3) Vec3 {
	return s.Color
}

// ------------------------------------------------------------

type GradientSkybox struct {
	HorizonColor, ZenithColor, GroundColor Vec3
	Intensity                              float64
}

func (s *GradientSkybox) isSkybox() {}

func (s *GradientSkybox) Sample(direction Vec3) Vec3 {
	angle := direction.Dot(Vec3{Y: 1})
	if angle < 0 {
		return s.GroundColor
	}
	return s.HorizonColor.Scale(1.0 - angle).Add(s.ZenithColor.Scale(angle)).Scale(s.Intensity)
}
