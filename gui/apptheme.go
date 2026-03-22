package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// appTheme wraps the default theme and overrides the primary accent colour
// to the iOS blue used in the elevation chart.
type appTheme struct{}

var _ fyne.Theme = (*appTheme)(nil)

func newAppTheme() fyne.Theme { return &appTheme{} }

func (t *appTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	if name == theme.ColorNamePrimary {
		return color.NRGBA{R: 0, G: 122, B: 255, A: 255}
	}
	return theme.DefaultTheme().Color(name, variant)
}

func (t *appTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (t *appTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *appTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}
