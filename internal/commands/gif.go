package commands

import (
	"ffmpeg/util"
	"fmt"
	"github.com/spf13/pflag"
	"os"
	"path/filepath"
	"strings"
)

func RunGif(args []string) {
	fs := pflag.NewFlagSet("gif", pflag.ExitOnError)
	dirPath := fs.StringP("dir", "d", "", "Directory to process")
	maxSize := fs.StringP("size", "s", "9M", "Max size for GIF")
	
	fs.Parse(args)

	if *dirPath == "" {
		fmt.Println("Error: --dir is required")
		fs.Usage()
		return
	}

	fmt.Printf("Processing GIFs in %s (Max size: %s)\n", *dirPath, *maxSize)
	compressGIF(*dirPath, *maxSize)
}

func compressGIF(dirPath, maxSize string) {
	dirEntries, err := os.ReadDir(dirPath)
	if err != nil {
		fmt.Printf("Error reading dir: %v\n", err)
		return
	}

	for _, entry := range dirEntries {
		if !entry.IsDir() {
			inputFile := filepath.Join(dirPath, entry.Name())
			ext := strings.ToLower(filepath.Ext(entry.Name()))

			if ext == ".gif" {
				outputFile := filepath.Join(dirPath, "out.gif")
				fmt.Printf("Compressing GIF: %s\n", inputFile)
				if err := util.CompressGif(inputFile, outputFile, maxSize, false); err != nil {
					fmt.Printf("Error compressing %s: %v\n", inputFile, err)
					continue
				}
				os.Remove(inputFile)
				os.Rename(outputFile, inputFile)
			} else if ext == ".mp4" {
				baseName := strings.TrimSuffix(entry.Name(), ext)
				outputFile := filepath.Join(dirPath, baseName+".gif")
				fmt.Printf("Converting MP4 to GIF: %s\n", inputFile)
				if err := util.CompressGif(inputFile, outputFile, maxSize, true); err != nil {
					fmt.Printf("Error converting %s: %v\n", inputFile, err)
					continue
				}
				os.Remove(inputFile)
			}
		}
	}
}
