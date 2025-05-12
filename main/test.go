package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	for _, arg := range []string{
		"G:\\BaiduNetdiskDownload\\AngelaWhite",
	} {
		getDir(arg)
	}

}

var (
	jpg, mp4 int
)

func getDir(path string) {
	dir, _ := os.ReadDir(path)
	for _, entry := range dir {
		if entry.IsDir() {
			getDir(filepath.Join(path, entry.Name()))
		} else {
			switch filepath.Ext(entry.Name()) {
			case ".torrent":
				os.Remove(filepath.Join(path, entry.Name()))
				break
			case ".jpeg":
				fallthrough
			case ".gif":
				fallthrough
			case ".jpg":
				fallthrough
			case ".png":
				jpg++
			case ".mp4":
				mp4++
			}
		}
	}
	fmt.Printf("JPG: %d\tMP4: %d\n", jpg, mp4)
}
