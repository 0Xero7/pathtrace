// package main

// import (
// 	"fmt"
// 	"image"
// 	"image/color"
// 	"image/png"
// 	"log"
// 	"math"
// 	"math/rand"
// 	"os"
// 	"runtime/pprof"
// 	"sync"
// 	"sync/atomic"
// 	"time"

// 	"fyne.io/fyne/v2"
// 	"fyne.io/fyne/v2/app"
// 	"fyne.io/fyne/v2/canvas"
// 	"fyne.io/fyne/v2/container"
// )

// var threadCount = 16 // Number of goroutines to use for rendering

// func main() {
// 	a := app.New()
// 	w := a.NewWindow("Path Tracer")

// 	width := 768
// 	height := 768

// 	showStats := true

// 	bounces := 2
// 	scatterRays := 3
// 	ambient := 0.01
// 	maxSteps := 1
// 	stepSize := 1000.0

// 	// Scene
// 	rotY := 170.0 * 0.0174533
// 	rotX := 165.0 * 0.0174533

// 	// rotY := 155.0 * 0.0174533
// 	// rotX := 200.0 * 0.0174533
// 	camera := Camera{
// 		// Position: Vec3{X: -1, Y: 2, Z: 2}, // 2b2
// 		// Position: Vec3{X: 0, Y: 0, Z: -5},
// 		// Position: Vec3{X: 0, Y: 0.8, Z: -3.2},
// 		// Position: Vec3{X: -0.12574528163782742, Y: 2.2389967140962512, Z: 2.2364934252835065},
// 		Position: Vec3{X: -3.2, Y: 0.5, Z: 21}, // <--- sponza
// 		Forward:  Vec3{X: 0, Y: 0, Z: -1}.Normalize(),
// 		Right:    Vec3{X: 1, Y: 0, Z: 0},
// 		Up:       Vec3{X: 0, Y: -1, Z: 0},
// 		// Forward:          Vec3{X: -0.024214702328704055, Y: -0.5226872289306594, Z: -0.8521805612098416},
// 		// Right:            Vec3{X: -0.9995965384680866, Y: 0, Z: 0.028403525883580263},
// 		// Up:               Vec3{X: 0.014846160235948829, Y: -0.8525245220595057, Z: 0.5224763447405635},
// 		FrustrumDistance: 2,
// 	}
// 	camera.ApplyRotation(rotY, rotX)

// 	pixelBuffer := make([][]Pixel, height)
// 	for i := range pixelBuffer {
// 		pixelBuffer[i] = make([]Pixel, width)
// 	}

// 	// boxMesh, _ := LoadObj("C:\\Users\\smpsm\\OneDrive\\Documents\\Untitled.obj", 1)
// 	// boxMesh, _, _ := LoadObj("C:\\Users\\smpsm\\OneDrive\\Documents\\2B2.obj", 1)
// 	// boxMesh, _, _ := LoadObj("C:\\Users\\smpsm\\OneDrive\\Documents\\CornellSphere.obj", 1)
// 	// boxMesh, _, _ := LoadObj("C:\\Users\\smpsm\\OneDrive\\Documents\\cornell.obj", 1)
// 	// boxMesh, _, _ := LoadObj("C:\\Users\\smpsm\\OneDrive\\Documents\\Emissions.obj", 1)
// 	boxMesh, _, _ := LoadObj("C:\\Users\\smpsm\\OneDrive\\Documents\\sponza.obj", 1.5)
// 	// boxMesh, _, _ := LoadObj("C:\\Users\\smpsm\\OneDrive\\Documents\\2B.obj", 2)
// 	// boxMesh, _, _ := LoadObj("C:\\Users\\smpsm\\OneDrive\\Documents\\cube.obj", 1.0)
// 	boxObj := GameObject{
// 		Position: Vec3{Z: 0},
// 		Mesh:     *boxMesh,
// 	}

// 	scene := []GameObject{
// 		boxObj,
// 	}

// 	// sunDirection := Vec3{X: -0.2, Y: 0.854357657716761, Z: 0.3126145946300566}.Normalize() // <- 2B
// 	// sunDirection := Vec3{X: -0.2, Y: 0.854357657716761, Z: -5}.Normalize()
// 	sunDirection := Vec3{X: -0.1, Y: 1, Z: 0.1}.Normalize() // <- sponza
// 	// sunDirection := Vec3{X: -0, Y: 1, Z: 0}.Normalize() // <- sponza
// 	// sunDirection := Vec3{X: 0.09349930860821053, Y: 0.9349930860821052, Z: 10.3421195818254895}.Normalize()

// 	// Set up the window
// 	w.Resize(fyne.NewSize(float32(width), float32(height)))
// 	w.SetFixedSize(true)
// 	w.CenterOnScreen()

// 	// Initial black image
// 	img := image.NewRGBA(image.Rect(0, 0, width, height))
// 	for y := 0; y < height; y++ {
// 		for x := 0; x < width; x++ {
// 			img.Set(x, y, color.RGBA{R: 0, G: 0, B: 0, A: 255})
// 		}
// 	}

// 	dirty := true
// 	w.SetContent(canvas.NewImageFromImage(img))

// 	// Add keystroke handling
// 	w.Canvas().SetOnTypedKey(func(key *fyne.KeyEvent) {
// 		switch key.Name {
// 		case fyne.KeyW:
// 			camera.Position = camera.Position.Add(camera.Forward.Scale(0.1))
// 			dirty = true
// 		case fyne.KeyA:
// 			camera.Position = camera.Position.Add(camera.Right.Scale(-0.1))
// 			dirty = true
// 		case fyne.KeyS:
// 			camera.Position = camera.Position.Add(camera.Forward.Scale(-0.1))
// 			dirty = true
// 		case fyne.KeyD:
// 			camera.Position = camera.Position.Add(camera.Right.Scale(0.1))
// 			dirty = true
// 		case fyne.KeyQ:
// 			camera.Position = camera.Position.Add(camera.Up.Scale(0.1))
// 			dirty = true
// 		case fyne.KeyE:
// 			camera.Position = camera.Position.Add(camera.Up.Scale(-0.1))
// 			dirty = true

// 		case fyne.KeyI:
// 			rotX -= 0.01
// 			camera.SetRotationFromAngles(float64(rotY), float64(rotX))
// 			dirty = true
// 		case fyne.KeyK:
// 			rotX += 0.01
// 			camera.SetRotationFromAngles(float64(rotY), float64(rotX))
// 			dirty = true
// 		case fyne.KeyJ:
// 			rotY -= 0.01
// 			camera.SetRotationFromAngles(float64(rotY), float64(rotX))
// 			dirty = true
// 		case fyne.KeyL:
// 			rotY += 0.01
// 			camera.SetRotationFromAngles(float64(rotY), float64(rotX))
// 			dirty = true

// 		case fyne.KeyPageUp:
// 			bounces++
// 			dirty = true
// 		case fyne.KeyPageDown:
// 			bounces--
// 			dirty = true

// 		case fyne.KeyUp:
// 			sunDirection.Z += 0.1
// 			dirty = true
// 		case fyne.KeyDown:
// 			sunDirection.Z -= 0.1
// 			dirty = true
// 		case fyne.KeyLeft:
// 			sunDirection.X -= 0.1
// 			dirty = true
// 		case fyne.KeyRight:
// 			sunDirection.X += 0.1
// 			dirty = true

// 		case fyne.KeyH:
// 			showStats = !showStats

// 		case fyne.KeySpace:
// 			f, err := os.Create("img.png")
// 			if err != nil {
// 				panic(err)
// 			}
// 			defer f.Close()
// 			if err = png.Encode(f, img); err != nil {
// 				log.Printf("failed to encode: %v", err)
// 			}
// 		}
// 	})

// 	w.Show()

// 	vertices, tris, normals, materials, uvs := DecomposeObjects(scene)
// 	println(len(materials))

// 	fmt.Println("BVH Building...")
// 	bvhx := BuildBVH(vertices, tris, -1000, 1000, -1000, 1000, -1000, 1000, 4, 42)
// 	fmt.Println("BVH Built!")
// 	fmt.Println(bvhx.GetStats(1))
// 	linearBVH := ConstructLinearBVH(bvhx)

// 	startTime := time.Now().UnixMilli()
// 	splitsX := 4
// 	splitsY := 4
// 	startX := 0.0
// 	startY := 0.0
// 	splitSizeX := float64(width / splitsX)
// 	splitSizeY := float64(height / splitsY)
// 	iteration := atomic.Int64{}
// 	tileIndex := atomic.Int64{}

// 	randomSampling := false

// 	cpuFile, err := os.Create("cpu.prof")
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	defer cpuFile.Close()

// 	pprof.StartCPUProfile(cpuFile)

// 	go func() {
// 		time.Sleep(30 * time.Second)
// 		pprof.StopCPUProfile()
// 	}()

// 	calculate := func(targetImg *image.RGBA, imgMutex *sync.Mutex, tt *time.Ticker) {
// 		for {
// 			select {
// 			case <-tt.C:
// 				return
// 			default:
// 				if int(iteration.Load()) >= int(splitSizeX*splitSizeY) {
// 					iteration.Store(0)
// 					if tileIndex.Add(1) == int64(splitsX)*int64(splitsY) {
// 						randomSampling = true
// 						tileIndex.Store(0)
// 					}
// 				}

// 				var dt int
// 				if randomSampling {
// 					dt = int(math.Floor(float64(time.Now().UnixMilli()-startTime)/1000)/10.0) % (splitsX * splitsY)
// 				} else {
// 					dt = int(tileIndex.Load()) % (splitsX * splitsY)
// 				}

// 				X := dt % splitsX
// 				Y := dt / splitsX

// 				offsetX := (splitSizeX * float64(X)) / float64(width)
// 				offsetY := (splitSizeY * float64(Y)) / float64(height)

// 				var rx, ry float64

// 				if randomSampling {
// 					// Calculate random pixel
// 					randX := rand.Float64() * splitSizeX
// 					randY := rand.Float64() * splitSizeY
// 					rx = offsetX + randX/float64(width)
// 					ry = offsetY + randY/float64(height)
// 				} else {
// 					//  Calculate random pixel
// 					randX := rand.Float64()
// 					randY := rand.Float64()
// 					dX := iteration.Load() % int64(splitSizeX)
// 					dY := (iteration.Load() / int64(splitSizeY)) % int64(splitSizeY)
// 					rx = offsetX + (float64(dX)+randX/float64(width))/float64(width)
// 					ry = offsetY + (float64(dY)+randY/float64(height))/float64(height)
// 					iteration.Add(1)
// 				}

// 				pixelX := int(math.Round(rx * float64(width-1)))
// 				pixelY := int(math.Round(ry * float64(height-1)))
// 				startX = ((1.0 - offsetX) * float64(width)) - splitSizeX
// 				startY = offsetY * float64(height)

// 				// Bounds check
// 				if pixelX < 0 || pixelX >= width || pixelY < 0 || pixelY >= height {
// 					fmt.Println(pixelX, pixelY)
// 					return
// 				}

// 				px := (rx - 0.5) * 2
// 				py := (ry - 0.5) * 2
// 				rayOrigin := camera.Position.Add(camera.Forward.Scale(camera.FrustrumDistance)).Add(camera.Up.Scale(py)).Add(camera.Right.Scale(px))
// 				rayDirection := Vec3{
// 					X: rayOrigin.X - camera.Position.X,
// 					Y: rayOrigin.Y - camera.Position.Y,
// 					Z: rayOrigin.Z - camera.Position.Z,
// 				}.Normalize()

// 				ray := NewRay(camera.Position, rayDirection)
// 				rayColor := TraceRay(ray, stepSize, linearBVH, maxSteps, bounces, scatterRays, vertices, normals, materials, uvs, ambient, sunDirection, false)
// 				// When a ray hits a pixel:
// 				pixel := &pixelBuffer[pixelY][pixelX]
// 				pixel.Lock.Lock()

// 				// Accumulate color
// 				pixel.R += float64(rayColor.X)
// 				pixel.G += float64(rayColor.Y)
// 				pixel.B += float64(rayColor.Z)
// 				pixel.SampleCount++

// 				// Calculate running average
// 				avgR := pixel.R / float64(pixel.SampleCount)
// 				avgG := pixel.G / float64(pixel.SampleCount)
// 				avgB := pixel.B / float64(pixel.SampleCount)
// 				pixel.Lock.Unlock()

// 				avgColor := Vec3{
// 					X: avgR,
// 					Y: avgG,
// 					Z: avgB,
// 				}.ToRGBA()

// 				// imgMutex.Lock()
// 				targetImg.Set(width-pixelX, pixelY, avgColor)
// 				// imgMutex.Unlock()
// 			}
// 		}
// 	}

// 	// Animation loop
// 	totalTime := 0.0
// 	timeStep := 500
// 	lastTimeStep := time.Now().UnixMilli()
// 	go func() {
// 		ticker := time.NewTicker(time.Millisecond * time.Duration(timeStep)) // ~60 FPS
// 		defer ticker.Stop()

// 		for {
// 			select {
// 			case <-ticker.C:
// 				time.Sleep(time.Millisecond)

// 				deltaTime := 0.1
// 				totalTime += deltaTime

// 				if dirty {
// 					fmt.Println("Camera:")
// 					fmt.Println("  Pos: ", camera.Position)
// 					fmt.Println("  Fwd: ", camera.Forward)
// 					fmt.Println("  Right: ", camera.Right)
// 					fmt.Println("  Up: ", camera.Up)
// 					fmt.Println("Sun:")
// 					fmt.Println("  Dir: ", sunDirection)
// 					fmt.Println("-----------------------------------")

// 					dirty = false
// 					// Create a new image for this frame
// 					img = image.NewRGBA(image.Rect(0, 0, width, height))

// 					// Clear to black
// 					for y := 0; y < height; y++ {
// 						for x := 0; x < width; x++ {
// 							img.Set(x, y, color.RGBA{R: 0, G: 0, B: 0, A: 255})
// 							pixelBuffer[x][y].Lock.Lock()
// 							pixelBuffer[x][y].R, pixelBuffer[x][y].G, pixelBuffer[x][y].B = 0, 0, 0
// 							pixelBuffer[x][y].SampleCount = 0
// 							pixelBuffer[x][y].Lock.Unlock()
// 						}
// 					}
// 				}

// 				// Render frame with multiple goroutines
// 				var imgMutex sync.Mutex
// 				for range threadCount {
// 					go calculate(img, &imgMutex, ticker)
// 				}

// 				// Update the display on the main UI thread
// 				fyne.Do(func() {
// 					newImage := canvas.NewImageFromImage(img)
// 					newImage.FillMode = canvas.ImageFillOriginal
// 					newImage.Move(fyne.NewPos(0, 0))
// 					newImage.Resize(fyne.NewSize(float32(width), float32(height)))

// 					ts := time.Now().UnixMilli() - lastTimeStep + 1
// 					raysTracedCount := float64(raysTraced.Load()) * 1000 / float64(ts)
// 					raysTraced.Store(0)
// 					lastTimeStep = time.Now().UnixMilli()

// 					unit := ""
// 					if raysTracedCount > 1e6 {
// 						raysTracedCount /= 1e6
// 						unit = "M"
// 					} else if raysTracedCount > 1e3 {
// 						raysTracedCount /= 1e3
// 						unit = "K"
// 					}

// 					text := canvas.NewText(fmt.Sprintf("%.1f %srays/s", raysTracedCount, unit), color.White)
// 					text.TextSize = 10
// 					text.Move(fyne.NewPos(5, 10))

// 					sampleRect := canvas.NewRectangle(color.Transparent)
// 					sampleRect.Move(fyne.NewPos(float32(startX), float32(startY)))
// 					sampleRect.Resize(fyne.NewSize(float32(width)/float32(splitsX), float32(height)/float32(splitsY)))
// 					sampleRect.StrokeColor = color.White
// 					sampleRect.StrokeWidth = 1.0

// 					container := container.NewWithoutLayout(newImage)
// 					if showStats {
// 						container.Add(text)
// 					}
// 					container.Add(sampleRect)

// 					w.SetContent(container)
// 				})

// 			default:
// 				time.Sleep(time.Millisecond)
// 			}
// 		}
// 	}()

// 	// Run the app (this blocks until the window is closed)
// 	a.Run()
// }

package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"math/rand"
	"os"
	"runtime/pprof"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
)

var threadCount = 16 // Number of goroutines to use for rendering

func main() {
	a := app.New()
	w := a.NewWindow("Path Tracer")

	width := 768
	height := 768

	showStats := true

	bounces := 4
	scatterRays := 4
	ambient := 0.0
	maxSteps := 1
	stepSize := 1000.0

	// // Scene Cornell
	// rotY := 0.0 * 0.0174533
	// rotX := 180.0 * 0.0174533

	// Scene Sponza
	rotY := 170.0 * 0.0174533
	rotX := 165.0 * 0.0174533

	camera := Camera{
		// Position: Vec3{X: 0, Y: 0.6, Z: -2.2}, // <--- cornell
		Position:         Vec3{X: -3.2, Y: 0.5, Z: 21}, // <--- sponza
		Forward:          Vec3{X: 0, Y: 0, Z: -1}.Normalize(),
		Right:            Vec3{X: 1, Y: 0, Z: 0},
		Up:               Vec3{X: 0, Y: -1, Z: 0},
		FrustrumDistance: 2,
	}
	camera.ApplyRotation(rotY, rotX)

	pixelBuffer := make([][]Pixel, height)
	for i := range pixelBuffer {
		pixelBuffer[i] = make([]Pixel, width)
	}

	// 	// boxMesh, _ := LoadObj("C:\\Users\\smpsm\\OneDrive\\Documents\\Untitled.obj", 1)
	// 	// boxMesh, _, _ := LoadObj("C:\\Users\\smpsm\\OneDrive\\Documents\\2B2.obj", 1)
	// boxMesh, _, _ := LoadObj("C:\\Users\\smpsm\\OneDrive\\Documents\\CornellSphere.obj", 1)
	// 	// boxMesh, _, _ := LoadObj("C:\\Users\\smpsm\\OneDrive\\Documents\\cornell.obj", 1)
	// 	// boxMesh, _, _ := LoadObj("C:\\Users\\smpsm\\OneDrive\\Documents\\Emissions.obj", 1)
	boxMesh, _, _ := LoadObj("C:\\Users\\smpsm\\OneDrive\\Documents\\sponza.obj", 1.5)
	// 	// boxMesh, _, _ := LoadObj("C:\\Users\\smpsm\\OneDrive\\Documents\\2B.obj", 2)
	// 	// boxMesh, _, _ := LoadObj("C:\\Users\\smpsm\\OneDrive\\Documents\\cube.obj", 1.0)
	boxObj := GameObject{
		Position: Vec3{Z: 0},
		Mesh:     *boxMesh,
	}

	scene := []GameObject{
		boxObj,
	}

	// sunDirection := Vec3{X: -0.1, Y: 1, Z: 10}.Normalize() // <- cornell sphere
	sunDirection := Vec3{X: -0.1, Y: 1, Z: 0.1}.Normalize() // <- sponza

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
	dirtyMutex := &sync.Mutex{}

	w.SetContent(canvas.NewImageFromImage(img))

	// Add keystroke handling
	w.Canvas().SetOnTypedKey(func(key *fyne.KeyEvent) {
		dirtyMutex.Lock()
		defer dirtyMutex.Unlock()

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

		case fyne.KeyI:
			rotX -= 0.01
			camera.SetRotationFromAngles(float64(rotY), float64(rotX))
			dirty = true
		case fyne.KeyK:
			rotX += 0.01
			camera.SetRotationFromAngles(float64(rotY), float64(rotX))
			dirty = true
		case fyne.KeyJ:
			rotY -= 0.01
			camera.SetRotationFromAngles(float64(rotY), float64(rotX))
			dirty = true
		case fyne.KeyL:
			rotY += 0.01
			camera.SetRotationFromAngles(float64(rotY), float64(rotX))
			dirty = true

		case fyne.KeyPageUp:
			bounces++
			dirty = true
		case fyne.KeyPageDown:
			bounces--
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

		case fyne.KeyH:
			showStats = !showStats

		case fyne.KeySpace:
			f, err := os.Create("img.png")
			if err != nil {
				panic(err)
			}
			defer f.Close()
			if err = png.Encode(f, img); err != nil {
				log.Printf("failed to encode: %v", err)
			}
		}
	})

	w.Show()

	vertices, tris, normals, materials, uvs := DecomposeObjects(scene)
	println(len(materials))

	fmt.Println("BVH Building...")
	bvhx := BuildBVH(vertices, tris, -1000, 1000, -1000, 1000, -1000, 1000, 4, 42)
	fmt.Println("BVH Built!")
	fmt.Println(bvhx.GetStats(1))
	linearBVH := ConstructLinearBVH(bvhx)

	startTime := time.Now().UnixMilli()
	splitsX := 4
	splitsY := 4
	startX := 0.0
	startY := 0.0
	splitSizeX := float64(width / splitsX)
	splitSizeY := float64(height / splitsY)
	iteration := atomic.Int64{}
	tileIndex := atomic.Int64{}

	randomSampling := false

	cpuFile, err := os.Create("cpu.prof")
	if err != nil {
		log.Fatal(err)
	}
	defer cpuFile.Close()

	pprof.StartCPUProfile(cpuFile)

	go func() {
		time.Sleep(30 * time.Second)
		pprof.StopCPUProfile()
	}()

	// Channel to signal render workers to stop
	stopRendering := make(chan bool)
	renderingActive := atomic.Bool{}
	renderingActive.Store(true)

	// Calculate function now runs continuously
	calculate := func() {
		for {
			select {
			case <-stopRendering:
				return
			default:
				if !renderingActive.Load() {
					time.Sleep(time.Millisecond)
					continue
				}

				if int(iteration.Load()) >= int(splitSizeX*splitSizeY) {
					iteration.Store(0)
					if tileIndex.Add(1) == int64(splitsX)*int64(splitsY) {
						randomSampling = true
						tileIndex.Store(0)
					}
				}

				var dt int
				if randomSampling {
					dt = int(math.Floor(float64(time.Now().UnixMilli()-startTime)/1000)/10.0) % (splitsX * splitsY)
				} else {
					dt = int(tileIndex.Load()) % (splitsX * splitsY)
				}

				X := dt % splitsX
				Y := dt / splitsX

				offsetX := (splitSizeX * float64(X)) / float64(width)
				offsetY := (splitSizeY * float64(Y)) / float64(height)

				var rx, ry float64

				if randomSampling {
					// Calculate random pixel
					randX := rand.Float64() * splitSizeX
					randY := rand.Float64() * splitSizeY
					rx = offsetX + randX/float64(width)
					ry = offsetY + randY/float64(height)
				} else {
					//  Calculate random pixel
					randX := rand.Float64()
					randY := rand.Float64()
					dX := iteration.Load() % int64(splitSizeX)
					dY := (iteration.Load() / int64(splitSizeY)) % int64(splitSizeY)
					rx = offsetX + (float64(dX)+randX/float64(width))/float64(width)
					ry = offsetY + (float64(dY)+randY/float64(height))/float64(height)
					iteration.Add(1)
				}

				pixelX := int(math.Round(rx * float64(width-1)))
				pixelY := int(math.Round(ry * float64(height-1)))
				startX = ((1.0 - offsetX) * float64(width)) - splitSizeX
				startY = offsetY * float64(height)

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

				ray := NewRay(camera.Position, rayDirection)
				rayColor := TraceRay(ray, stepSize, linearBVH, maxSteps, bounces, scatterRays, vertices, normals, materials, uvs, ambient, sunDirection, false)

				// When a ray hits a pixel:
				pixel := &pixelBuffer[pixelY][pixelX]
				pixel.Lock.Lock()

				// Accumulate color
				pixel.R += float64(rayColor.X)
				pixel.G += float64(rayColor.Y)
				pixel.B += float64(rayColor.Z)
				pixel.SampleCount++

				// Calculate running average
				avgR := pixel.R / float64(pixel.SampleCount)
				avgG := pixel.G / float64(pixel.SampleCount)
				avgB := pixel.B / float64(pixel.SampleCount)
				pixel.Lock.Unlock()

				avgColor := Vec3{
					X: avgR,
					Y: avgG,
					Z: avgB,
				}.ToRGBA()

				img.Set(width-pixelX, pixelY, avgColor)
			}
		}
	}

	// Start render workers that run continuously
	for i := 0; i < threadCount; i++ {
		go calculate()
	}

	// Display update loop - runs at fixed 30 FPS
	lastStatsUpdate := time.Now()
	go func() {
		displayTicker := time.NewTicker(time.Second / 30) // 30 FPS
		defer displayTicker.Stop()

		for range displayTicker.C {
			dirtyMutex.Lock()
			if dirty {
				fmt.Println("Camera:")
				fmt.Println("  Pos: ", camera.Position)
				fmt.Println("  Fwd: ", camera.Forward)
				fmt.Println("  Right: ", camera.Right)
				fmt.Println("  Up: ", camera.Up)
				fmt.Println("Sun:")
				fmt.Println("  Dir: ", sunDirection)
				fmt.Println("-----------------------------------")

				// Pause rendering while we clear
				renderingActive.Store(false)
				time.Sleep(10 * time.Millisecond) // Give workers time to pause

				// Clear image and pixel buffer
				for y := 0; y < height; y++ {
					for x := 0; x < width; x++ {
						img.Set(x, y, color.RGBA{R: 0, G: 0, B: 0, A: 255})
						pixelBuffer[y][x].Lock.Lock()
						pixelBuffer[y][x].R, pixelBuffer[y][x].G, pixelBuffer[y][x].B = 0, 0, 0
						pixelBuffer[y][x].SampleCount = 0
						pixelBuffer[y][x].Lock.Unlock()
					}
				}

				// Reset iteration counters
				iteration.Store(0)
				tileIndex.Store(0)
				randomSampling = false

				// Resume rendering
				renderingActive.Store(true)
				dirty = false
			}
			dirtyMutex.Unlock()

			// Update the display
			fyne.Do(func() {
				newImage := canvas.NewImageFromImage(img)
				newImage.FillMode = canvas.ImageFillOriginal
				newImage.Move(fyne.NewPos(0, 0))
				newImage.Resize(fyne.NewSize(float32(width), float32(height)))

				container := container.NewWithoutLayout(newImage)

				// Update stats every second
				if showStats {
					raysTracedCount := float64(raysTraced.Load()) * 1000 / float64(time.Since(lastStatsUpdate).Abs().Milliseconds())
					if time.Since(lastStatsUpdate) > time.Second {
						lastStatsUpdate = time.Now()
						raysTraced.Store(0)
					}

					unit := ""
					if raysTracedCount > 1e6 {
						raysTracedCount /= 1e6
						unit = "M"
					} else if raysTracedCount > 1e3 {
						raysTracedCount /= 1e3
						unit = "K"
					}

					text := canvas.NewText(fmt.Sprintf("%.1f %srays/s", raysTracedCount, unit), color.White)
					text.TextSize = 10
					text.Move(fyne.NewPos(5, 10))
					container.Add(text)
				}

				sampleRect := canvas.NewRectangle(color.Transparent)
				sampleRect.Move(fyne.NewPos(float32(startX), float32(startY)))
				sampleRect.Resize(fyne.NewSize(float32(width)/float32(splitsX), float32(height)/float32(splitsY)))
				sampleRect.StrokeColor = color.White
				sampleRect.StrokeWidth = 1.0
				container.Add(sampleRect)

				w.SetContent(container)
			})
		}
	}()

	// Run the app (this blocks until the window is closed)
	a.Run()

	// Clean up
	close(stopRendering)
}
