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

	sunDirection := Vec3{X: -1, Y: 1, Z: -2}.Normalize()

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
				if rayPosition.Y <= -5 {
					ndotr := math.Min(1.0, math.Max(ambient, Vec3{Y: 1}.Dot(sunDirection)))
					sd := Vec3{X: -sunDirection.X, Y: sunDirection.Y, Z: -sunDirection.Z}
					if raymarch(rayPosition, sd, sphere1, sphere2, 1000) {
						ndotr = 0.1
					}

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
