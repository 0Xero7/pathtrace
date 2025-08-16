package main

import "sync"

type Pixel struct {
	R, G, B     float64
	SampleCount int
	Lock        sync.Mutex
}

func DecomposeObjects(objects []Object) ([]Vec3, []int, []Vec3) {
	vertices := make([]Vec3, 0)
	tris := make([]int, 0)
	normals := make([]Vec3, 0)

	for _, object := range objects {
		for _, v := range object.Mesh.Vertices {
			vertices = append(vertices, v.Add(object.Position))
		}

		tris = append(tris, object.Mesh.Tris...)
		normals = append(normals, object.Mesh.Normals...)
	}

	return vertices, tris, normals
}
