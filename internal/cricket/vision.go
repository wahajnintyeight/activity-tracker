package cricket

import (
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"time"
)

type StrikerPosition struct {
	IsStriker bool
	Bounds    image.Rectangle
}

type StrikerDetector struct {
	BrightThreshold float64
	MinBrightRatio  float64
	MinColumnsOn    float64
}

func DefaultStrikerDetector() StrikerDetector {
	return StrikerDetector{
		BrightThreshold: 0.82, // Lowered from 0.90 to catch off-white markers
		MinBrightRatio:  0.03, // Lowered from 0.05
		MinColumnsOn:    0.12, // Lowered from 0.17
	}
}

// DetectStriker is the entry point for striker detection on a zone strip.
// It specifically looks for the bright triangular marker in the left gutter.
func DetectStriker(zone *image.RGBA, side string, debug bool) bool {
	ts := time.Now().UnixNano() / 1e6
	if debug {
		// Create debug crop for this zone
		debugDir := "debug_zones"
		if _, err := os.Stat(debugDir); os.IsNotExist(err) {
			os.Mkdir(debugDir, 0755)
		}

		// Save the zone image for visual verification
		debugPath := fmt.Sprintf("%s/zone-%s-%d.png", debugDir, side, ts)
		if f, err := os.Create(debugPath); err == nil {
			png.Encode(f, zone)
			f.Close()
		}
	}

	detector := DefaultStrikerDetector()
	return detector.HasStrikerMarker(zone, side, debug, ts)
}

func (d StrikerDetector) HasStrikerMarker(zone *image.RGBA, side string, debug bool, ts int64) bool {

	if zone == nil {
		return false
	}

	b := zone.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return false
	}

	// markerW: scan the leftmost portion where the marker is expected
	markerW := clampInt(int(math.Round(float64(w)*0.18)), 10, 45) // Increased max from 28 to 45
	print("[Marker W] : ", markerW)
	// padding: ignore top/bottom edges to avoid HUD border noise
	padTop := clampInt(int(math.Round(float64(h)*0.10)), 0, h) // Reduced padding from 18% to 10%
	padBot := clampInt(int(math.Round(float64(h)*0.10)), 0, h) // Reduced padding from 18% to 10%

	y0 := b.Min.Y + padTop
	y1 := b.Max.Y - padBot
	if y1 <= y0 {
		return false
	}

	x0 := b.Min.X
	x1 := b.Min.X + markerW
	if x1 > b.Max.X {
		x1 = b.Max.X
	}

	// Create and save a debug crop of ONLY the gutter area being scanned
	if debug {
		gutterRect := image.Rect(x0, y0, x1, y1)
		gutterImg := zone.SubImage(gutterRect).(*image.RGBA)
		gutterPath := fmt.Sprintf("%s/gutter-%s-%d.png", "debug_zones", side, ts)
		if f, err := os.Create(gutterPath); err == nil {
			png.Encode(f, gutterImg)
			f.Close()
		}
	}

	total := 0
	bright := 0
	colsOn := 0

	for x := x0; x < x1; x++ {
		colTotal := 0
		colBright := 0

		for y := y0; y < y1; y++ {
			r, g, bb, _ := zone.At(x, y).RGBA()
			// Calculate relative brightness (0.0 to 1.0)
			br := (float64(r) + float64(g) + float64(bb)) / (3.0 * 65535.0)

			colTotal++
			total++
			if br >= d.BrightThreshold {
				colBright++
				bright++
			}
		}

		// A column is "on" if it contains a significant vertical segment of bright pixels
		if colTotal > 0 && float64(colBright)/float64(colTotal) >= 0.12 {
			colsOn++
		}
	}

	if total == 0 {
		return false
	}

	brightRatio := float64(bright) / float64(total)
	colsOnRatio := float64(colsOn) / float64(maxInt(1, x1-x0))

	isDetected := brightRatio >= d.MinBrightRatio && colsOnRatio >= d.MinColumnsOn

	// Log the detection metrics for debugging
	debugSuffix := ""
	if debug {
		debugSuffix = fmt.Sprintf(" | Zone: zone-%s-%d.png", side, ts)
	}
	fmt.Printf("[Striker Check] Side: %s | BrightRatio: %.3f (min: %.2f) | ColsOnRatio: %.3f (min: %.2f) | Detected: %v%s\n",
		side, brightRatio, d.MinBrightRatio, colsOnRatio, d.MinColumnsOn, isDetected, debugSuffix)

	return isDetected
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
