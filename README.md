## How to run
- Install Go
- Install Fyne - https://docs.fyne.io/started/
- Make sure fyne is properly installed
- Run `go get .`
- In `main.go`, set the `threadCount` according to your CPU's threads - 1 (for stability).
- In `main.go`, load a `.obj` file of your choosing
- Run `go run .`
Enjoy!

## Warning
This will push your PC extremely hard. It may crash if `threadCount` is set too high.