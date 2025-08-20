package main

import (
	"math"
	"math/rand"
)

type Tile struct {
	X      uint32
	Y      uint32
	Width  uint32
	Height uint32

	Pixels []*Pixel
}

func (t *Tile) GetLeastSampledPixel(maxSamples int) *Pixel {
	// Find the pixel with the least samples
	leastSampledPixel := &Pixel{SampleCount: math.MaxInt64}
	for _, pixel := range t.Pixels {
		if pixel.SampleCount >= maxSamples {
			continue
		}

		if pixel.SampleCount < leastSampledPixel.SampleCount {
			leastSampledPixel = pixel
		}
	}
	return leastSampledPixel
}

func (t *Tile) GetNoisiestPixel(maxSamples int) *Pixel {
	var pixel *Pixel
	maxPriority := -1.0

	for _, p := range t.Pixels {
		if p.SampleCount >= maxSamples {
			continue
		}

		// Skip pixels with too few samples
		if p.SampleCount < 4 {
			return p
		}

		// Skip converged pixels
		if p.SampleCount > 32 && p.Variance < 0.001 {
			continue
		}

		// Weight by inverse sample count (prefer undersampled pixels)
		priority := p.Variance / math.Sqrt(float64(p.SampleCount))

		if priority > maxPriority || (priority == maxPriority && p.SampleCount < pixel.SampleCount) || (priority == maxPriority && p.SampleCount == pixel.SampleCount && rand.Float64() < 0.5) {
			pixel = p
			maxPriority = priority
		}
	}
	return pixel
}
