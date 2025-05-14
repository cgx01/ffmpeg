package main

import (
	"ffmpeg/util"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	videoExtRegex = regexp.MustCompile(`(?i)\.(mkv|avi|mov|mpeg|mpg|3gp|asf|divx|xvid|m2ts|ts|f4v|swf|mxf|prores|vfw|nut|ivf|m1v|m2v|mj2|mjp2|mpv2|qt|yuv|amv|drc|fli|flv|gvi|gxf|m2t|m4v|mjp|mk3d|mks|mpv|mpeg1|mpeg2|mpeg4|mts|nsv|nuv|ogm|ogv|ogx|ps|rec|rm|rmvb|roq|svi|vob|webm|wm|wmv|wtv|xesc)$`)
)

func main() {
	if len(os.Args) < 3 {
		fmt.Printf("请输入ffmpeg的bin目录和目录路径 多个目录用空格隔开\n")
		fmt.Printf("命令格式为: exe C:\\ffmpeg\\bin 需要修改的目录（多个目录用空格隔开）")
		return
	}
	ffmpeg := os.Args[1]
	if ffmpeg != "" {
		os.Setenv("PATH", os.Getenv("PATH")+";"+ffmpeg)
		// 执行 ffmpeg 命令
		cmd := exec.Command("ffmpeg", "-version")
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Fatalf("执行 ffmpeg 失败: %v\n输出: %s", err, string(output))
		}
	}
	dir := os.Args[2:]
	for _, s := range dir {
		if err := processDirectory(s); err != nil {
			return
		}
	}
}

// processDirectory 递归处理目录及其子目录中的文件
func processDirectory(mkvPath string) error {
	dirEntries, err := os.ReadDir(mkvPath)
	if err != nil {
		return fmt.Errorf("读取目录 %s 失败: %w", mkvPath, err)
	}
	for _, entry := range dirEntries {
		entryPath := filepath.Join(mkvPath, entry.Name())
		fmt.Printf("entryPath:%s, dirEntries:%d\n", entryPath, len(dirEntries))
		if entry.IsDir() {
			processFile(entryPath)
		} else if videoExtRegex.MatchString(entryPath) || filepath.Ext(entryPath) == ".mp4" || filepath.Ext(entryPath) == ".ass" || filepath.Ext(entryPath) == ".str" {
			processFile(filepath.Dir(entryPath))
		}
	}
	return nil
}

// processFile 处理单个文件，判断是否为 MKV 文件并进行转换
func processFile(filePath string) {
	_, err := os.Stat(filepath.Join(filePath, "a.txt"))
	if !os.IsNotExist(err) || strings.Contains(filePath, "VR") {
		fmt.Printf("return : %s\n", filePath)
		return
	}
	if strings.Contains(filePath, " ") {
		os.Rename(filepath.Join(filePath), strings.ReplaceAll(filePath, " ", ""))
		filePath = strings.ReplaceAll(filePath, " ", "")
	}
	dir, _ := os.ReadDir(filePath)
	var videoPath, subtitlePath string
	for _, dirEntry := range dir {
		if dirEntry.IsDir() {
			processDirectory(filepath.Join(filePath, dirEntry.Name()))
			continue
		}
		if filepath.Ext(dirEntry.Name()) == ".mp4" || videoExtRegex.MatchString(filepath.Ext(dirEntry.Name())) {
			if videoPath != "" {
				newVideo := filepath.Join(filePath, dirEntry.Name())
				readFile, _ := os.ReadFile(videoPath)
				file, _ := os.ReadFile(newVideo)
				if len(file) > len(readFile) {
					videoPath = newVideo
				}
			} else {
				videoPath = filepath.Join(filePath, dirEntry.Name())
			}
		}
		if strings.HasSuffix(dirEntry.Name(), ".str") || strings.HasSuffix(dirEntry.Name(), ".ass") || strings.HasSuffix(dirEntry.Name(), ".srt") {
			subtitlePath = filepath.Join(filePath, dirEntry.Name())
		}
	}
	newVideoPath := util.ReplaceChar(filepath.Base(videoPath))
	defer func() {
		if err != nil {
			// 以追加模式打开文件（若文件不存在则创建，允许写入）
			file, err := os.OpenFile("error.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				fmt.Printf("无法打开文件: %v", err)
			}
			defer file.Close()
			fmt.Printf("%s\n", filePath)
			if _, err := file.WriteString(videoPath + "\n"); err != nil {
				fmt.Printf("写入文件失败: %v", err)
			}

		}
	}()
	if subtitlePath != "" && videoPath != "" {
		os.Rename(videoPath, newVideoPath)
		newSubtitlePath := util.ReplaceChar(filepath.Base(subtitlePath))
		os.Rename(subtitlePath, newSubtitlePath)
		if err = util.ConvertMKVToMP4(newVideoPath, "out.mp4", newSubtitlePath, true); err != nil {
			log.Printf("合并视频【%s】字幕出错 %v", videoPath, err)
			os.Rename(newVideoPath, videoPath)
			os.Rename(newSubtitlePath, subtitlePath)
			return
		} else {
			os.Remove(newVideoPath)
			os.Remove(newSubtitlePath)
			if filepath.Ext(videoPath) != ".mp4" {
				baseName := strings.TrimSuffix(videoPath, filepath.Ext(videoPath))
				outputFile := filepath.Join(baseName + ".mp4")
				os.Rename("out.mp4", outputFile)
			} else {
				os.Rename("out.mp4", videoPath)
			}
			os.Create(filepath.Join(filePath, "a.txt"))
			return
		}
	} else if filepath.Ext(newVideoPath) != ".mp4" && videoExtRegex.MatchString(filepath.Ext(videoPath)) {
		os.Rename(videoPath, newVideoPath)
		baseName := strings.TrimSuffix(videoPath, filepath.Ext(videoPath))
		outputFile := filepath.Join(baseName + ".mp4")
		err = util.ConvertMKVToMP4(newVideoPath, outputFile, "", false)
		if err != nil {
			log.Printf("【%s】转换 mp4 失败 文件路径为: %v\n", filePath, err)
		} else {
			log.Printf("【%s】转换 mp4 成功\n", videoPath)
			os.Rename(newVideoPath, outputFile)
		}
	}
}
