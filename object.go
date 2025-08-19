package main

type GameObject[T any] struct {
	Position Vec3
	Mesh     *Mesh
	Object   T
}
