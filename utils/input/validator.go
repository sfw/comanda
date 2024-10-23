package input

import (
	"fmt"
	"os"
	"path/filepath"
)

// Validator handles input validation
type Validator struct {
	allowedExtensions []string
}

// NewValidator creates a new input validator
func NewValidator(allowedExtensions []string) *Validator {
	return &Validator{
		allowedExtensions: allowedExtensions,
	}
}

// ValidatePath checks if a path exists and is accessible
func (v *Validator) ValidatePath(path string) error {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("path does not exist: %s", path)
		}
		return fmt.Errorf("error accessing path %s: %w", path, err)
	}
	return nil
}

// ValidateFileExtension checks if a file has an allowed extension
func (v *Validator) ValidateFileExtension(path string) error {
	if len(v.allowedExtensions) == 0 {
		return nil
	}

	ext := filepath.Ext(path)
	for _, allowedExt := range v.allowedExtensions {
		if ext == allowedExt {
			return nil
		}
	}
	return fmt.Errorf("invalid file extension for %s. Allowed extensions: %v", path, v.allowedExtensions)
}

// ValidateDirectoryAccess checks if a directory is readable
func (v *Validator) ValidateDirectoryAccess(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("error accessing directory %s: %w", path, err)
	}
	defer file.Close()

	_, err = file.Readdir(1)
	if err != nil {
		return fmt.Errorf("error reading directory %s: %w", path, err)
	}
	return nil
}
