package main

import (
	"image"
	"math"
	"os"
)

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

// ------------------------------------------------------------

type ImageSkybox struct {
	img       *image.Image
	Intensity float64
}

func (s *ImageSkybox) isSkybox() {}

func NewImageSkybox(path string, intensity float64) *ImageSkybox {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}

	img, _, err := image.Decode(f)
	if err != nil {
		panic(err)
	}

	return &ImageSkybox{
		img:       &img,
		Intensity: intensity,
	}
}

func (s *ImageSkybox) Sample(direction Vec3) Vec3 {
	// Ensure the direction vector is normalized.
	dir := direction.Normalize()

	// --- Correct Spherical Coordinate Conversion for Y-up ---
	// Azimuthal angle (phi) is the angle in the XZ plane, used for the U (horizontal) coordinate.
	// We use atan2(Z, X) for a Y-up system where phi=0 is along the +X axis.
	phi := math.Atan2(dir.Z, dir.X)

	// Polar angle (theta) is the angle from the positive Y ("up") axis, used for the V (vertical) coordinate.
	theta := math.Acos(dir.Y)

	// --- Correct Normalization to [0, 1] UV coordinates ---
	// Map phi from [-PI, PI] to [0, 1]
	u := (phi + math.Pi) / (2 * math.Pi)
	// Map theta from [0, PI] to [0, 1]
	v := theta / math.Pi

	// Get image dimensions.
	bounds := (*s.img).Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Map UV coordinates to pixel coordinates.
	// We subtract a tiny epsilon from the max to prevent out-of-bounds errors.
	x := int(u * float64(width-1))
	y := int(v * float64(height-1))

	// --- Correct Color Conversion ---
	// The RGBA() method returns uint32 values in the range [0, 65535].
	// We must divide by 65535.0 to normalize them to [0, 1].
	r, g, b, _ := (*s.img).At(x, y).RGBA()
	return Vec3{
		X: float64(r) / 65535.0,
		Y: float64(g) / 65535.0,
		Z: float64(b) / 65535.0,
	}.Scale(s.Intensity)
}
