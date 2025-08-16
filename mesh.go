package main

import "github.com/g3n/engine/loader/obj"

type Mesh struct {
	Vertices  []Vec3
	Tris      []int
	Normals   []Vec3
	Materials []*obj.Material
	UVs       []float64
}
