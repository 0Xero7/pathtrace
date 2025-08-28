package main

import (
	"image"
	"image/color"
	"math"
	"sync"
	"unsafe"
)

type Pixel struct {
	X, Y        uint32
	R, G, B     float64
	SampleCount int
	Variance    float64 // Add this
	M2          Vec3    // For online variance calculation
	Mean        Vec3    // Running mean
	Lock        sync.Mutex
}

func (p *Pixel) AddSample(color Vec3) {
	// p.Lock.Lock()
	// defer p.Lock.Unlock()

	p.SampleCount++
	n := float64(p.SampleCount)

	// Update mean and variance
	delta := Vec3{
		X: color.X - p.Mean.X,
		Y: color.Y - p.Mean.Y,
		Z: color.Z - p.Mean.Z,
	}

	p.Mean.X += delta.X / n
	p.Mean.Y += delta.Y / n
	p.Mean.Z += delta.Z / n

	delta2 := Vec3{
		X: color.X - p.Mean.X,
		Y: color.Y - p.Mean.Y,
		Z: color.Z - p.Mean.Z,
	}

	p.M2.X += delta.X * delta2.X
	p.M2.Y += delta.Y * delta2.Y
	p.M2.Z += delta.Z * delta2.Z

	if p.SampleCount > 1 {
		// Luminance-weighted variance
		variance := (p.M2.X + p.M2.Y + p.M2.Z) / (3.0 * (n - 1))
		p.Variance = variance
	}

	p.R += color.X
	p.G += color.Y
	p.B += color.Z
}

type CachedImage struct {
	Pixels        uintptr
	pixelsSlice   []uint32 // Keep slice alive to prevent GC
	Width, Height int
}

func CacheImage(img image.Image) CachedImage {
	width := img.Bounds().Dx()
	height := img.Bounds().Dy()
	pixels := make([]uint32, width*height)

	for i := range height {
		for j := range width {
			r, g, b, a := img.At(j, i).RGBA()

			// Pack with correct endianness for your platform
			packed := uint32(r>>8) | (uint32(g>>8) << 8) |
				(uint32(b>>8) << 16) | (uint32(a>>8) << 24)

			pixels[i*width+j] = packed
		}
	}

	return CachedImage{
		Width:       width,
		Height:      height,
		Pixels:      uintptr(unsafe.Pointer(&pixels[0])),
		pixelsSlice: pixels,
	}
}

var images map[string]CachedImage = map[string]CachedImage{}

//go:nosplit
func SampleDiffuseMap(img *CachedImage, x, y float64) color.RGBA {
	px := int(x * float64(img.Width-1))
	py := int(y * float64(img.Height-1))

	offset := (py*(img.Width) + px) << 2

	packed := *(*uint32)(unsafe.Pointer(img.Pixels + uintptr(offset)))
	return color.RGBA{
		R: uint8(packed),
		G: uint8(packed >> 8),
		B: uint8(packed >> 16),
		A: uint8(packed >> 24),
	}
}

func SampleBumpMap(img *CachedImage, x, y float64, strength float64) Vec3 {
	// Wrap coordinates to handle texture edges
	// x = math.Mod(x, 1.0)
	// y = math.Mod(y, 1.0)

	// Calculate offset based on texture resolution for consistent sampling
	w, h := float64(img.Width), float64(img.Height)
	offsetX := 1.0 / w
	offsetY := 1.0 / h

	// Sample current pixel and neighbors
	center := SampleDiffuseMap(img, x, y)
	right := SampleDiffuseMap(img, math.Mod(x+offsetX, 1.0), y)
	up := SampleDiffuseMap(img, x, math.Mod(y+offsetY, 1.0))

	// Convert to height using luminance for better results
	centerHeight := (0.299*float64(center.R) + 0.587*float64(center.G) + 0.114*float64(center.B)) / 255.0
	rightHeight := (0.299*float64(right.R) + 0.587*float64(right.G) + 0.114*float64(right.B)) / 255.0
	upHeight := (0.299*float64(up.R) + 0.587*float64(up.G) + 0.114*float64(up.B)) / 255.0

	// Calculate gradient
	dx := (rightHeight - centerHeight) * strength
	dy := (upHeight - centerHeight) * strength

	// Create and normalize normal
	normal := Vec3{
		X: -dx,
		Y: -dy,
		Z: 1.0,
	}

	return normal.Normalize() // This is essential!
}

func SampleBumpMap2(img *CachedImage, x, y float64, strength float64) Vec3 {
	// x = (math.Mod(float64(x), 1))
	// y = (math.Mod(float64(y), 1))

	// Sample current pixel and neighbors for gradient calculation
	center := SampleDiffuseMap(img, x, y)
	right := SampleDiffuseMap(img, x+0.001, y) // Small offset
	up := SampleDiffuseMap(img, x, y+0.001)

	// Convert to height (use red channel or luminance)
	centerHeight := float64(center.R) / 255.0
	rightHeight := float64(right.R) / 255.0
	upHeight := float64(up.R) / 255.0

	// Calculate gradient (slope) to get normal
	dx := (rightHeight - centerHeight) * strength
	dy := (upHeight - centerHeight) * strength

	// Create normal from gradient
	normal := Vec3{
		X: -dx, // Negative because texture coords are flipped
		Y: -dy,
		Z: 1.0, // Always pointing "up" in tangent space
	}

	return normal //.Normalize()
}

func TransformNormalToWorldSpace(tangentNormal, worldNormal Vec3, tri *BVHTriangle, intersection_point Vec3, uvs []float64) Vec3 {
	// If no normal map data, just return the geometric normal
	if tangentNormal.X == 0 && tangentNormal.Y == 0 && tangentNormal.Z == 1 {
		return worldNormal
	}

	// Get triangle edges in world space
	edge1 := tri.B.Sub(tri.A) // A -> B
	edge2 := tri.C.Sub(tri.A) // A -> C

	// Get UV coordinates for this triangle
	triangleIndex := tri.Index / 3
	baseUVIndex := triangleIndex * 6

	// UV coordinates for each vertex
	uv0_x, uv0_y := uvs[baseUVIndex], uvs[baseUVIndex+1]   // Vertex A
	uv1_x, uv1_y := uvs[baseUVIndex+2], uvs[baseUVIndex+3] // Vertex B
	uv2_x, uv2_y := uvs[baseUVIndex+4], uvs[baseUVIndex+5] // Vertex C

	// Calculate UV deltas
	deltaUV1_x := uv1_x - uv0_x // UV delta A -> B
	deltaUV1_y := uv1_y - uv0_y
	deltaUV2_x := uv2_x - uv0_x // UV delta A -> C
	deltaUV2_y := uv2_y - uv0_y

	// Calculate determinant for the inverse matrix
	det := deltaUV1_x*deltaUV2_y - deltaUV2_x*deltaUV1_y

	// Handle degenerate UV coordinates
	if math.Abs(det) < 1e-6 {
		// Fallback: create arbitrary tangent space
		var up Vec3
		if math.Abs(worldNormal.Y) < 0.9 {
			up = Vec3{X: 0, Y: 1, Z: 0}
		} else {
			up = Vec3{X: 1, Y: 0, Z: 0}
		}

		tangent := worldNormal.Cross(up).Normalize()
		bitangent := worldNormal.Cross(tangent).Normalize()

		// Transform using fallback tangent space
		worldSpaceNormal := Vec3{
			X: tangentNormal.X*tangent.X + tangentNormal.Y*bitangent.X + tangentNormal.Z*worldNormal.X,
			Y: tangentNormal.X*tangent.Y + tangentNormal.Y*bitangent.Y + tangentNormal.Z*worldNormal.Y,
			Z: tangentNormal.X*tangent.Z + tangentNormal.Y*bitangent.Z + tangentNormal.Z*worldNormal.Z,
		}

		return worldSpaceNormal.Normalize()
	}

	// Calculate inverse determinant
	invDet := 1.0 / det

	// Calculate tangent and bitangent from UV coordinates
	tangent := Vec3{
		X: invDet * (deltaUV2_y*edge1.X - deltaUV1_y*edge2.X),
		Y: invDet * (deltaUV2_y*edge1.Y - deltaUV1_y*edge2.Y),
		Z: invDet * (deltaUV2_y*edge1.Z - deltaUV1_y*edge2.Z),
	}

	bitangent := Vec3{
		X: invDet * (-deltaUV2_x*edge1.X + deltaUV1_x*edge2.X),
		Y: invDet * (-deltaUV2_x*edge1.Y + deltaUV1_x*edge2.Y),
		Z: invDet * (-deltaUV2_x*edge1.Z + deltaUV1_x*edge2.Z),
	}

	// Orthogonalize using Gram-Schmidt process while preserving orientation
	// 1. Keep normal as-is (it's our reference)
	// 2. Make tangent perpendicular to normal
	tangent = tangent.Sub(worldNormal.Scale(tangent.Dot(worldNormal))).Normalize()

	// 3. Make bitangent perpendicular to both normal and tangent, but preserve handedness
	// First check the handedness (orientation) of our coordinate system
	calculatedBitangent := worldNormal.Cross(tangent)

	// Check if our UV-derived bitangent agrees with the calculated one
	if bitangent.Dot(calculatedBitangent) < 0 {
		// They point in opposite directions, flip the calculated one
		bitangent = calculatedBitangent.Scale(-1.0).Normalize()
	} else {
		bitangent = calculatedBitangent.Normalize()
	}

	// Transform the tangent space normal to world space
	worldSpaceNormal := Vec3{
		X: tangentNormal.X*tangent.X + tangentNormal.Y*bitangent.X + tangentNormal.Z*worldNormal.X,
		Y: tangentNormal.X*tangent.Y + tangentNormal.Y*bitangent.Y + tangentNormal.Z*worldNormal.Y,
		Z: tangentNormal.X*tangent.Z + tangentNormal.Y*bitangent.Z + tangentNormal.Z*worldNormal.Z,
	}

	return worldSpaceNormal.Normalize()
}

func DecomposeObjects(objects []*GameObject[any]) ([]Vec3, []int, []Vec3, []*Material, []float64, []EmissiveTriangle) {
	vertices := make([]Vec3, 0)
	tris := make([]int, 0)
	normals := make([]Vec3, 0)
	materials := make([]*Material, 0)
	uvs := make([]float64, 0)

	for _, object := range objects {
		for _, v := range object.Mesh.Vertices {
			vertices = append(vertices, v.Add(object.Position))
		}

		tris = append(tris, object.Mesh.Tris...)
		normals = append(normals, object.Mesh.Normals...)
		materials = append(materials, object.Mesh.Materials...)
		uvs = append(uvs, object.Mesh.UVs...)
	}

	emissives := make([]EmissiveTriangle, 0)
	for i := 0; i < len(tris); i += 3 {
		if materials[i/3].Emissive.R > 0 || materials[i/3].Emissive.G > 0 || materials[i/3].Emissive.B > 0 {
			tri := EmissiveTriangle{
				VertexIndices: [3]int{tris[i], tris[i+1], tris[i+2]},
				NormalIndices: [3]int{i, i + 1, i + 2},
				MaterialIndex: i / 3,
			}
			emissives = append(emissives, tri)
		}
	}

	return vertices, tris, normals, materials, uvs, emissives
}

func MISWeight(pdf1, pdf2 float64) float64 {
	if pdf1 <= 0 {
		return 0
	}
	if pdf2 <= 0 {
		return 1 // If other PDF is 0, this technique gets full weight
	}
	p1 := pdf1 * pdf1
	p2 := pdf2 * pdf2
	return p1 / (p1 + p2)
}
