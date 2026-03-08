package cricket

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"os/exec"
	"strings"
)

// OCRClient interface for different OCR implementations
type OCRClient interface {
	ExtractText(img *image.RGBA) (string, error)
	Close() error
}

// WinOCRClient uses Windows native OCR via PowerShell bridge
// This is the same OCR engine used by Windows Snipping Tool
// Much more robust for game HUDs with transparency and gradients than Tesseract
type WinOCRClient struct {
	usePreprocessing bool
}

// NewOCRClient creates a new Windows OCR client
func NewOCRClient() OCRClient {
	log.Println("Initializing Windows Native OCR (same engine as Snipping Tool)")
	return &WinOCRClient{
		usePreprocessing: false, // Try without preprocessing first
	}
}

// preprocessImage converts the capture to grayscale and increases contrast
// Optional preprocessing - WinOCR often works better with original images
func (o *WinOCRClient) preprocessImage(img *image.RGBA) *image.Gray {
	bounds := img.Bounds()
	grayImg := image.NewGray(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			oldColor := img.At(x, y)
			grayColor := color.GrayModel.Convert(oldColor).(color.Gray)

			// Multi-level thresholding to preserve different text shades
			if grayColor.Y > 180 {
				grayImg.SetGray(x, y, color.Gray{Y: 255})
			} else if grayColor.Y > 100 {
				grayImg.SetGray(x, y, color.Gray{Y: 220})
			} else if grayColor.Y > 50 {
				grayImg.SetGray(x, y, color.Gray{Y: 180})
			} else {
				grayImg.SetGray(x, y, color.Gray{Y: 0})
			}
		}
	}

	return grayImg
}

// ExtractText performs OCR using Windows native OCR engine
func (o *WinOCRClient) ExtractText(img *image.RGBA) (string, error) {
	// Create temporary file for image
	tmpFile, err := os.CreateTemp("", "cricket-winocr-*.png")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close() // Close immediately so os.Remove works on Windows
	defer os.Remove(tmpPath)

	// Save image (with optional preprocessing)
	var imgToSave image.Image = img
	if o.usePreprocessing {
		imgToSave = o.preprocessImage(img)
	}

	f, err := os.Create(tmpPath)
	if err != nil {
		return "", fmt.Errorf("failed to create image file: %w", err)
	}

	if err := png.Encode(f, imgToSave); err != nil {
		f.Close()
		return "", fmt.Errorf("failed to encode image: %w", err)
	}
	f.Close()

	// Create PowerShell script file to avoid escaping issues
	psScriptPath := tmpPath + ".ps1"
	defer os.Remove(psScriptPath)

	// PowerShell script using Windows.Media.Ocr API
	psScript := fmt.Sprintf(`Add-Type -AssemblyName System.Runtime.WindowsRuntime

$asTaskGeneric = ([System.WindowsRuntimeSystemExtensions].GetMethods() | Where-Object { 
	$_.Name -eq 'AsTask' -and 
	$_.GetParameters().Count -eq 1 -and 
	$_.GetParameters()[0].ParameterType.Name -eq 'IAsyncOperation%s1' 
})[0]

Function Await($WinRtTask, $ResultType) {
	$asTask = $asTaskGeneric.MakeGenericMethod($ResultType)
	$netTask = $asTask.Invoke($null, @($WinRtTask))
	$netTask.Wait(-1) | Out-Null
	$netTask.Result
}

[Windows.Storage.StorageFile,Windows.Storage,ContentType=WindowsRuntime] | Out-Null
[Windows.Media.Ocr.OcrEngine,Windows.Foundation,ContentType=WindowsRuntime] | Out-Null
[Windows.Foundation.IAsyncOperation%s1,Windows.Foundation,ContentType=WindowsRuntime] | Out-Null
[Windows.Graphics.Imaging.BitmapDecoder,Windows.Graphics,ContentType=WindowsRuntime] | Out-Null

$path = "%s"
$file = Await ([Windows.Storage.StorageFile]::GetFileFromPathAsync($path)) ([Windows.Storage.StorageFile])
$stream = Await ($file.OpenAsync([Windows.Storage.FileAccessMode]::Read)) ([Windows.Storage.Streams.IRandomAccessStream])
$decoder = Await ([Windows.Graphics.Imaging.BitmapDecoder]::CreateAsync($stream)) ([Windows.Graphics.Imaging.BitmapDecoder])
$bitmap = Await ($decoder.GetSoftwareBitmapAsync()) ([Windows.Graphics.Imaging.SoftwareBitmap])

$engine = [Windows.Media.Ocr.OcrEngine]::TryCreateFromUserProfileLanguages()
$result = Await ($engine.RecognizeAsync($bitmap)) ([Windows.Media.Ocr.OcrResult])

Write-Output $result.Text
`, "`", "`", tmpPath)

	// Write PowerShell script to file
	if err := os.WriteFile(psScriptPath, []byte(psScript), 0644); err != nil {
		return "", fmt.Errorf("failed to write PowerShell script: %w", err)
	}

	// Execute PowerShell script
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", psScriptPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("WinOCR failed: %w, output: %s", err, string(output))
	}

	text := strings.TrimSpace(string(output))
	// Log removed - will be logged in tracker to avoid duplicate logs

	return text, nil
}

// Close releases OCR resources
func (o *WinOCRClient) Close() error {
	return nil
}
