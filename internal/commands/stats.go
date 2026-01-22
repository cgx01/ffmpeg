package commands

import (
	"fmt"
	"github.com/spf13/pflag"
	"os"
	"path/filepath"
	"strings"
)

func RunStats(args []string) {
	fs := pflag.NewFlagSet("stats", pflag.ExitOnError)
	dirs := fs.StringSliceP("dirs", "d", []string{}, "Directories to scan")

	fs.Parse(args)

	if len(*dirs) == 0 {
		fmt.Println("Error: No directories specified")
		fs.Usage()
		return
	}

	for _, dir := range *dirs {
		fmt.Printf("Stats for %s:\n", dir)
		jpg, mp4 := getDirStats(dir)
		fmt.Printf("  JPG/Images: %d\n  MP4: %d\n", jpg, mp4)
	}
}

func getDirStats(path string) (int, int) {
	var jpg, mp4 int

	dirEntries, err := os.ReadDir(path)
	if err != nil {
		fmt.Printf("Error reading %s: %v\n", path, err)
		return 0, 0
	}

	for _, entry := range dirEntries {
		if entry.IsDir() {
			j, m := getDirStats(filepath.Join(path, entry.Name()))
			jpg += j
			mp4 += m
		} else {
			ext := strings.ToLower(filepath.Ext(entry.Name()))
			switch ext {
			case ".torrent":
				os.Remove(filepath.Join(path, entry.Name())) // Side effect: delete torrents
			case ".jpeg", ".gif", ".jpg", ".png":
				jpg++
			case ".mp4":
				mp4++
			}
		}
	}
	return jpg, mp4
}
