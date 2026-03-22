package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/dialog"
)

func main() {
	a := app.NewWithID("com.gpxlidar.gui")
	a.Settings().SetTheme(newAppTheme())

	w := a.NewWindow("GPX LiDAR Elevation Enhancer")
	w.Resize(fyne.NewSize(960, 740))
	w.SetMaster()

	// Check Python environment in the background; show a warning dialog if needed.
	go func() {
		if err := checkPythonEnv(); err != nil {
			fyne.Do(func() {
				dialog.ShowError(err, w)
			})
		}
	}()

	w.SetContent(buildUI(a, w))
	w.ShowAndRun()
}
