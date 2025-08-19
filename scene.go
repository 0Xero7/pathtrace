package main

type Scene struct {
	Camera *Camera
	Meshes []*GameObject[any]
	Lights []*GameObject[Light]
	// AmbientLight AmbientLight
}
