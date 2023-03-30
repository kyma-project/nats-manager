package file

import "os"

func DirExists(path string) bool {
	if path == "" {
		return false
	}
	stats, err := os.Stat(path)
	return !os.IsNotExist(err) && (stats != nil && stats.IsDir())
}
