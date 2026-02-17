package art

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	xdraw "golang.org/x/image/draw"
)

// Composer handles image download and composition for game art
type Composer struct {
	cacheDir string
	logger   *slog.Logger
	client   *http.Client
}

// NewComposer creates a new art composer
func NewComposer(cacheDir string, logger *slog.Logger) *Composer {
	if logger == nil {
		logger = slog.Default()
	}

	return &Composer{
		cacheDir: cacheDir,
		logger:   logger,
		client:   &http.Client{Timeout: 30 * time.Second},
	}
}

// ComposeHeader creates a 460x215 header image:
// - Background: Screenshot (scaled/cropped to fill)
// - Overlay: Logo (centered, max 50% width, preserve aspect ratio)
// Falls back to cover art if no logo, or artwork/cover if no screenshot
func (c *Composer) ComposeHeader(screenshotURL, logoURL, coverURL, artworkURL, gameID string) ([]byte, error) {
	// Target dimensions (Steam header size)
	targetWidth, targetHeight := 460, 215

	var backgroundImg image.Image
	var backgroundSource string

	// Try screenshot first for background
	if screenshotURL != "" {
		img, err := c.downloadImage(screenshotURL)
		if err != nil {
			c.logger.Warn("failed to download screenshot for header", "error", err, "gameID", gameID)
		} else {
			backgroundImg = img
			backgroundSource = "screenshot"
		}
	}

	// Fallback to artwork
	if backgroundImg == nil && artworkURL != "" {
		img, err := c.downloadImage(artworkURL)
		if err != nil {
			c.logger.Warn("failed to download artwork for header", "error", err, "gameID", gameID)
		} else {
			backgroundImg = img
			backgroundSource = "artwork"
		}
	}

	// Final fallback to cover
	if backgroundImg == nil && coverURL != "" {
		img, err := c.downloadImage(coverURL)
		if err != nil {
			c.logger.Warn("failed to download cover for header", "error", err, "gameID", gameID)
		} else {
			backgroundImg = img
			backgroundSource = "cover"
		}
	}

	if backgroundImg == nil {
		return nil, fmt.Errorf("no background image available for header composition")
	}

	c.logger.Debug("using background for header", "gameID", gameID, "source", backgroundSource)

	// Create target canvas
	canvas := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))

	// Scale background to cover canvas (showing more of the image, like CSS background-size: cover)
	scaledBg := c.scaleToCover(backgroundImg, targetWidth, targetHeight)
	draw.Draw(canvas, canvas.Bounds(), scaledBg, image.Point{}, draw.Src)

	// Try to overlay logo
	if logoURL != "" {
		logoImg, err := c.downloadImage(logoURL)
		if err != nil {
			c.logger.Warn("failed to download logo for header", "error", err, "gameID", gameID)
		} else {
			// Scale logo to max 50% width while preserving aspect ratio
			maxLogoWidth := int(float32(targetWidth) * .6)
			scaledLogo := c.scalePreserveAspect(logoImg, maxLogoWidth, targetHeight)

			// Center the logo
			logoBounds := scaledLogo.Bounds()
			x := (targetWidth - logoBounds.Dx()) / 2
			y := (targetHeight - logoBounds.Dy()) / 2
			centerPoint := image.Point{X: x, Y: y}

			// Draw logo with alpha blending
			draw.Draw(canvas, logoBounds.Add(centerPoint), scaledLogo, image.Point{}, draw.Over)
			c.logger.Debug("composed logo onto header", "gameID", gameID)
		}
	} 

	// Encode as PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, canvas); err != nil {
		return nil, fmt.Errorf("failed to encode header image: %w", err)
	}

	return buf.Bytes(), nil
}

// DownloadArt downloads art from URL and returns the image
func (c *Composer) DownloadArt(url string) ([]byte, string, error) {
	return c.downloadImageBytes(url)
}

// CacheArt saves art to the cache directory
func (c *Composer) CacheArt(source string, instanceID, artType string, data []byte) error {
	artDir := filepath.Join(c.cacheDir, source, instanceID)
	if err := os.MkdirAll(artDir, 0755); err != nil {
		return fmt.Errorf("failed to create art cache directory: %w", err)
	}

	artPath := filepath.Join(artDir, artType+".png")
	if err := os.WriteFile(artPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write art cache: %w", err)
	}

	return nil
}

// GetCachedArt retrieves cached art if it exists
func (c *Composer) GetCachedArt(source string, instanceID, artType string) ([]byte, error) {
	artPath := filepath.Join(c.cacheDir, source, instanceID, artType+".png")
	data, err := os.ReadFile(artPath)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// HasCachedArt checks if art exists in cache
func (c *Composer) HasCachedArt(source string, instanceID, artType string) bool {
	artPath := filepath.Join(c.cacheDir, source, instanceID, artType+".png")
	_, err := os.Stat(artPath)
	return err == nil
}

// DownloadAllArt downloads all art types concurrently
func (c *Composer) DownloadAllArt(artURLs map[string]string) map[string][]byte {
	results := make(map[string][]byte)
	var mu sync.Mutex
	var wg sync.WaitGroup

	artTypes := []string{"screenshot", "logo", "cover", "artwork"}
	c.logger.Info(fmt.Sprintf("Downloading artURLs: %#v", artURLs))

	for _, artType := range artTypes {
		if url, ok := artURLs[artType]; ok && url != "" {
			wg.Add(1)
			go func(t, u string) {
				defer wg.Done()
				data, _, err := c.downloadImageBytes(u)
				if err != nil {
					c.logger.Warn("failed to download art", "type", t, "error", err)
					return
				}
				mu.Lock()
				results[t] = data
				mu.Unlock()
			}(artType, url)
		}
	}

	wg.Wait()
	return results
}

// downloadImage downloads and decodes an image from URL
func (c *Composer) downloadImage(url string) (image.Image, error) {
	data, format, err := c.downloadImageBytes(url)
	if err != nil {
		return nil, err
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode %s image: %w", format, err)
	}

	return img, nil
}

// downloadImageBytes downloads image bytes and detects format
func (c *Composer) downloadImageBytes(url string) ([]byte, string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("image download returned status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read image data: %w", err)
	}

	// Detect format from content type or data
	contentType := resp.Header.Get("Content-Type")
	var format string
	switch contentType {
	case "image/png":
		format = "png"
	case "image/jpeg", "image/jpg":
		format = "jpeg"
	default:
		// Try to detect from data
		if len(data) > 0 {
			if data[0] == 0x89 && data[1] == 0x50 {
				format = "png"
			} else if data[0] == 0xFF && data[1] == 0xD8 {
				format = "jpeg"
			}
		}
	}

	return data, format, nil
}

// scalePreserveAspect scales image preserving aspect ratio, fitting within max dimensions
func (c *Composer) scalePreserveAspect(src image.Image, maxWidth, maxHeight int) image.Image {
	srcBounds := src.Bounds()
	srcWidth := srcBounds.Dx()
	srcHeight := srcBounds.Dy()

	// Calculate scale to fit within max dimensions
	scaleX := float64(maxWidth) / float64(srcWidth)
	scaleY := float64(maxHeight) / float64(srcHeight)

	// Use smaller scale to fit
	scale := scaleX
	if scaleY < scaleX {
		scale = scaleY
	}

	newWidth := int(float64(srcWidth) * scale)
	newHeight := int(float64(srcHeight) * scale)

	if newWidth == 0 {
		newWidth = 1
	}
	if newHeight == 0 {
		newHeight = 1
	}

	// Create scaled image with RGBA to preserve alpha
	dst := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
	// Use Src to preserve source alpha, not Over which composites on existing dst
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, srcBounds, draw.Src, nil)

	return dst
}

// scaleToCover scales image to cover target dimensions (like CSS background-size: cover)
// The image is scaled to completely cover the target, maintaining aspect ratio
// Any excess is cropped from edges to center the result
func (c *Composer) scaleToCover(src image.Image, targetWidth, targetHeight int) image.Image {
	srcBounds := src.Bounds()
	srcWidth := srcBounds.Dx()
	srcHeight := srcBounds.Dy()

	// Calculate scale factors
	scaleX := float64(targetWidth) / float64(srcWidth)
	scaleY := float64(targetHeight) / float64(srcHeight)

	// Use larger scale to cover (same as scaleAndCrop but centers differently)
	scale := scaleX
	if scaleY > scaleX {
		scale = scaleY
	}

	newWidth := int(float64(srcWidth) * scale)
	newHeight := int(float64(srcHeight) * scale)

	// Create scaled image
	dst := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, srcBounds, draw.Src, nil)

	// Crop to target size (center)
	if newWidth > targetWidth || newHeight > targetHeight {
		x := (newWidth - targetWidth) / 2
		y := (newHeight - targetHeight) / 2
		cropped := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))
		draw.Draw(cropped, cropped.Bounds(), dst, image.Point{X: x, Y: y}, draw.Src)
		return cropped
	}

	return dst
}
