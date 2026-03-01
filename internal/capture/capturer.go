package capture

import (
	"bytes"
	"fmt"
	"image/jpeg"
	"time"

	"activity-tracker/internal/window"

	"github.com/kbinani/screenshot"
)

type Screenshot struct {
	ImageData    []byte
	Timestamp    time.Time
	ActiveWindow *window.ActiveWindow
}

type Capturer struct {
	quality int
}

func New(quality int) *Capturer {
	return &Capturer{
		quality: quality,
	}
}

func (c *Capturer) Capture() (*Screenshot, error) {
	// Get active window info
	activeWin, err := window.GetActive()
	if err != nil {
		// Log warning but continue
		activeWin = nil
	}

	// Capture screenshot from primary display
	bounds := screenshot.GetDisplayBounds(0)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return nil, fmt.Errorf("capture failed: %w", err)
	}

	// Compress to JPEG
	var buf bytes.Buffer
	opts := &jpeg.Options{Quality: c.quality}
	if err := jpeg.Encode(&buf, img, opts); err != nil {
		return nil, fmt.Errorf("encode failed: %w", err)
	}

	return &Screenshot{
		ImageData:    buf.Bytes(),
		Timestamp:    time.Now(),
		ActiveWindow: activeWin,
	}, nil
}
