package main

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"math"
	"math/rand"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
)

func raymarch(origin, direction Vec3, sphere1, sphere2 Sphere, maxSteps int) bool {
	rayPosition := origin
	for range maxSteps {
		distanceFromSphere1 := rayPosition.Sub(sphere1.Position).Length()
		if distanceFromSphere1 <= sphere1.Radius {
			return true
		}

		distanceFromSphere2 := rayPosition.Sub(sphere2.Position).Length()
		if distanceFromSphere2 <= sphere2.Radius {
			return true
		}
		rayPosition = rayPosition.Add(direction.Scale(0.1))
	}
	return false
}

func main() {
	a := app.New()
	w := a.NewWindow("Path Tracer")

	width := 512
	height := 512

	ambient := 0.12
	maxSteps := 2
	stepSize := 100.0

	// Scene
	camera := Camera{
		Position:         Vec3{X: 0, Y: 4, Z: 7},
		Forward:          Vec3{X: 0, Y: 0, Z: 1},
		Right:            Vec3{X: 1, Y: 0, Z: 0},
		Up:               Vec3{X: 0, Y: -1, Z: 0},
		FrustrumDistance: 1,
	}
	pixelBuffer := make([][]Pixel, height)
	for i := range pixelBuffer {
		pixelBuffer[i] = make([]Pixel, width)
	}

	boxMesh, _ := LoadObj("C:\\Users\\smpsm\\OneDrive\\Documents\\2B.obj", 3)
	boxObj := Object{
		Position: Vec3{Z: 10},
		Mesh:     *boxMesh,
	}

	scene := []Object{
		boxObj,
	}

	sunDirection := Vec3{X: 0, Y: 1, Z: -1}.Normalize()

	// Set up the window
	w.Resize(fyne.NewSize(float32(width), float32(height)))
	w.SetFixedSize(true)
	w.CenterOnScreen()

	// Initial black image
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: 0, G: 0, B: 0, A: 255})
		}
	}

	dirty := true
	w.SetContent(canvas.NewImageFromImage(img))

	// Add keystroke handling
	w.Canvas().SetOnTypedKey(func(key *fyne.KeyEvent) {
		switch key.Name {
		case fyne.KeyW:
			camera.Position = camera.Position.Add(camera.Forward.Scale(0.1))
			dirty = true
		case fyne.KeyA:
			camera.Position = camera.Position.Add(camera.Right.Scale(-0.1))
			dirty = true
		case fyne.KeyS:
			camera.Position = camera.Position.Add(camera.Forward.Scale(-0.1))
			dirty = true
		case fyne.KeyD:
			camera.Position = camera.Position.Add(camera.Right.Scale(0.1))
			dirty = true
		case fyne.KeyQ:
			camera.Position = camera.Position.Add(camera.Up.Scale(0.1))
			dirty = true
		case fyne.KeyE:
			camera.Position = camera.Position.Add(camera.Up.Scale(-0.1))
			dirty = true

		case fyne.KeyUp:
			sunDirection.Z += 0.1
			dirty = true
		case fyne.KeyDown:
			sunDirection.Z -= 0.1
			dirty = true
		case fyne.KeyLeft:
			sunDirection.X -= 0.1
			dirty = true
		case fyne.KeyRight:
			sunDirection.X += 0.1
			dirty = true
		}
	})

	w.Show()

	vertices, tris, normals := DecomposeObjects(scene)
	// println(len(normals))

	fmt.Println("BVH Building...")
	bvh := BuildBVH(vertices, tris, -1000, 1000, -1000, 1000, -1000, 1000, 16, 48)
	fmt.Println("BVH Built!")

	calculate := func(ctx context.Context, targetImg *image.RGBA, imgMutex *sync.Mutex) {
		done := false
		go func() {
			<-ctx.Done()
			done = true
		}()

		for !done {
			rx := rand.Float64()
			ry := rand.Float64()
			pixelX := int(math.Round(rx * float64(width-1)))
			pixelY := int(math.Round(ry * float64(height-1)))

			// Bounds check
			if pixelX < 0 || pixelX >= width || pixelY < 0 || pixelY >= height {
				continue
			}

			px := (rx - 0.5) * 2
			py := (ry - 0.5) * 2
			rayOrigin := camera.Position.Add(camera.Forward.Scale(camera.FrustrumDistance)).Add(camera.Up.Scale(py)).Add(camera.Right.Scale(px))
			rayDirection := Vec3{
				X: rayOrigin.X - camera.Position.X,
				Y: rayOrigin.Y - camera.Position.Y,
				Z: rayOrigin.Z - camera.Position.Z,
			}.Normalize()

			rayPosition := rayOrigin
			for range maxSteps {
				if done {
					break
				}

				intersects, t, tri := bvh.CheckIntersection(rayPosition, rayDirection, stepSize, vertices)
				if intersects {
					intersection_point := rayPosition.Add(rayDirection.Scale(t))
					normal := InterpolateNormal(
						intersection_point,
						tri.A,
						tri.B,
						tri.C,
						normals[tri.Index],
						normals[tri.Index+1],
						normals[tri.Index+2],
					)

					ndotr := math.Min(1.0, math.Max(ambient, normal.Dot(sunDirection)))
					rayColor := color.RGBA{R: uint8(ndotr * 255), G: uint8(ndotr * 255), B: uint8(ndotr * 255), A: 255}

					// When a ray hits a pixel:
					pixel := &pixelBuffer[pixelY][pixelX]
					pixel.Lock.Lock()

					// Accumulate color
					pixel.R += float64(rayColor.R)
					pixel.G += float64(rayColor.G)
					pixel.B += float64(rayColor.B)
					pixel.SampleCount++

					// Calculate running average
					avgR := pixel.R / float64(pixel.SampleCount)
					avgG := pixel.G / float64(pixel.SampleCount)
					avgB := pixel.B / float64(pixel.SampleCount)
					pixel.Lock.Unlock()

					imgMutex.Lock()
					// Update display
					targetImg.Set(pixelX, pixelY, color.RGBA{
						R: uint8(avgR),
						G: uint8(avgG),
						B: uint8(avgB),
						A: 255,
					})
					imgMutex.Unlock()
					break
				}

				// min_t := stepSize + 1
				// for i := 0; i < len(tris); i += 3 {
				// 	intersects, t := IntersectSegmentTriangle(rayPosition, rayDirection, stepSize, vertices[tris[i]], vertices[tris[i+1]], vertices[tris[i+2]])
				// 	if !intersects || t > min_t {
				// 		continue
				// 	}

				// 	min_t = t

				// 	intersection_point := rayPosition.Add(rayDirection.Scale(t))
				// 	// A := vertices[tris[i]]
				// 	// B := vertices[tris[i+1]]
				// 	// C := vertices[tris[i+2]]
				// 	// edge1 := B.Sub(A)
				// 	// edge2 := C.Sub(A)
				// 	// normal := edge1.Cross(edge2).Normalize()

				// 	normal := InterpolateNormal(
				// 		intersection_point,
				// 		vertices[tris[i]],
				// 		vertices[tris[i+1]],
				// 		vertices[tris[i+2]],
				// 		normals[i],
				// 		normals[i+1],
				// 		normals[i+2],
				// 	)

				// 	ndotr := math.Min(1.0, math.Max(ambient, normal.Dot(sunDirection)))
				// 	imgMutex.Lock()
				// 	targetImg.Set(pixelX, pixelY, color.RGBA{R: uint8(ndotr * 255), G: uint8(ndotr * 255), B: uint8(ndotr * 255), A: 255})
				// 	imgMutex.Unlock()
				// 	// break
				// }

				rayPosition = rayPosition.Add(rayDirection.Scale(stepSize))
			}
		}
	}

	// Animation loop
	totalTime := 0.0

	go func() {
		ticker := time.NewTicker(16 * time.Millisecond) // ~60 FPS
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				deltaTime := 0.1
				totalTime += deltaTime

				if dirty {
					dirty = false
					// Create a new image for this frame
					img = image.NewRGBA(image.Rect(0, 0, width, height))

					// Clear to black
					for y := 0; y < height; y++ {
						for x := 0; x < width; x++ {
							img.Set(x, y, color.RGBA{R: 0, G: 0, B: 0, A: 255})
							pixelBuffer[x][y].Lock.Lock()
							pixelBuffer[x][y].R, pixelBuffer[x][y].G, pixelBuffer[x][y].B = 0, 0, 0
							pixelBuffer[x][y].SampleCount = 0
							pixelBuffer[x][y].Lock.Unlock()
						}
					}
				}

				// Render frame with multiple goroutines
				var imgMutex sync.Mutex
				ctx, cancel := context.WithTimeout(context.Background(), 128*time.Millisecond)

				for range 16 {
					go calculate(ctx, img, &imgMutex)
				}
				<-ctx.Done()
				cancel()

				// Update the display on the main UI thread
				fyne.Do(func() {
					newImage := canvas.NewImageFromImage(img)
					newImage.FillMode = canvas.ImageFillOriginal
					w.SetContent(newImage)
				})

			default:
				time.Sleep(time.Millisecond)
			}
		}
	}()

	// Run the app (this blocks until the window is closed)
	a.Run()
}
