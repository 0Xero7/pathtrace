package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"sync"

	"github.com/g3n/engine/loader/obj"
)

type Pixel struct {
	R, G, B     float64
	SampleCount int
	Lock        sync.Mutex
}

var images map[string]*image.Image = map[string]*image.Image{}

func Sample(path string, x, y float64) color.RGBA {
	x = (math.Mod(float64(x), 1))
	y = (math.Mod(float64(y), 1))

	img, found := images[path]
	if !found {
		file, err := os.Open(path)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		imag, _, err := image.Decode(file)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		images[path] = &imag
		img = &imag
	}

	width := (*img).Bounds().Dx()
	height := (*img).Bounds().Dy()

	col := (*img).At(int(float64(width)*x), int(float64(height)*y))
	r, g, b, a := col.RGBA()
	return color.RGBA{
		R: uint8(r),
		G: uint8(g),
		B: uint8(b),
		A: uint8(a),
	}
}

func DecomposeObjects(objects []Object) ([]Vec3, []int, []Vec3, []*obj.Material, []float64) {
	vertices := make([]Vec3, 0)
	tris := make([]int, 0)
	normals := make([]Vec3, 0)
	materials := make([]*obj.Material, 0)
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

	return vertices, tris, normals, materials, uvs
}
