package main

import (
	"context"
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

func do(width, height int, image *image.RGBA, flip bool) {
	for i := range width {
		for j := range height {
			if ((i/64+j/64)%2 == 0) == flip {
				image.Set(i, j, color.White)
			} else {
				image.Set(i, j, color.Black)
			}
		}
	}
}

func main() {
	a := app.New()
	w := a.NewWindow("Path Tracer")

	width := 512
	height := 512

	ambient := 0.12
	maxSteps := 1000
	stepSize := 0.1

	_ = Plane{
		Position: Vec3{X: 0, Y: -1, Z: 0},
		Normal:   Vec3{X: 0, Y: 1, Z: 0},
	}
	sphere1 := Sphere{
		Position: Vec3{X: 10, Y: 10, Z: 20},
		Radius:   2,
	}
	sphere2 := Sphere{
		Position: Vec3{X: -10, Y: 0, Z: 20},
		Radius:   5,
	}
	camera := Camera{
		Position:         Vec3{X: 0, Y: 10, Z: 0},
		Forward:          Vec3{X: 0, Y: -0.5, Z: 0.866},
		Right:            Vec3{X: 1, Y: 0, Z: 0},
		Up:               Vec3{X: 0, Y: -0.866, Z: -0.5},
		FrustrumDistance: 0.7,
	}

	sunDirection := Vec3{X: -1, Y: 1, Z: 2}.Normalize()

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

	w.SetContent(canvas.NewImageFromImage(img))
	w.Show()

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

				distanceFromSphere1 := rayPosition.Sub(sphere1.Position).Length()
				if distanceFromSphere1 <= sphere1.Radius {
					normal := sphere1.Position.Sub(rayPosition).Normalize()
					ndotr := math.Min(1.0, math.Max(ambient, normal.Dot(sunDirection)))

					imgMutex.Lock()
					targetImg.Set(pixelX, pixelY, color.RGBA{R: uint8(ndotr * 0), G: uint8(ndotr * 70), B: uint8(ndotr * 255), A: 255})
					imgMutex.Unlock()
					break
				}

				distanceFromSphere2 := rayPosition.Sub(sphere2.Position).Length()
				if distanceFromSphere2 <= sphere2.Radius {
					normal := sphere2.Position.Sub(rayPosition).Normalize()
					ndotr := math.Min(1.0, math.Max(ambient, normal.Dot(sunDirection)))

					imgMutex.Lock()
					targetImg.Set(pixelX, pixelY, color.RGBA{R: uint8(ndotr * 218), G: uint8(ndotr * 62), B: uint8(ndotr * 62), A: 255})
					imgMutex.Unlock()
					break
				}

				// Plane
				if rayPosition.Y <= 0 && math.Abs(rayPosition.X) <= 3 && math.Abs(rayPosition.Z) <= 3 {
					ndotr := math.Min(1.0, math.Max(ambient, Vec3{Y: 1}.Dot(sunDirection)))

					imgMutex.Lock()
					targetImg.Set(pixelX, pixelY, color.RGBA{R: uint8(ndotr * 255), G: uint8(ndotr * 255), B: uint8(ndotr * 255), A: 255})
					imgMutex.Unlock()
					break
				}

				rayPosition = rayPosition.Add(rayDirection.Scale(stepSize))
			}
		}
	}

	// Animation loop
	totalTime := 0.0
	dirty := true

	go func() {
		ticker := time.NewTicker(16 * time.Millisecond) // ~60 FPS
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				deltaTime := 0.1
				totalTime += deltaTime

				// Update scene parameters
				// sunDirection.X = -math.Sin(totalTime)
				// sphere1.Position.X = 10 * math.Sin(totalTime)
				// sphere1.Position.Y = 10 * math.Cos(totalTime)
				// sphere2.Position.X = -10 * math.Sin(totalTime)
				// sphere2.Position.Y = -10 * math.Cos(totalTime)
				// y := math.Pow(math.Sin(totalTime), 2)
				// sy := 2 * math.Sin(y)
				// cy := 2 * math.Cos(y)
				// camera.Forward = Vec3{0, sy, cy}
				// camera.Up = Vec3{0, -cy, sy}
				// dirty = true

				if dirty {
					dirty = false
					// Create a new image for this frame
					img = image.NewRGBA(image.Rect(0, 0, width, height))

					// Clear to black
					for y := 0; y < height; y++ {
						for x := 0; x < width; x++ {
							img.Set(x, y, color.RGBA{R: 0, G: 0, B: 0, A: 255})
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
