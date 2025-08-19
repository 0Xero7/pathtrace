package main

import (
	"fmt"
	"image"
	"os"
)

func tile(val float64) float64 {
	f := val - float64(int(val))
	return f
}

func LoadObj(path string, scaleFactor float64) (*Mesh, *Decoder, error) {
	object, err := Decode(path, "")

	for _, m := range object.Warnings {
		println(m)
	}

	// Load images
	for _, mat := range object.Materials {
		imagList := []string{mat.MapBump, mat.MapKd}

		for j, texPath := range imagList {
			if texPath == "" {
				continue
			} else {
				file, err := os.Open(texPath)
				if err != nil {
					fmt.Println(err)
					os.Exit(1)
				}

				imag, _, err := image.Decode(file)
				if err != nil {
					fmt.Println("Error while decoding file")
					fmt.Println(err)
					os.Exit(1)
				}

				cachedImage := CacheImage(imag)
				images[texPath] = cachedImage
				if j == 0 {
					mat.BumpImage = &cachedImage
				} else {
					mat.DiffuseImage = &cachedImage
				}
			}
		}
	}

	if err != nil {
		return nil, nil, err
	}

	vertices := make([]Vec3, 0)
	tris := make([]int, 0)
	normals := make([]Vec3, 0)
	mats := make([]*Material, 0)
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
		mats = append(mats, object.Materials[face.Material])

		if len(object.Uvs) > 0 {
			for i := range 3 {
				uvIndex := face.Uvs[i]
				u := (float64(object.Uvs[uvIndex*2]))   // X coordinate
				v := (float64(object.Uvs[uvIndex*2+1])) // Y coordinate
				uvs = append(uvs, u, v)
			}
		}

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
