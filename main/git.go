package main

import (
	"fmt"
	"godemo/util"
	"os"
	"path/filepath"
	"strings"
	// 导入支持其他图像格式的包
	_ "golang.org/x/image/webp"
)

func main() {
	for _, g := range []string{"C:\\Users\\Administrator\\Desktop\\新建文件夹"} {
		compressGIF(g)
	}
}
func compressGIF(file string) {
	var inputFile, outputFile string
	dir, _ := os.ReadDir(file)
	for _, d := range dir {
		if !d.IsDir() {
			inputFile = filepath.Join(file, d.Name())
			if filepath.Ext(d.Name()) == ".gif" {
				outputFile = filepath.Join(file, "out.gif")
				if err := util.CompressGif(inputFile, outputFile, "9M", false); err != nil {
					fmt.Printf("%v\n", err)
					continue
				}
				os.Remove(inputFile)
				os.Rename(outputFile, inputFile)
			} else if filepath.Ext(d.Name()) == ".mp4" {
				baseName := strings.TrimSuffix(d.Name(), ".mp4")
				outputFile = filepath.Join(file, baseName+".gif")
				if err := util.CompressGif(inputFile, outputFile, "9M", true); err != nil {
					fmt.Printf("%v\n", err)
					continue
				}
				os.Remove(inputFile)
			}
		}
	}
}
