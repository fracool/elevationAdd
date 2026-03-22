package main

import (
	"fmt"
	"image"
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

	mu     sync.Mutex
	points []TrackPoint
	dists  []float64 // cumulative distances parallel to points

	// Pre-computed per render. Invalidated when data or width changes.
	colWidth int
	colEle   []float64
	minE     float64
	maxE     float64
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

	// drawImage renders the whole chart image at once — far faster than
	// the per-pixel callback because elevationRange is called once, not
	// once per pixel, and there is no per-pixel lock overhead.
	c.raster = canvas.NewRaster(c.drawImage)
	c.raster.SetMinSize(fyne.NewSize(320, 160))
	c.raster.Hide() // hidden until data is loaded

	return c
}

// SetData updates the chart with new track points and triggers a redraw.
func (c *ElevationChart) SetData(pts []TrackPoint) {
	dists := CumulativeDistance(pts)
	minE, maxE := elevationRange(pts)
	totalDist := 0.0
	if len(dists) > 0 {
		totalDist = dists[len(dists)-1]
	}

	c.mu.Lock()
	c.points = pts
	c.dists = dists
	c.colWidth = 0 // invalidate cached columns
	c.colEle = nil
	c.minE = minE
	c.maxE = maxE
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
	centre := container.NewStack(c.raster, container.NewCenter(c.emptyLbl))
	content := container.NewBorder(
		container.NewHBox(c.titleLbl),
		container.NewHBox(c.minLbl, widget.NewLabel(""), c.distLbl),
		nil, nil,
		centre,
	)
	return widget.NewSimpleRenderer(content)
}

// drawImage renders the full elevation chart into an image.NRGBA.
// Called by canvas.Raster once per frame — not once per pixel.
func (c *ElevationChart) drawImage(w, h int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))

	c.mu.Lock()
	pts := c.points
	dists := c.dists
	if len(pts) >= 2 && c.colWidth != w {
		c.colEle = precomputeColumns(pts, dists, w)
		c.colWidth = w
		// Recompute range in case it wasn't set yet (e.g. width changed
		// before SetData had a chance to store it).
		c.minE, c.maxE = elevationRange(pts)
	}
	cols := c.colEle
	minE := c.minE
	maxE := c.maxE
	c.mu.Unlock()

	bg := theme.BackgroundColor()
	if len(pts) < 2 || cols == nil {
		fillImage(img, bg)
		return img
	}

	if maxE == minE {
		maxE = minE + 1
	}

	const padLeft, padRight, padTop, padBottom = 2, 2, 2, 2
	plotW := w - padLeft - padRight
	plotH := h - padTop - padBottom

	// Pre-compute the elevation row for each column.
	eleRow := make([]int, plotW)
	for px := 0; px < plotW && px < len(cols); px++ {
		e := cols[px]
		if math.IsNaN(e) {
			eleRow[px] = -1
			continue
		}
		norm := (e - minE) / (maxE - minE)
		eleRow[px] = plotH - 1 - int(norm*float64(plotH-4)) - 2
	}

	// Pre-compute grid rows.
	gridRows := [3]int{}
	for i, frac := range []float64{0.25, 0.5, 0.75} {
		gridRows[i] = plotH - 1 - int(frac*float64(plotH-4)) - 2
	}

	// Render row by row (better cache locality than column by column).
	for py := 0; py < h; py++ {
		for px := 0; px < w; px++ {
			// Border padding
			ix := px - padLeft
			iy := py - padTop
			if ix < 0 || iy < 0 || ix >= plotW || iy >= plotH {
				setPixel(img, px, py, bg)
				continue
			}
			if ix >= len(eleRow) {
				setPixel(img, px, py, bg)
				continue
			}

			er := eleRow[ix]
			if er < 0 {
				setPixel(img, px, py, bg)
				continue
			}

			// Grid lines
			isGrid := false
			for _, gr := range gridRows {
				if iy == gr {
					isGrid = true
					break
				}
			}
			if isGrid {
				setPixel(img, px, py, chartGridColor)
				continue
			}

			if iy == er || iy == er-1 {
				setPixel(img, px, py, chartLineColor)
			} else if iy > er {
				setPixel(img, px, py, chartFillColor)
			} else {
				setPixel(img, px, py, bg)
			}
		}
	}
	return img
}

// setPixel writes an NRGBA colour directly into the image buffer.
func setPixel(img *image.NRGBA, x, y int, c color.Color) {
	r, g, b, a := c.RGBA()
	off := img.PixOffset(x, y)
	img.Pix[off+0] = uint8(r >> 8)
	img.Pix[off+1] = uint8(g >> 8)
	img.Pix[off+2] = uint8(b >> 8)
	img.Pix[off+3] = uint8(a >> 8)
}

// fillImage fills the entire image with a single colour.
func fillImage(img *image.NRGBA, c color.Color) {
	r, g, b, a := c.RGBA()
	r8, g8, b8, a8 := uint8(r>>8), uint8(g>>8), uint8(b>>8), uint8(a>>8)
	pix := img.Pix
	for i := 0; i < len(pix); i += 4 {
		pix[i+0] = r8
		pix[i+1] = g8
		pix[i+2] = b8
		pix[i+3] = a8
	}
}

// precomputeColumns builds a []float64 of length plotW with the interpolated
// elevation at each pixel column. Single O(n+plotW) sweep.
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
