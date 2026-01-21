package main

import (
	"ffmpeg/util"
	"fmt"
	"github.com/spf13/pflag"
	"os"
	"path/filepath"
	"strings"
)

type config struct {
	dirs *[]string
}

func (c *config) bindFlags() {
	c.dirs = pflag.StringSliceP("dirs", "d", []string{}, "需要处理的目录（多个目录用逗号分隔或多次指定）")
	//pflag.StringVarP(&c.dirs, "dirs", "d", "", "需要处理的目录（多个目录用空格分隔）")
}

func main() {
	var c config
	c.bindFlags()
	pflag.Parse()
	if len(*c.dirs) == 0 {
		panic("目录位置为空")
	}
	fmt.Printf("%v\n", *c.dirs)

	for _, dir := range *c.dirs {
		if err := parseDirsVidInfo(dir); err != nil {
			panic(err)
		}
	}
	//fmt.Println(num)

}

//func main() {
//	err := os.Rename("F:\\迅雷下载\\mollyflwers\\Blacked.25.11.19.Blake.Blossom.Sky.Wonderland.And.Scarlet.Skies.Hot.Girl.Trio.Blake.Sky.And.Scarlet.Have.Crazy.BBC.Orgy.XXX.1080p.MP4-P2P[XC]\\blacked.25.11.19.blake.blossom.sky.wonderland.and.scarlet.skies.hot.girl.trio.blake.sky.and.scarlet.have.crazy.bbc.orgy.xxx(1).srt",
//		"F:\\迅雷下载\\mollyflwers\\Blacked.25.11.19.Blake.Blossom.Sky.Wonderland.And.Scarlet.Skies.Hot.Girl.Trio.Blake.Sky.And.Scarlet.Have.Crazy.BBC.Orgy.XXX.1080p.MP4-P2P[XC]\\blacked.25.11.19.blake.blossom.sky.wonderland.and.scarlet.skies..srt")
//	if err != nil {
//		panic(err)
//	}
//}

// 定义需要排除的目录(绝对路径，支持跨平台路径格式)
var excludedDirs = map[string]bool{}
var num int

func parseDirsVidInfo(dir string) error {
	//targetTime, _ := time.Parse(time.DateTime, "2026-01-17 22:00:00")
	// 替代原有的 time.Parse
	//targetTime, _ := time.ParseInLocation(time.DateTime, "2026-01-17 22:00:00", time.Local)
	dirEntries, _ := os.ReadDir(dir)
	for _, dirEntry := range dirEntries {
		entryName := dirEntry.Name()
		entryPath := filepath.Join(dir, entryName)
		// 转换为绝对路径，确保排除目录判断准确
		absEntryPath, err := filepath.Abs(entryPath)
		if err != nil {
			return fmt.Errorf("获取绝对路径失败 %s：%w", entryPath, err)
		}
		if dirEntry.IsDir() {
			if excludedDirs[absEntryPath] {
				continue
			}
			if err := parseDirsVidInfo(entryPath); err != nil {
				return errf("处理子目录 %s 失败: %v", entryPath, err)
			}
			continue
		} else if dirEntry.Type().IsRegular() && (util.VideoExtRegex.MatchString(strings.ToLower(filepath.Ext(dirEntry.Name()))) || strings.ToLower(filepath.Ext(dirEntry.Name())) == ".mp4") {
			inputFile := filepath.Join(dir, dirEntry.Name())
			if _, err := os.Stat(inputFile); err == nil {
				//fmt.Println(fileInfo.ModTime().Format(time.DateTime), "+", targetTime.Format(time.DateTime))
				//fmt.Println(fileInfo.ModTime().Before(targetTime))
				//if fileInfo.ModTime().Before(targetTime) {
				fmt.Println(inputFile)
				num++
				fmt.Println(num)
				tempOutput := filepath.Join(dir, "."+filepath.Base(inputFile)+".tmp.mp4")
				if err := util.ConvertMKVToMP4(inputFile, tempOutput, "", false); err != nil {
					return err
				} else {
					if filepath.Ext(inputFile) != ".mp4" {
						baseName := strings.TrimSuffix(inputFile, filepath.Ext(inputFile))
						tmpFile := filepath.Join(baseName + ".mp4")
						if err := os.Rename(tempOutput, tmpFile); err != nil {
							// 替换失败，保留临时文件供排查
							return errf("替换原文件失败，临时文件保留在 %s: %w", tempOutput, err)
						}
						os.Remove(inputFile)
					} else {
						if err := os.Rename(tempOutput, inputFile); err != nil {
							// 替换失败，保留临时文件供排查
							return errf("替换原文件失败，临时文件保留在 %s: %w", tempOutput, err)
						}
					}
				}
				//}
			}
		}
	}
	return nil
}

// 辅助函数：格式化错误信息
func errf(format string, v ...interface{}) error {
	return fmt.Errorf(format, v...)
}

// 辅助函数：格式化 panic 信息
func panicf(format string, v ...interface{}) {
	panic(fmt.Sprintf(format, v...))
}
