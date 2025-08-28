package main

import (
	"fmt"
	"math"
)

type BVHTriangle struct {
	A, B, C  Vec3
	Centroid Vec3
	X, Y, Z  int
	Index    int

	MinX, MaxX float64
	MinY, MaxY float64
	MinZ, MaxZ float64
}

type Ray struct {
	Origin    Vec3
	Direction Vec3
	// InverseDirection Vec3
}

func NewRay(origin, direction Vec3) Ray {
	// inverseDirection := Vec3{
	// 	X: 1.0 / direction.X,
	// 	Y: 1.0 / direction.Y,
	// 	Z: 1.0 / direction.Z,
	// }
	return Ray{
		Origin:    origin,
		Direction: direction,
		// InverseDirection: inverseDirection,
	}
}

type FauxBox struct {
	X1, X2 float64
	Y1, Y2 float64
	Z1, Z2 float64

	TriangleCount int
}

func (b *FauxBox) Grow(tri *BVHTriangle) {
	if b.TriangleCount == 0 {
		b.X1 = tri.MinX
		b.X2 = tri.MaxX
		b.Y1 = tri.MinY
		b.Y2 = tri.MaxY
		b.Z1 = tri.MinZ
		b.Z2 = tri.MaxZ
	} else {
		b.X1 = min(b.X1, tri.MinX)
		b.X2 = max(b.X2, tri.MaxX)
		b.Y1 = min(b.Y1, tri.MinY)
		b.Y2 = max(b.Y2, tri.MaxY)
		b.Z1 = min(b.Z1, tri.MinZ)
		b.Z2 = max(b.Z2, tri.MaxZ)
	}
	b.TriangleCount += 1
}
func (b *FauxBox) Area() float64 {
	dx := b.X2 - b.X1
	dy := b.Y2 - b.Y1
	dz := b.Z2 - b.Z1
	return 2.0 * (dx*dy + dy*dz + dx*dz)
	// return math.Abs(b.X2-b.X1) * math.Abs(b.Y2-b.Y1) * math.Abs(b.Z2-b.Z1)
}

type Box struct {
	X1, X2 float64
	Y1, Y2 float64
	Z1, Z2 float64

	IsLeaf   bool
	Trianges []*BVHTriangle
	Children []*Box
}

func (b *Box) Grow(tri *BVHTriangle) {
	if len(b.Trianges) == 0 {
		b.X1 = tri.MinX
		b.X2 = tri.MaxX
		b.Y1 = tri.MinY
		b.Y2 = tri.MaxY
		b.Z1 = tri.MinZ
		b.Z2 = tri.MaxZ
	} else {
		b.X1 = min(b.X1, tri.MinX)
		b.X2 = max(b.X2, tri.MaxX)
		b.Y1 = min(b.Y1, tri.MinY)
		b.Y2 = max(b.Y2, tri.MaxY)
		b.Z1 = min(b.Z1, tri.MinZ)
		b.Z2 = max(b.Z2, tri.MaxZ)
	}
	b.Trianges = append(b.Trianges, tri)
}

func (b *Box) Area() float64 {
	dx := b.X2 - b.X1
	dy := b.Y2 - b.Y1
	dz := b.Z2 - b.Z1
	return 2.0 * (dx*dy + dy*dz + dx*dz)
	// return math.Abs(b.X2-b.X1) * math.Abs(b.Y2-b.Y1) * math.Abs(b.Z2-b.Z1)
}

type BVHStats struct {
	MaxDepth, MaxTris, MinTris, TotalLeafs, TotalNodes, TotalTriangles int
}

func (b BVHStats) String() string {
	return fmt.Sprintf("MaxDepth: %d\nMinTris: %d\nAverage Tris: %.2f\nMaxTris: %d\nTotalLeafs: %d\nTotalNodes: %d\nTotalTriangles: %d\n", b.MaxDepth, b.MinTris, float64(b.TotalTriangles)/float64(b.TotalLeafs), b.MaxTris, b.TotalLeafs, b.TotalNodes, b.TotalTriangles)
}

func (box Box) GetStats(depth int) BVHStats {
	if box.IsLeaf {
		return BVHStats{
			MaxDepth:       depth,
			MinTris:        len(box.Trianges),
			MaxTris:        len(box.Trianges),
			TotalNodes:     1,
			TotalLeafs:     1,
			TotalTriangles: len(box.Trianges),
		}
	}

	temp := BVHStats{
		MaxDepth:       -1,
		MinTris:        math.MaxInt,
		MaxTris:        -1,
		TotalNodes:     0,
		TotalTriangles: 0,
	}
	for _, child := range box.Children {
		stat := child.GetStats(depth + 1)
		temp.TotalNodes += stat.TotalNodes
		temp.TotalTriangles += stat.TotalTriangles
		temp.MaxDepth = max(temp.MaxDepth, stat.MaxDepth)
		temp.TotalLeafs += stat.TotalLeafs
		temp.MinTris = min(temp.MinTris, stat.MinTris)
		temp.MaxTris = max(temp.MaxTris, stat.MaxTris)
	}
	return temp
}

func buildBVH(tris []*BVHTriangle, x1, x2, y1, y2, z1, z2 float64, threshold int, depth int, parentCost float64) *Box {
	box := new(Box)
	box.X1, box.X2, box.Y1, box.Y2, box.Z1, box.Z2 = x1, x2, y1, y2, z1, z2
	box.Trianges = make([]*BVHTriangle, 0, len(tris))

	if len(tris) <= threshold || depth <= 0 {
		box.IsLeaf = true
		box.Trianges = append(box.Trianges, tris...)
		// box.Trianges = append(box.Trianges, tris...)
		return box
	}

	starts := [3]float64{x1, y1, z1}
	ends := [3]float64{x2, y2, z2}

	bestCost := math.MaxFloat64
	bestAxis := -1
	bestSplitPoint := -1.0

	for axis := range 3 {
		l, r := starts[axis], ends[axis]
		n := 128

		for i := 0; i <= n; i += 1 {
			mid := l + (r-l)*float64(i)/float64(n)

			leftBox := FauxBox{}
			rightBox := FauxBox{}

			for _, tri := range tris {
				// if tri.MinX < x1 || tri.MaxX > x2 ||
				// 	tri.MinY < y1 || tri.MaxY > y2 ||
				// 	tri.MinZ < z1 || tri.MaxZ > z2 {
				// 	continue
				// }

				var centroid float64
				switch axis {
				case 0:
					centroid = tri.Centroid.X
				case 1:
					centroid = tri.Centroid.Y
				case 2:
					centroid = tri.Centroid.Z
				}

				if centroid < mid {
					leftBox.Grow(tri)
				} else {
					rightBox.Grow(tri)
				}
			}

			cost := 1.0/8 + float64(leftBox.TriangleCount)*leftBox.Area() + float64(rightBox.TriangleCount)*rightBox.Area()
			if cost < bestCost {
				bestCost = cost
				bestAxis = axis
				bestSplitPoint = mid
			}
		}
	}

	if bestCost >= parentCost {
		box.Trianges = make([]*BVHTriangle, 0, len(tris))
		box.Trianges = append(box.Trianges, tris...)
		box.IsLeaf = true
		return box
	}

	var childA, childB *Box
	bestLeftBox, bestRightBox := getSplit(tris, bestAxis, bestSplitPoint)
	childA = buildBVH(bestLeftBox.Trianges, bestLeftBox.X1, bestLeftBox.X2, bestLeftBox.Y1, bestLeftBox.Y2, bestLeftBox.Z1, bestLeftBox.Z2, threshold, depth-1, bestCost)
	childB = buildBVH(bestRightBox.Trianges, bestRightBox.X1, bestRightBox.X2, bestRightBox.Y1, bestRightBox.Y2, bestRightBox.Z1, bestRightBox.Z2, threshold, depth-1, bestCost)

	if childA != nil {
		box.Children = append(box.Children, childA)
	}
	if childB != nil {
		box.Children = append(box.Children, childB)
	}
	return box
}

func getSplit(tris []*BVHTriangle, axis int, mid float64) (*Box, *Box) {
	left := Box{}
	right := Box{}

	for _, tri := range tris {
		centroids := [3]float64{tri.Centroid.X, tri.Centroid.Y, tri.Centroid.Z}

		if centroids[axis] < mid {
			left.Grow(tri)
		} else {
			right.Grow(tri)
		}
	}

	return &left, &right
}

func BuildBVH(verts []Vec3, tris []int, x1, x2, y1, y2, z1, z2 float64, threshold int, depth int) *Box {
	triangles := make([]*BVHTriangle, 0)
	for i := 0; i < len(tris); i += 3 {
		a := verts[tris[i]]
		b := verts[tris[i+1]]
		c := verts[tris[i+2]]
		centroid := a.Add(b).Add(c).Scale(1.0 / 3.0)

		triangles = append(triangles, &BVHTriangle{
			Index:    i,
			A:        a,
			B:        b,
			C:        c,
			Centroid: centroid,
			X:        tris[i],
			Y:        tris[i+1],
			Z:        tris[i+2],
			MinX:     min(a.X, b.X, c.X),
			MaxX:     max(a.X, b.X, c.X),
			MinY:     min(a.Y, b.Y, c.Y),
			MaxY:     max(a.Y, b.Y, c.Y),
			MinZ:     min(a.Z, b.Z, c.Z),
			MaxZ:     max(a.Z, b.Z, c.Z),
		})
	}

	return buildBVH(triangles, x1, x2, y1, y2, z1, z2, threshold, depth, math.MaxFloat64)
}

func (box *Box) intersectAABB(ray *Ray, stepSize float64) float64 {
	inverseDirectionX := 1.0 / ray.Direction.X
	t1 := (box.X1 - ray.Origin.X) * inverseDirectionX
	t2 := (box.X2 - ray.Origin.X) * inverseDirectionX

	tMin, tMax := t1, t2
	if tMin > tMax {
		tMin, tMax = tMax, tMin
	}

	inverseDirectionY := 1.0 / ray.Direction.Y
	t1 = (box.Y1 - ray.Origin.Y) * inverseDirectionY
	t2 = (box.Y2 - ray.Origin.Y) * inverseDirectionY

	tMin = max(tMin, min(t1, t2))
	tMax = min(tMax, max(t1, t2))

	if tMin > tMax {
		return math.MaxFloat64
	}

	inverseDirectionZ := 1.0 / ray.Direction.Z
	t1 = (box.Z1 - ray.Origin.Z) * inverseDirectionZ
	t2 = (box.Z2 - ray.Origin.Z) * inverseDirectionZ

	tMin = max(tMin, min(t1, t2))
	tMax = min(tMax, max(t1, t2))

	if tMin > tMax || tMax < 0 {
		return math.MaxFloat64
	}

	// Clamp to stepSize
	if tMin > stepSize {
		return math.MaxFloat64
	}

	return max(0, tMin)
}

func (box *Box) CheckIntersection(ray *Ray, stepSize float64, vertices []Vec3, knownIntersects bool) (bool, float64, *BVHTriangle) {
	if !knownIntersects {
		if box.intersectAABB(ray, stepSize) == math.MaxFloat64 {
			return false, 0, nil
		}
	}

	if box.IsLeaf {
		closest := math.MaxFloat64
		var closestTriangle *BVHTriangle
		found := false

		for _, tri := range box.Trianges {
			intersects, t := IntersectSegmentTriangle(ray.Origin, ray.Direction, stepSize, tri.A, tri.B, tri.C)
			if intersects && t < closest && t > 0 { // Make sure t > 0 (in front of ray)
				closest = t
				closestTriangle = tri
				found = true
			}
		}

		if found {
			return true, closest, closestTriangle
		}
		return false, 0, nil
	}

	// Check children and return the closest intersection
	// One Child
	if len(box.Children) == 1 {
		return box.Children[0].CheckIntersection(ray, stepSize, vertices, false)
	}

	// Two children
	i, j := 0, 1
	closest := math.MaxFloat64
	distI := box.Children[0].intersectAABB(ray, stepSize)
	distJ := box.Children[1].intersectAABB(ray, stepSize)
	if distI > distJ {
		i, j = 1, 0
		distJ = distI
	}

	intersects, t, tri := box.Children[i].CheckIntersection(ray, stepSize, vertices, false)
	if intersects && t > 0 {
		closest = t
	}
	if distJ > closest || distJ == math.MaxFloat64 {
		return intersects, t, tri
	}

	intersects2, t2, tri2 := box.Children[j].CheckIntersection(ray, stepSize, vertices, true)
	if intersects2 && t2 > 0 && t2 < closest {
		return intersects2, t2, tri2
	}
	if !intersects && !intersects2 {
		return false, 0, nil
	}

	return intersects, t, tri
}
