package main

import (
	"math"

	"github.com/g3n/engine/loader/obj"
)

func LoadObj(path string, scaleFactor float64) (*Mesh, *obj.Decoder, error) {
	object, err := obj.Decode(path, "")
	if err != nil {
		return nil, nil, err
	}

	vertices := make([]Vec3, 0)
	tris := make([]int, 0)
	normals := make([]Vec3, 0)
	mats := make([]*obj.Material, 0)
	uvs := make([]float64, 0)

	object_normals := make([]Vec3, 0)
	for i := 0; i < len(object.Normals); i += 3 {
		object_normals = append(object_normals, Vec3{
			X: float64(object.Normals[i]),
			Y: float64(object.Normals[i+1]),
			Z: float64(object.Normals[i+2]),
		}.Normalize())
	}

	for i := 0; i < len(object.Vertices); i += 3 {
		vertices = append(vertices, Vec3{X: float64(object.Vertices[i]) * float64(scaleFactor), Y: float64(object.Vertices[i+1]) * float64(scaleFactor), Z: float64(object.Vertices[i+2]) * float64(scaleFactor)})
	}

	for _, face := range object.Objects[0].Faces {
		tris = append(tris, face.Vertices...)
		// fmt.Println(face.Uvs)
		mats = append(mats, object.Materials[face.Material])

		uvs = append(uvs, math.Mod(float64(object.Uvs[face.Uvs[0]]), 1.0))
		uvs = append(uvs, math.Mod(float64(object.Uvs[face.Uvs[0]+1]), 1.0))
		uvs = append(uvs, math.Mod(float64(object.Uvs[face.Uvs[1]]), 1.0))
		uvs = append(uvs, math.Mod(float64(object.Uvs[face.Uvs[1]+1]), 1.0))
		uvs = append(uvs, math.Mod(float64(object.Uvs[face.Uvs[2]]), 1.0))
		uvs = append(uvs, math.Mod(float64(object.Uvs[face.Uvs[2]+1]), 1.0))

		normals = append(normals, object_normals[face.Normals[0]])
		normals = append(normals, object_normals[face.Normals[1]])
		normals = append(normals, object_normals[face.Normals[2]])
	}

	return &Mesh{
		Vertices:  vertices,
		Tris:      tris,
		Normals:   normals,
		Materials: mats,
		UVs:       uvs,
	}, object, nil
}
