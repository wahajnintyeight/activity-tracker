package cricket

import (
	"image"

	"github.com/kbinani/screenshot"
)

// CaptureScoreboardArea captures a specific region of the screen
// This should be the scoreboard area (usually bottom-left or bottom-center)
func CaptureScoreboardArea(rect image.Rectangle) (*image.RGBA, error) {
	img, err := screenshot.CaptureRect(rect)
	if err != nil {
		return nil, err
	}
	return img, nil
}

// GetScoreboardRect returns the default scoreboard coordinates
// These should be configurable via environment variables
// Default assumes 1920x1080 resolution with scoreboard at bottom-left
func GetScoreboardRect(x, y, width, height int) image.Rectangle {
	return image.Rect(x, y, x+width, y+height)
}
