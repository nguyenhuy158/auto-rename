// File system helpers for auto-rename
package infrastructure

import (
	"os"
)

func GetFileInfo(path string) (int64, string, string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, "", "", err
	}
	return info.Size(), info.Mode().String(), info.ModTime().Format("2006-01-02 15:04:05"), nil
}
