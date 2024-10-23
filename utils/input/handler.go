package input

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

// InputType represents the type of input being processed
type InputType int

const (
	FileInput InputType = iota
	DirectoryInput
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

// ProcessPath handles both file and directory inputs
func (h *Handler) ProcessPath(path string) error {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("error accessing path %s: %w", path, err)
	}

	if fileInfo.IsDir() {
		return h.processDirectory(path)
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
		if input.Type == FileInput {
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
