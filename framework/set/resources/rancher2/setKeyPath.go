package rancher2

import (
	"os"
	"path/filepath"
)

// SetKeyPath is a function that will set the path to the key file.
func SetKeyPath(keyPath string) string {
	userDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	keyPath = filepath.Join(userDir, keyPath)

	return keyPath
}
