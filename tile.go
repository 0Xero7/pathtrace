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

		var priority float64

		if p.SampleCount < 4 {
			priority = 1e9 // Keep this logic to handle initial sampling
		} else {
			// CONVERGENCE CHECK: Use a new contrast threshold.
			// This value will need tuning! It's on a different scale than variance.
			// A value between 0.01 and 0.1 is a good starting point.
			if p.SampleCount > 32 && p.Contrast < 0.01 {
				continue
			}

			// PRIORITY CALCULATION: Swap Variance for Contrast.
			priority = p.Contrast / math.Sqrt(float64(p.SampleCount))
		}

		// Selection logic remains the same...
		if pixel == nil || priority > maxPriority || (priority == maxPriority && p.SampleCount < pixel.SampleCount) {
			pixel = p
			maxPriority = priority
		}
	}
	return pixel
}

func (t *Tile) GetNoisiestPixel2(maxSamples int) *Pixel {
	var pixel *Pixel
	maxPriority := -1.0

	for _, p := range t.Pixels {
		var priority float64 = 0
		if p.SampleCount >= maxSamples {
			continue
		}
		// Skip converged pixels
		if p.SampleCount > 32 && p.Variance < 0.001 {
			continue
		}

		// Skip pixels with too few samples
		if p.SampleCount < 16 {
			// return p
			priority = 1e9 // High priority for undersampled pixels
		} else {
			// Weight by inverse sample count (prefer undersampled pixels)
			priority = p.Variance / math.Sqrt(float64(p.SampleCount))
		}

		if priority > maxPriority || (priority == maxPriority && p.SampleCount < pixel.SampleCount) || (priority == maxPriority && p.SampleCount == pixel.SampleCount && rand.Float64() < 0.5) {
			pixel = p
			maxPriority = priority
		}
	}
	return pixel
}
