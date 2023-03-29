package file

import "os"

func DirExists(file string) bool {
	if file == "" {
		return false
	}
	stats, err := os.Stat(file)
	return !os.IsNotExist(err) && (stats != nil && stats.IsDir())
}
