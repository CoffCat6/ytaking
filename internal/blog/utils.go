package blog

import (
	"fmt"
	"os"
	"path/filepath"
)

// atomicWriteFile writes data to a temporary file first, then renames it to the target file.
// This ensures that the target file is not corrupted if the write fails or the program crashes.
func atomicWriteFile(filename string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	// Create a temporary file in the same directory
	tmpFile, err := os.CreateTemp(dir, filepath.Base(filename)+".*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	
	// Ensure we close and remove the temp file if something goes wrong
	closed := false
	defer func() {
		if !closed {
			_ = tmpFile.Close()
			_ = os.Remove(tmpPath)
		}
	}()

	// Write data
	if _, err := tmpFile.Write(data); err != nil {
		return err
	}

	// Sync to disk
	if err := tmpFile.Sync(); err != nil {
		return err
	}

	// Close the file explicitly
	if err := tmpFile.Close(); err != nil {
		return err
	}
	closed = true

	// Atomic rename
	if err := os.Rename(tmpPath, filename); err != nil {
		// If rename fails, we should remove the temp file (handled by defer if closed is false, 
		// but here closed is true, so we might need to remove it manually if we want to be strict,
		// though usually rename failure leaves tmp file. Let's just return error.)
		_ = os.Remove(tmpPath) // cleanup
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}
