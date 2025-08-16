package main

import (
	"math"
)

type Box struct {
	X1, X2 float64
	Y1, Y2 float64
	Z1, Z2 float64

	IsLeaf   bool
	Trianges []int
	Children []Box
}

func triangleIntersectsBox(v1, v2, v3 Vec3, x1, x2, y1, y2, z1, z2 float64) bool {
	// Get triangle's bounding box
	minX := math.Min(math.Min(v1.X, v2.X), v3.X)
	maxX := math.Max(math.Max(v1.X, v2.X), v3.X)
	minY := math.Min(math.Min(v1.Y, v2.Y), v3.Y)
	maxY := math.Max(math.Max(v1.Y, v2.Y), v3.Y)
	minZ := math.Min(math.Min(v1.Z, v2.Z), v3.Z)
	maxZ := math.Max(math.Max(v1.Z, v2.Z), v3.Z)

	// Check if bounding boxes overlap
	return !(maxX < x1 || minX > x2 || maxY < y1 || minY > y2 || maxZ < z1 || minZ > z2)
}

func BuildBVH(vertices []Vec3, tris []int, x1, x2, y1, y2, z1, z2 float64, threshold int, depth int) *Box {
	filteredVertices := make(map[int]bool)
	for i, v := range vertices {
		if Between(v.X, x1, x2) && Between(v.Y, y1, y2) && Between(v.Z, z1, z2) {
			filteredVertices[i] = true
		}
	}

	box := new(Box)
	box.X1, box.X2, box.Y1, box.Y2, box.Z1, box.Z2 = x1, x2, y1, y2, z1, z2

	if len(filteredVertices) > threshold && depth > 0 {
		lengths := []float64{
			math.Abs(box.X1 - box.X2),
			math.Abs(box.Y1 - box.Y2),
			math.Abs(box.Z1 - box.Z2),
		}
		biggest := 0
		if lengths[1] > lengths[0] {
			biggest = 1
		}
		if lengths[2] > lengths[biggest] {
			biggest = 2
		}

		newLengths := make([]float64, 3)
		copy(newLengths, lengths)
		newLengths[biggest] /= 2.0
		var childA, childB *Box

		switch biggest {
		case 0: // Split along X
			midX := (x1 + x2) / 2.0
			childA = BuildBVH(vertices, tris, x1, midX, y1, y2, z1, z2, threshold, depth-1)
			childB = BuildBVH(vertices, tris, midX, x2, y1, y2, z1, z2, threshold, depth-1)
		case 1: // Split along Y
			midY := (y1 + y2) / 2.0
			childA = BuildBVH(vertices, tris, x1, x2, y1, midY, z1, z2, threshold, depth-1)
			childB = BuildBVH(vertices, tris, x1, x2, midY, y2, z1, z2, threshold, depth-1)
		case 2: // Split along Z
			midZ := (z1 + z2) / 2.0
			childA = BuildBVH(vertices, tris, x1, x2, y1, y2, z1, midZ, threshold, depth-1)
			childB = BuildBVH(vertices, tris, x1, x2, y1, y2, midZ, z2, threshold, depth-1)
		}

		box.Children = append(box.Children, *childA, *childB)
	} else {
		box.IsLeaf = true
		for i := 0; i < len(tris); i += 3 {
			a, b, c := tris[i], tris[i+1], tris[i+2]
			if filteredVertices[a] || filteredVertices[b] || filteredVertices[c] {
				box.Trianges = append(box.Trianges, a)
				box.Trianges = append(box.Trianges, b)
				box.Trianges = append(box.Trianges, c)

				// Maybe extend boxes too?
			}
		}
	}

	return box
}

func (box Box) intersectAABB(origin, direction Vec3, stepSize float64) bool {
	min := []float64{box.X1, box.Y1, box.Z1}
	max := []float64{box.X2, box.Y2, box.Z2}
	ray_origin := []float64{origin.X, origin.Y, origin.Z}
	dir := []float64{direction.X, direction.Y, direction.Z}

	tMin := 0.0
	tMax := stepSize
	eplison := 1e-9

	for i := range 3 {
		if math.Abs(dir[i]) < eplison {
			if ray_origin[i] < min[i] || ray_origin[i] > max[i] {
				return false
			}
		} else {
			t1 := (min[i] - ray_origin[i]) / dir[i]
			t2 := (max[i] - ray_origin[i]) / dir[i]

			if t1 > t2 {
				t1, t2 = t2, t1
			}

			// Update the overall intersection interval
			tMin = math.Max(tMin, t1)
			tMax = math.Min(tMax, t2)

			// Exit early if the interval is invalid
			if tMin > tMax {
				return false
			}
		}
	}

	return true
}

func (box Box) CheckIntersection(origin, direction Vec3, stepSize float64, vertices []Vec3) (bool, float64, int, int, int) {
	if !box.intersectAABB(origin, direction, stepSize) {
		return false, 0, 0, 0, 0
	}

	if box.IsLeaf {
		for i := 0; i < len(box.Trianges); i += 3 {
			a := vertices[box.Trianges[i]]
			b := vertices[box.Trianges[i+1]]
			c := vertices[box.Trianges[i+2]]

			intersects, t := IntersectSegmentTriangle(origin, direction, stepSize, a, b, c)
			if intersects {
				return true, t, box.Trianges[i], box.Trianges[i+1], box.Trianges[i+2]
			}
		}

		return false, 0, 0, 0, 0
	}

	for _, c := range box.Children {
		intersects, t, i, j, k := c.CheckIntersection(origin, direction, stepSize, vertices)
		if intersects {
			return true, t, i, j, k
		}
	}

	return false, 0, 0, 0, 0
}
