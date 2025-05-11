package main

import (
	"os"
	"path/filepath"
)

func main() {
	for _, arg := range []string{
		//"G:\\欧美新系列⭐WIFEY淫妻",
		"G:\\",
	} {
		removeFile(arg)
	}

}

func removeFile(path string) {
	dir, _ := os.ReadDir(path)
	var isRemove bool
	for _, entry := range dir {
		if entry.IsDir() {
			removeFile(filepath.Join(path, entry.Name()))
		} else {
			_, err := os.Stat(filepath.Join(path, "a.txt"))
			if !os.IsNotExist(err) {
				isRemove = true
			}
			if isRemove && filepath.Ext(entry.Name()) == ".torrent" {
				os.Remove(filepath.Join(path, entry.Name()))
			}
		}
	}
}
