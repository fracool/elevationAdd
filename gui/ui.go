package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// AppState holds all mutable UI state and widget references.
type AppState struct {
	// Settings
	gpxPath    string
	rasterDir  string
	outputPath string
	densify    bool
	maxSpacing float64

	// Widgets
	gpxEntry     *widget.Entry
	rasterEntry  *widget.Entry
	outputEntry  *widget.Entry
	spacingSlider *widget.Slider
	spacingLabel *widget.Label
	densifyCheck *widget.Check
	runBtn       *widget.Button
	progressBar  *widget.ProgressBarInfinite
	logWidget    *widget.Entry
	chartBefore  *ElevationChart
	chartAfter   *ElevationChart

	// Runner
	runner  *Runner
	running bool

	// Fyne refs
	app    fyne.App
	window fyne.Window
}

// buildUI constructs and returns the full application UI.
func buildUI(a fyne.App, w fyne.Window) fyne.CanvasObject {
	state := &AppState{
		app:        a,
		window:     w,
		maxSpacing: a.Preferences().FloatWithFallback("max_spacing", 1.0),
		densify:    a.Preferences().Bool("densify"),
	}
	state.runner = &Runner{state: state}

	// ── GPX input row ──────────────────────────────────────────────────────
	state.gpxEntry = widget.NewEntry()
	state.gpxEntry.SetPlaceHolder("Select input .gpx file…")
	if last := a.Preferences().String("last_gpx_path"); last != "" {
		state.gpxEntry.SetText(last)
		state.gpxPath = last
	}
	state.gpxEntry.OnChanged = func(v string) {
		state.gpxPath = v
		a.Preferences().SetString("last_gpx_path", v)
		// Auto-fill output path
		if v != "" && state.outputEntry.Text == "" {
			ext := filepath.Ext(v)
			auto := v[:len(v)-len(ext)] + "-lidar.gpx"
			state.outputEntry.SetText(auto)
			state.outputPath = auto
		}
		go state.loadBeforeChart()
	}
	gpxBrowse := widget.NewButtonWithIcon("", theme.FolderOpenIcon(), func() {
		fd := dialog.NewFileOpen(func(r fyne.URIReadCloser, err error) {
			if err != nil || r == nil {
				return
			}
			p := r.URI().Path()
			state.gpxEntry.SetText(p)
		}, w)
		fd.SetFilter(storage.NewExtensionFileFilter([]string{".gpx"}))
		if lastDir := a.Preferences().String("last_gpx_dir"); lastDir != "" {
			if luri, err := storage.ListerForURI(storage.NewFileURI(lastDir)); err == nil {
				fd.SetLocation(luri)
			}
		}
		fd.Show()
	})
	gpxRow := container.NewBorder(nil, nil, nil, gpxBrowse, state.gpxEntry)

	// ── Raster folder row ──────────────────────────────────────────────────
	state.rasterEntry = widget.NewEntry()
	state.rasterEntry.SetPlaceHolder("Select folder containing LiDAR .tif files…")
	if last := a.Preferences().String("last_raster_dir"); last != "" {
		state.rasterEntry.SetText(last)
		state.rasterDir = last
	}
	state.rasterEntry.OnChanged = func(v string) {
		state.rasterDir = v
		a.Preferences().SetString("last_raster_dir", v)
	}
	rasterBrowse := widget.NewButtonWithIcon("", theme.FolderIcon(), func() {
		dialog.ShowFolderOpen(func(lu fyne.ListableURI, err error) {
			if err != nil || lu == nil {
				return
			}
			p := lu.Path()
			state.rasterEntry.SetText(p)
			a.Preferences().SetString("last_raster_dir", p)
		}, w)
	})
	rasterRow := container.NewBorder(nil, nil, nil, rasterBrowse, state.rasterEntry)

	// ── Output file row ────────────────────────────────────────────────────
	state.outputEntry = widget.NewEntry()
	state.outputEntry.SetPlaceHolder("Output .gpx file path…")
	if state.gpxPath != "" && state.outputPath == "" {
		ext := filepath.Ext(state.gpxPath)
		state.outputPath = state.gpxPath[:len(state.gpxPath)-len(ext)] + "-lidar.gpx"
		state.outputEntry.SetText(state.outputPath)
	}
	state.outputEntry.OnChanged = func(v string) {
		state.outputPath = v
	}
	outputBrowse := widget.NewButtonWithIcon("", theme.DocumentSaveIcon(), func() {
		fd := dialog.NewFileSave(func(w fyne.URIWriteCloser, err error) {
			if err != nil || w == nil {
				return
			}
			p := w.URI().Path()
			if !strings.HasSuffix(strings.ToLower(p), ".gpx") {
				p += ".gpx"
			}
			state.outputEntry.SetText(p)
			state.outputPath = p
		}, state.window)
		fd.SetFilter(storage.NewExtensionFileFilter([]string{".gpx"}))
		fd.Show()
	})
	outputRow := container.NewBorder(nil, nil, nil, outputBrowse, state.outputEntry)

	// ── Densify + spacing ──────────────────────────────────────────────────
	state.spacingLabel = widget.NewLabel(fmt.Sprintf("%.1f m", state.maxSpacing))
	state.spacingSlider = widget.NewSlider(0.5, 20.0)
	state.spacingSlider.Step = 0.5
	state.spacingSlider.Value = state.maxSpacing
	state.spacingSlider.OnChanged = func(v float64) {
		state.maxSpacing = v
		state.spacingLabel.SetText(fmt.Sprintf("%.1f m", v))
		a.Preferences().SetFloat("max_spacing", v)
	}

	state.densifyCheck = widget.NewCheck("Interpolate intermediate points (densify track)", func(v bool) {
		state.densify = v
		a.Preferences().SetBool("densify", v)
		if v {
			state.spacingSlider.Enable()
		} else {
			state.spacingSlider.Disable()
		}
	})
	state.densifyCheck.SetChecked(state.densify)
	if !state.densify {
		state.spacingSlider.Disable()
	}

	spacingRow := container.NewBorder(nil, nil, nil, state.spacingLabel, state.spacingSlider)

	// ── Settings form ──────────────────────────────────────────────────────
	form := widget.NewForm(
		widget.NewFormItem("GPX Track File", gpxRow),
		widget.NewFormItem("LiDAR Raster Folder (UK)", rasterRow),
		widget.NewFormItem("Output File", outputRow),
		widget.NewFormItem("Densify", state.densifyCheck),
		widget.NewFormItem("Max Point Spacing", spacingRow),
	)

	// ── Run / Cancel button + progress ─────────────────────────────────────
	state.runBtn = widget.NewButtonWithIcon("Run Enhancement", theme.MediaPlayIcon(), func() {
		if state.running {
			state.runner.Cancel()
		} else {
			state.startRun()
		}
	})
	state.runBtn.Importance = widget.HighImportance

	state.progressBar = widget.NewProgressBarInfinite()
	state.progressBar.Stop()
	state.progressBar.Hide()

	buttonRow := container.New(layout.NewHBoxLayout(),
		state.runBtn,
		state.progressBar,
		layout.NewSpacer(),
	)

	settingsCard := widget.NewCard("Settings", "", container.NewVBox(form, buttonRow))

	// ── Log area ───────────────────────────────────────────────────────────
	state.logWidget = widget.NewMultiLineEntry()
	state.logWidget.Disable()
	state.logWidget.SetMinRowsVisible(8)
	state.logWidget.Wrapping = fyne.TextWrapWord
	logScroll := container.NewVScroll(state.logWidget)
	logScroll.SetMinSize(fyne.NewSize(0, 180))
	logCard := widget.NewCard("Processing Log", "", logScroll)

	// ── Elevation charts ───────────────────────────────────────────────────
	state.chartBefore = NewElevationChart("Before — Original GPX")
	state.chartAfter = NewElevationChart("After — LiDAR Enhanced")

	chartSplit := container.NewHSplit(state.chartBefore, state.chartAfter)
	chartSplit.Offset = 0.5
	chartCard := widget.NewCard("Elevation Profiles", "", chartSplit)

	// Load before-chart if GPX path was restored from prefs
	if state.gpxPath != "" {
		go state.loadBeforeChart()
	}

	// ── Master layout ──────────────────────────────────────────────────────
	mainSplit := container.NewVSplit(
		settingsCard,
		container.NewVBox(logCard, chartCard),
	)
	mainSplit.Offset = 0.38

	return mainSplit
}

// startRun validates inputs and kicks off the subprocess.
func (s *AppState) startRun() {
	if err := s.validateInputs(); err != nil {
		dialog.ShowError(err, s.window)
		return
	}

	s.running = true
	s.logWidget.SetText("")
	s.progressBar.Show()
	s.progressBar.Start()
	s.runBtn.SetText("Cancel")
	s.runBtn.SetIcon(theme.CancelIcon())
	s.chartAfter.Clear()

	go s.runner.Run(RunArgs{
		GPXPath:    s.gpxPath,
		RasterDir:  s.rasterDir,
		OutputPath: s.outputPath,
		Densify:    s.densify,
		MaxSpacing: s.maxSpacing,
	})
}

// onRunComplete is called by Runner (via fyne.Do) when the process exits.
func (s *AppState) onRunComplete(success bool) {
	s.running = false
	s.progressBar.Stop()
	s.progressBar.Hide()
	s.runBtn.SetText("Run Enhancement")
	s.runBtn.SetIcon(theme.MediaPlayIcon())

	if success {
		go s.loadAfterChart()
	}
}

func (s *AppState) validateInputs() error {
	if s.gpxPath == "" {
		return fmt.Errorf("Please select an input GPX file.")
	}
	if _, err := os.Stat(s.gpxPath); err != nil {
		return fmt.Errorf("GPX file not found:\n%s", s.gpxPath)
	}
	if s.rasterDir == "" {
		return fmt.Errorf("Please select the LiDAR raster folder.")
	}
	if _, err := os.Stat(s.rasterDir); err != nil {
		return fmt.Errorf("Raster folder not found:\n%s", s.rasterDir)
	}
	if s.outputPath == "" {
		return fmt.Errorf("Please specify an output file path.")
	}
	outDir := filepath.Dir(s.outputPath)
	if _, err := os.Stat(outDir); err != nil {
		return fmt.Errorf("Output directory does not exist:\n%s", outDir)
	}
	if s.densify && (s.maxSpacing < 0.1 || s.maxSpacing > 100) {
		return fmt.Errorf("Max spacing must be between 0.1 and 100 metres.")
	}
	return nil
}

func (s *AppState) loadBeforeChart() {
	if s.gpxPath == "" {
		return
	}
	pts, err := ParseGPX(s.gpxPath)
	if err != nil {
		return
	}
	fyne.Do(func() {
		s.chartBefore.SetData(pts)
	})
}

func (s *AppState) loadAfterChart() {
	if s.outputPath == "" {
		return
	}
	pts, err := ParseGPX(s.outputPath)
	if err != nil {
		return
	}
	fyne.Do(func() {
		s.chartAfter.SetData(pts)
	})
}
