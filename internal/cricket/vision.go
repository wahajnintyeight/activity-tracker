package cricket

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"time"
)

type StrikerPosition struct {
	IsStriker bool
	Bounds    image.Rectangle
}

// DetectStriker scans a specific vertical strip of the cropped image for a cluster of bright pixels.
func DetectStriker(img *image.RGBA, xRange [2]int, yRange [2]int, side string) bool {
	// Create debug crop for this zone
	debugDir := "debug_zones"
	if _, err := os.Stat(debugDir); os.IsNotExist(err) {
		os.Mkdir(debugDir, 0755)
	}

	// Define the rectangle for the zone
	rect := image.Rect(xRange[0], yRange[0], xRange[1], yRange[1])
	subImg := img.SubImage(rect).(*image.RGBA)

	// Save the zone image
	debugPath := fmt.Sprintf("%s/zone-%s-%d.png", debugDir, side, time.Now().UnixNano()/1e6)
	if f, err := os.Create(debugPath); err == nil {
		png.Encode(f, subImg)
		f.Close()
	}

	brightPixelThreshold := 0.85
	requiredCount := 20
	foundCount := 0

	for y := yRange[0]; y < yRange[1]; y++ {
		for x := xRange[0]; x < xRange[1]; x++ {
			c := img.At(x, y)
			r, g, b, _ := c.RGBA()

			// Calculate relative brightness
			brightness := (float64(r)/65535.0 + float64(g)/65535.0 + float64(b)/65535.0) / 3.0

			if brightness > brightPixelThreshold {
				foundCount++
			}
		}
	}

	return foundCount >= requiredCount
}
