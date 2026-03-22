package main

import (
	"fmt"
	"image/color"
	"math"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ElevationChart is a custom Fyne widget that draws a filled elevation profile.
type ElevationChart struct {
	widget.BaseWidget

	title    string
	raster   *canvas.Raster
	titleLbl *widget.Label
	minLbl   *canvas.Text
	maxLbl   *canvas.Text
	distLbl  *canvas.Text
	emptyLbl *widget.Label
	hasData  bool

	mu     sync.RWMutex
	points []TrackPoint
	dists  []float64 // cumulative distances parallel to points

	// Pre-computed per-column lookup. Recomputed when width changes.
	colWidth int
	colEle   []float64
}

var (
	chartLineColor = color.NRGBA{R: 0, G: 122, B: 255, A: 255}
	chartFillColor = color.NRGBA{R: 0, G: 122, B: 255, A: 55}
	chartGridColor = color.NRGBA{R: 128, G: 128, B: 128, A: 40}
)

// NewElevationChart creates a new elevation profile chart widget.
func NewElevationChart(title string) *ElevationChart {
	c := &ElevationChart{title: title}
	c.ExtendBaseWidget(c)

	c.titleLbl = widget.NewLabel(title)
	c.titleLbl.TextStyle = fyne.TextStyle{Bold: true}

	c.minLbl = canvas.NewText("", theme.ForegroundColor())
	c.minLbl.TextSize = 10
	c.maxLbl = canvas.NewText("", theme.ForegroundColor())
	c.maxLbl.TextSize = 10
	c.distLbl = canvas.NewText("", theme.ForegroundColor())
	c.distLbl.TextSize = 10

	c.emptyLbl = widget.NewLabel("No data")
	c.emptyLbl.Alignment = fyne.TextAlignCenter

	c.raster = canvas.NewRasterWithPixels(c.drawPixel)
	c.raster.SetMinSize(fyne.NewSize(320, 160))
	c.raster.Hide() // hidden until data is loaded

	return c
}

// SetData updates the chart with new track points and triggers a redraw.
func (c *ElevationChart) SetData(pts []TrackPoint) {
	c.mu.Lock()
	c.points = pts
	c.dists = CumulativeDistance(pts)
	c.colWidth = 0 // invalidate pre-computed lookup
	c.colEle = nil

	// Update axis labels
	minE, maxE := elevationRange(pts)
	totalDist := 0.0
	if len(c.dists) > 0 {
		totalDist = c.dists[len(c.dists)-1]
	}
	c.mu.Unlock()

	hasEle := len(pts) >= 2 && !math.IsNaN(minE)
	c.hasData = hasEle

	if hasEle {
		c.minLbl.Text = fmt.Sprintf("%.0fm", minE)
		c.maxLbl.Text = fmt.Sprintf("%.0fm", maxE)
		c.distLbl.Text = fmt.Sprintf("%.1f km", totalDist/1000)
		c.emptyLbl.Hide()
		c.raster.Show()
	} else if len(pts) >= 2 {
		// Track points exist but no elevation data
		c.minLbl.Text = ""
		c.maxLbl.Text = ""
		c.distLbl.Text = fmt.Sprintf("%.1f km", totalDist/1000)
		c.emptyLbl.SetText("No elevation data in source file")
		c.emptyLbl.Show()
		c.raster.Hide()
	} else {
		c.minLbl.Text = ""
		c.maxLbl.Text = ""
		c.distLbl.Text = ""
		c.emptyLbl.SetText("No data")
		c.emptyLbl.Show()
		c.raster.Hide()
	}
	c.minLbl.Refresh()
	c.maxLbl.Refresh()
	c.distLbl.Refresh()

	c.raster.Refresh()
	c.Refresh()
}

// Clear removes all data from the chart.
func (c *ElevationChart) Clear() {
	c.SetData(nil)
}

func (c *ElevationChart) CreateRenderer() fyne.WidgetRenderer {
	// Stack the raster and empty label so only one is visible at a time.
	centre := container.NewStack(c.raster, container.NewCenter(c.emptyLbl))
	content := container.NewBorder(
		container.NewHBox(c.titleLbl),
		container.NewHBox(c.minLbl, widget.NewLabel(""), c.distLbl),
		nil, nil,
		centre,
	)
	return widget.NewSimpleRenderer(content)
}

// drawPixel is called by canvas.Raster for every pixel.
func (c *ElevationChart) drawPixel(x, y, w, h int) color.Color {
	c.mu.RLock()
	pts := c.points
	dists := c.dists
	c.mu.RUnlock()

	if len(pts) < 2 {
		return theme.BackgroundColor()
	}

	// Recompute column lookup table when width changes
	c.mu.Lock()
	if c.colWidth != w {
		c.colEle = precomputeColumns(pts, dists, w)
		c.colWidth = w
	}
	cols := c.colEle
	c.mu.Unlock()

	if cols == nil {
		return theme.BackgroundColor()
	}

	// Plot area padding (pixels)
	const padLeft, padRight, padTop, padBottom = 2, 2, 2, 2
	plotW := w - padLeft - padRight
	plotH := h - padTop - padBottom

	px := x - padLeft
	py := y - padTop
	if px < 0 || py < 0 || px >= plotW || py >= plotH {
		return theme.BackgroundColor()
	}

	// Column elevation
	if px >= len(cols) {
		return theme.BackgroundColor()
	}
	eleAtX := cols[px]
	if math.IsNaN(eleAtX) {
		return theme.BackgroundColor()
	}

	minE, maxE := elevationRange(pts)
	if maxE == minE {
		maxE = minE + 1
	}

	// Map elevation to pixel row (y=0 is top)
	eleNorm := (eleAtX - minE) / (maxE - minE)
	// Add small padding so the line doesn't sit at the very edge
	eleRow := plotH - 1 - int(eleNorm*float64(plotH-4)) - 2

	// Horizontal grid lines at 25% / 50% / 75%
	for _, frac := range []float64{0.25, 0.5, 0.75} {
		gridRow := plotH - 1 - int(frac*float64(plotH-4)) - 2
		if py == gridRow {
			return chartGridColor
		}
	}

	if py == eleRow || py == eleRow-1 {
		return chartLineColor
	}
	if py > eleRow {
		return chartFillColor
	}
	return theme.BackgroundColor()
}

// precomputeColumns builds a []float64 of length plotW with the interpolated
// elevation at each pixel column. O(n) per column avoided by single sweep.
func precomputeColumns(pts []TrackPoint, dists []float64, w int) []float64 {
	const padLeft, padRight = 2, 2
	plotW := w - padLeft - padRight
	if plotW <= 0 || len(pts) < 2 {
		return nil
	}

	totalDist := dists[len(dists)-1]
	if totalDist == 0 {
		return nil
	}

	cols := make([]float64, plotW)
	ptIdx := 0

	for px := 0; px < plotW; px++ {
		distAtX := (float64(px) / float64(plotW)) * totalDist

		// Advance ptIdx to the segment containing distAtX
		for ptIdx < len(dists)-2 && dists[ptIdx+1] < distAtX {
			ptIdx++
		}

		i := ptIdx
		d0, d1 := dists[i], dists[i+1]
		e0, e1 := pts[i].Ele, pts[i+1].Ele

		if math.IsNaN(e0) || math.IsNaN(e1) {
			cols[px] = math.NaN()
			continue
		}

		var f float64
		if d1 > d0 {
			f = (distAtX - d0) / (d1 - d0)
		}
		cols[px] = e0 + f*(e1-e0)
	}
	return cols
}

func elevationRange(pts []TrackPoint) (minE, maxE float64) {
	minE = math.MaxFloat64
	maxE = -math.MaxFloat64
	for _, p := range pts {
		if !math.IsNaN(p.Ele) {
			if p.Ele < minE {
				minE = p.Ele
			}
			if p.Ele > maxE {
				maxE = p.Ele
			}
		}
	}
	if minE == math.MaxFloat64 {
		return math.NaN(), math.NaN()
	}
	return
}
