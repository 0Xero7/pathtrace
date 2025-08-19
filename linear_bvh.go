package main

import "math"

const INF = math.MaxFloat64

type LinearBVHNode struct {
	// Bounds               [6]float64
	MinBounds, MaxBounds Vec3

	IsLeaf bool

	TriangleOffset uint32
	TriangleCount  uint32

	SecondChildOffset uint32
}

func (l *LinearBVHNode) intersectAABB(ray Ray, stepSize float64) float64 {
	inverseDirection := ray.Direction.Inverse()
	leftBounds := l.MinBounds.Sub(ray.Origin).ComponentMul(inverseDirection)
	rightBounds := l.MaxBounds.Sub(ray.Origin).ComponentMul(inverseDirection)

	// inverseDirectionX := 1.0 / ray.Direction.X
	// Unrolled loop - no arrays!
	t1 := leftBounds.X  //(l.Bounds[0] - ray.Origin.X) * inverseDirection.X
	t2 := rightBounds.X // (l.Bounds[3] - ray.Origin.X) * inverseDirection.X
	if t1 > t2 {
		t1, t2 = t2, t1
	}

	tMin := t1
	tMax := t2
	// inverseDirectionY := 1.0 / ray.Direction.Y
	t1 = leftBounds.Y  //(l.Bounds[1] - ray.Origin.Y) * inverseDirection.Y
	t2 = rightBounds.Y //(l.Bounds[4] - ray.Origin.Y) * inverseDirection.Y
	if t1 > t2 {
		t1, t2 = t2, t1
	}
	tMin = max(tMin, t1)
	tMax = min(tMax, t2)

	// if tMin > tMax {
	// 	return maxFloat64
	// }

	// inverseDirectionZ := 1.0 / ray.Direction.Z
	t1 = leftBounds.Z  // (l.Bounds[2] - ray.Origin.Z) * inverseDirection.Z
	t2 = rightBounds.Z // (l.Bounds[5] - ray.Origin.Z) * inverseDirection.Z
	if t1 > t2 {
		t1, t2 = t2, t1
	}

	tMin = max(tMin, t1)
	tMax = min(tMax, t2)
	if tMin > min(tMax, stepSize) || tMax < 0 {
		return INF
	}

	// // Clamp to stepSize
	// if tMin > stepSize {
	// 	return INF
	// }

	return max(0, tMin)
}

type LinearBVH struct {
	Nodes     []LinearBVHNode
	Triangles []*BVHTriangle
}

func convert(root *Box, obj *LinearBVH) {
	node := new(LinearBVHNode)
	node.IsLeaf = root.IsLeaf
	// node.Bounds[0] = root.X1
	// node.Bounds[1] = root.Y1
	// node.Bounds[2] = root.Z1

	// node.Bounds[3] = root.X2
	// node.Bounds[4] = root.Y2
	// node.Bounds[5] = root.Z2

	node.MinBounds = Vec3{X: root.X1, Y: root.Y1, Z: root.Z1}
	node.MaxBounds = Vec3{X: root.X2, Y: root.Y2, Z: root.Z2}

	if root.IsLeaf {
		node.TriangleOffset = uint32(len(obj.Triangles))
		obj.Triangles = append(obj.Triangles, root.Trianges...)
		node.TriangleCount = uint32(len(root.Trianges))
	}

	ptr := len(obj.Nodes)
	obj.Nodes = append(obj.Nodes, *node)
	if !root.IsLeaf {
		convert(root.Children[0], obj)
		if len(root.Children) == 2 {
			obj.Nodes[ptr].SecondChildOffset = uint32(len(obj.Nodes))
			convert(root.Children[1], obj)
		}
	}
}

func ConstructLinearBVH(root *Box) *LinearBVH {
	node := new(LinearBVH)
	convert(root, node)
	return node
}

// ----------------------------------------------------------------------

func (box *LinearBVH) CheckIntersection(ray Ray, stepSize float64, vertices []Vec3) (bool, float64, *BVHTriangle) {
	stack := make([]uint32, 64)
	stack[0] = 0
	nptr := 1

	best_t := stepSize
	var best_tri *BVHTriangle = nil

	for nptr > 0 {
		ptr := uint32(stack[nptr-1])
		node := box.Nodes[ptr]
		nptr--

		if node.IsLeaf {
			for i := 0; i < int(node.TriangleCount); i++ {
				tri := box.Triangles[node.TriangleOffset+uint32(i)]
				intersects, t := IntersectSegmentTriangle(ray.Origin, ray.Direction, best_t, tri.A, tri.B, tri.C)
				if intersects && t < best_t && t > 0 { // Make sure t > 0 (in front of ray)
					best_t = t
					best_tri = tri
				}
			}
		} else {
			firstChildDistance := box.Nodes[ptr+1].intersectAABB(ray, best_t)
			if node.SecondChildOffset == 0 { // Only one child!
				if firstChildDistance < best_t {
					stack[nptr] = ptr + 1
					nptr++
				}
			} else { // Two children
				secondChildDistance := box.Nodes[node.SecondChildOffset].intersectAABB(ray, best_t)
				i, j := ptr+1, node.SecondChildOffset
				if firstChildDistance > secondChildDistance {
					i, j = j, i
					firstChildDistance, secondChildDistance = secondChildDistance, firstChildDistance
				}
				if firstChildDistance == INF {
					continue
				}

				if secondChildDistance < best_t {
					stack[nptr] = j
					nptr++
				}
				stack[nptr] = i
				nptr++
			}
		}
	}

	if best_tri == nil {
		return false, 0, nil
	}
	return true, best_t, best_tri
}
