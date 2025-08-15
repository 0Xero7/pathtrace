package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"math/rand"
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
	w := a.NewWindow("Hello")

	width := 512
	height := 512

	ambient := 0.2

	maxSteps := 1000
	stepSize := 0.1
	sphere1 := Sphere{
		Position: Vec3{X: 0, Y: 0, Z: 20},
		Radius:   5,
	}
	sphere2 := Sphere{
		Position: Vec3{X: -5, Y: 0, Z: 20},
		Radius:   5,
	}
	camera := Camera{
		Position:         Vec3{X: 0, Y: 0, Z: 0},
		Forward:          Vec3{X: 0, Y: 0, Z: 1},
		Right:            Vec3{X: 1, Y: 0, Z: 0},
		Up:               Vec3{X: 0, Y: 1, Z: 0},
		FrustrumDistance: 1,
	}

	sunDirection := Vec3{X: -1, Y: 1, Z: 2}.Normalize()

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	w.SetContent(canvas.NewImageFromImage(img))
	w.Resize(fyne.NewSize(float32(width), float32(height)))
	w.Show()

	totalTime := 0.0

	for range 1000 {
		deltaTime := 0.05
		totalTime += deltaTime

		img = image.NewRGBA(image.Rect(0, 0, width, height))
		sunDirection.X = -math.Sin(totalTime)
		sphere1.Position.X = 10 * math.Sin(totalTime)
		sphere1.Position.Y = 10 * math.Cos(totalTime)

		sphere2.Position.X = -10 * math.Sin(totalTime)
		sphere2.Position.Y = -10 * math.Cos(totalTime)

		for range 1 {
			fyne.Do(func() {
				t := time.Now().UnixMilli()
				rays := 0
				for {
					dt := time.Now().UnixMilli()
					if dt-t > 64 {
						fmt.Printf("Calculated %d rays\n", rays)
						break
					}
					rays += 1

					rx := rand.Float64()
					ry := rand.Float64()
					pixelX := math.Round(rx * float64(width))
					pixelY := math.Round(ry * float64(height))

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
						distanceFromSphere1 := rayPosition.Sub(sphere1.Position).Length()
						if distanceFromSphere1 <= sphere1.Radius {
							normal := sphere1.Position.Sub(rayPosition).Normalize()
							ndotr := math.Min(1.0, math.Max(ambient, normal.Dot(sunDirection)))
							img.Set(int(pixelX), int(pixelY), color.RGBA{R: uint8(ndotr * 0), G: uint8(ndotr * 0), B: uint8(ndotr * 255), A: 255})
							break
						}

						distanceFromSphere2 := rayPosition.Sub(sphere2.Position).Length()
						if distanceFromSphere2 <= sphere2.Radius {
							normal := sphere2.Position.Sub(rayPosition).Normalize()
							ndotr := math.Min(1.0, math.Max(ambient, normal.Dot(sunDirection)))
							img.Set(int(pixelX), int(pixelY), color.RGBA{R: uint8(ndotr * 255), G: uint8(ndotr * 0), B: uint8(ndotr * 0), A: 255})
							break
						}

						rayPosition = rayPosition.Add(rayDirection.Scale(stepSize))
					}
				}

				w.SetContent(canvas.NewImageFromImage(img))
				w.Show()
				time.Sleep(time.Millisecond * 1)
			})
		}
	}
}
