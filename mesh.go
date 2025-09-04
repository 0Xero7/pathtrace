package main

type Mesh struct {
	Vertices  []Vec3
	Tris      []int
	Normals   []Vec3
	Materials []*Material
	UVs       []float32
}
