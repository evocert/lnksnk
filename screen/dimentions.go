package screen

import (
	"github.com/fstanis/screenresolution"
)

// Size returns the width and height of the terminal screen
func Size() (w int, h int) {
	resolution := screenresolution.GetPrimary()
	screenresolution.GetPrimary()
	w, h = resolution.Width, resolution.Height
	return w, h
}
