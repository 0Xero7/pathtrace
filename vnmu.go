package main

type EmissiveTriangle struct {
	VertexIndices, NormalIndices [3]int
	MaterialIndex                int
}

type VNMU struct {
	Vertices, Normals []Vec3
	Materials         []*Material
	UVs               []float64
	EmissiveTriangles []EmissiveTriangle
}
