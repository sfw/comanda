package input

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/jpeg" // Register JPEG format
	"image/png"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/kbinani/screenshot"
)

// InputType represents the type of input being processed
type InputType int

const (
	FileInput InputType = iota
	DirectoryInput
	ScreenshotInput
	ImageInput
)

// Input represents a file or directory to be processed
type Input struct {
	Path     string
	Type     InputType
	Contents []byte
}

// Handler processes input files and directories
type Handler struct {
	inputs []*Input
}

// NewHandler creates a new input handler
func NewHandler() *Handler {
	return &Handler{
		inputs: make([]*Input, 0),
	}
}

// isImageFile checks if the file is an image based on extension
func (h *Handler) isImageFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	imageExts := map[string]bool{
		".png":  true,
		".jpg":  true,
		".jpeg": true,
		".gif":  true,
		".bmp":  true,
	}
	return imageExts[ext]
}

// ProcessPath handles both file and directory inputs
func (h *Handler) ProcessPath(path string) error {
	if path == "screenshot" {
		return h.processScreenshot()
	}

	fileInfo, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("error accessing path %s: %w", path, err)
	}

	if fileInfo.IsDir() {
		return h.processDirectory(path)
	}

	if h.isImageFile(path) {
		return h.processImage(path)
	}
	return h.processFile(path)
}

// processFile handles single file input
func (h *Handler) processFile(path string) error {
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("error reading file %s: %w", path, err)
	}

	input := &Input{
		Path:     path,
		Type:     FileInput,
		Contents: contents,
	}
	h.inputs = append(h.inputs, input)
	return nil
}

// processImage handles image file input
func (h *Handler) processImage(path string) error {
	// Read the image file
	imgFile, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("error opening image %s: %w", path, err)
	}
	defer imgFile.Close()

	// Decode the image
	img, _, err := image.Decode(imgFile)
	if err != nil {
		return fmt.Errorf("error decoding image %s: %w", path, err)
	}

	// Create a buffer to store PNG data
	var buf bytes.Buffer

	// Encode image to PNG (standardize format for vision models)
	if err := png.Encode(&buf, img); err != nil {
		return fmt.Errorf("failed to encode image: %w", err)
	}

	// Encode PNG data to base64
	base64Data := base64.StdEncoding.EncodeToString(buf.Bytes())

	input := &Input{
		Path:     path,
		Type:     ImageInput,
		Contents: []byte(base64Data),
	}
	h.inputs = append(h.inputs, input)
	return nil
}

// processDirectory handles directory input
func (h *Handler) processDirectory(path string) error {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return fmt.Errorf("error reading directory %s: %w", path, err)
	}

	for _, file := range files {
		fullPath := filepath.Join(path, file.Name())
		if err := h.ProcessPath(fullPath); err != nil {
			return err
		}
	}
	return nil
}

// processScreenshot captures a screenshot in XGA resolution (1024x768)
func (h *Handler) processScreenshot() error {
	// Capture the primary display
	bounds := screenshot.GetDisplayBounds(0)

	// Create a new bounds with XGA resolution (1024x768)
	bounds.Max.X = bounds.Min.X + 1024
	bounds.Max.Y = bounds.Min.Y + 768

	// Capture the screen
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return fmt.Errorf("failed to capture screenshot: %w", err)
	}

	// Create a buffer to store PNG data
	var buf bytes.Buffer

	// Encode image to PNG
	if err := png.Encode(&buf, img); err != nil {
		return fmt.Errorf("failed to encode screenshot: %w", err)
	}

	// Encode PNG data to base64
	base64Data := base64.StdEncoding.EncodeToString(buf.Bytes())

	input := &Input{
		Path:     "screenshot",
		Type:     ScreenshotInput,
		Contents: []byte(base64Data),
	}
	h.inputs = append(h.inputs, input)
	return nil
}

// GetInputs returns all processed inputs
func (h *Handler) GetInputs() []*Input {
	return h.inputs
}

// GetFileContents returns the contents of a specific file
func (h *Handler) GetFileContents(path string) ([]byte, error) {
	for _, input := range h.inputs {
		if input.Path == path {
			return input.Contents, nil
		}
	}
	return nil, fmt.Errorf("file %s not found in processed inputs", path)
}

// GetAllContents returns all file contents concatenated
func (h *Handler) GetAllContents() []byte {
	var allContents []byte
	for _, input := range h.inputs {
		if input.Type == FileInput || input.Type == ScreenshotInput || input.Type == ImageInput {
			allContents = append(allContents, input.Contents...)
			allContents = append(allContents, '\n')
		}
	}
	return allContents
}

// Clear removes all processed inputs
func (h *Handler) Clear() {
	h.inputs = make([]*Input, 0)
}
