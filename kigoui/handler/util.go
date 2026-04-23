package kigoui

import (
	"github.com/kbinani/screenshot"
)

func GetScreenDimensions() (int, int) {
	bounds := screenshot.GetDisplayBounds(screenshot.NumActiveDisplays())
	return bounds.Dx(), bounds.Dy()
}
