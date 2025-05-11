package main

import (
	"fmt"
	"godemo/util"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	videoExtRegex = regexp.MustCompile(`(?i)\.(|mkv|avi|mov|wmv|flv|webm|mpeg|mpg|3gp|m4v)$`)
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
		processDirectory(s)
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
		if entry.IsDir() {
			processFile(entryPath)
		} else {
			processFile(filepath.Dir(entryPath))
		}
	}
	return nil
}

// processFile 处理单个文件，判断是否为 MKV 文件并进行转换
func processFile(filePath string) {
	_, err := os.Stat(filepath.Join(filePath, "a.txt"))
	if !os.IsNotExist(err) {
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
		if filepath.Ext(dirEntry.Name()) == ".mp4" || filepath.Ext(dirEntry.Name()) == ".mkv" {
			videoPath = filepath.Join(filePath, dirEntry.Name())
		}
		if strings.HasSuffix(dirEntry.Name(), ".str") || strings.HasSuffix(dirEntry.Name(), ".ass") || strings.HasSuffix(dirEntry.Name(), ".srt") {
			subtitlePath = filepath.Join(filePath, dirEntry.Name())
		}
	}
	newVideoPath := util.ReplaceChar(filepath.Base(videoPath))
	if subtitlePath != "" && filepath.Ext(videoPath) == ".mp4" {
		os.Rename(videoPath, newVideoPath)
		newSubtitlePath := util.ReplaceChar(filepath.Base(subtitlePath))
		os.Rename(subtitlePath, newSubtitlePath)
		if err := util.ConvertMKVToMP4(newVideoPath, "out.mp4", newSubtitlePath, true); err != nil {
			log.Printf("合并视频【%s】字幕出错 %v", videoPath, err)
			os.Rename(newVideoPath, videoPath)
			os.Rename(newSubtitlePath, subtitlePath)
			return
		} else {
			os.Remove(newVideoPath)
			os.Remove(newSubtitlePath)
			os.Rename("out.mp4", videoPath)
			os.Create(filepath.Join(filePath, "a.txt"))
			return
		}
	} else if filepath.Ext(newVideoPath) != ".mp4" && videoExtRegex.MatchString(filepath.Ext(newVideoPath)) && filepath.Ext(newVideoPath) != "." {
		baseName := strings.TrimSuffix(newVideoPath, filepath.Ext(newVideoPath))
		outputFile := filepath.Join(newVideoPath, baseName+".mp4")
		err := util.ConvertMKVToMP4(newVideoPath, outputFile, "", false)
		if err != nil {
			log.Printf("【%s】转换 mp4 失败: %v\n", newVideoPath, err)
		} else {
			log.Printf("【%s】转换 mp4 成功\n", newVideoPath)
			os.Remove(newVideoPath)
			processFile(filePath)
		}
	}
}

//// processFile 处理单个文件，判断是否为 MKV 文件并进行转换
//func processFile(mkvPath, mp4Path, filePath string) {
//	_, err := os.Stat(filepath.Join(mkvPath, "a.txt"))
//	if !os.IsNotExist(err) {
//		return
//	}
//	inputFile := filepath.Join(mkvPath, filePath)
//	//if strings.Contains(inputFile, "[") || strings.Contains(inputFile, "]") {
//	//	inputFile = strings.ReplaceAll(strings.ReplaceAll(inputFile, "[", "."), "]", "")
//	//	os.Rename(filepath.Join(mkvPath, filePath), inputFile)
//	//}
//	if subtitles, _ := util.CheckVideoHasSubtitles(inputFile); subtitles {
//		subtitle := filepath.Join(mp4Path, strings.TrimSuffix(filepath.Base(filePath), ".mp4")+".srt")
//		//subtitle := strings.TrimSuffix(filepath.Base(filePath), ".mp4") + ".srt"
//		if err := util.ExtractSubtitles(inputFile, subtitle); err != nil {
//			log.Printf("提取【%s】字幕出错", inputFile)
//			return
//		}
//		remoVoidSub := filepath.Join(filepath.Dir(filepath.Dir(inputFile)), filePath)
//		//remoVoidSub := filePath
//		if err := util.RemoveSubtitles(inputFile, remoVoidSub); err != nil {
//			log.Printf("去除视频【%s】字幕出错", inputFile)
//			return
//		} else {
//			os.Remove(inputFile)
//			os.Rename(remoVoidSub, inputFile)
//		}
//		out := "out.mp4"
//		if err := util.ConvertMKVToMP4(inputFile, out, subtitle, true); err != nil {
//			log.Printf("合并视频【%s】字幕出错", inputFile)
//			//os.Rename(filepath.Base(inputFile), inputFile)
//			//os.Rename(filepath.Base(subtitle), subtitle)
//			os.Rename(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(filepath.Base(remoVoidSub), "[", "."), "]", ""), " ", ""), inputFile)
//			os.Rename(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(filepath.Base(subtitle), "[", "."), "]", ""), " ", ""), subtitle)
//			return
//		} else {
//			//os.Remove(filepath.Base(remoVoidSub))
//			//os.Remove(filepath.Base(subtitle))
//			if filepath.Ext(filepath.Base(remoVoidSub)) == ".mkv" {
//				remoVoidSub = filepath.Join(mkvPath, strings.TrimSuffix(filepath.Base(remoVoidSub), ".mkv")+".mp4")
//			}
//			os.Remove(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(filepath.Base(remoVoidSub), "[", "."), "]", ""), " ", ""))
//			os.Remove(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(filepath.Base(subtitle), "[", "."), "]", ""), " ", ""))
//			os.Rename(out, inputFile)
//			os.Create(filepath.Join(mkvPath, "a.txt"))
//		}
//	} else {
//		dir, _ := os.ReadDir(mkvPath)
//		var videoPath, subtitlePath string
//		var readFile int64
//		var isFirst = true
//		for _, dirEntry := range dir {
//			fileInfo, _ := dirEntry.Info()
//			if isFirst {
//				readFile = fileInfo.Size()
//				isFirst = false
//				videoPath = filepath.Join(mkvPath, dirEntry.Name())
//			}
//			if fileInfo.Size() > readFile {
//				readFile = fileInfo.Size()
//				videoPath = filepath.Join(mkvPath, dirEntry.Name())
//			}
//			if strings.HasSuffix(dirEntry.Name(), ".str") || strings.HasSuffix(dirEntry.Name(), ".ass") || strings.HasSuffix(dirEntry.Name(), ".srt") {
//				subtitlePath = filepath.Join(mkvPath, dirEntry.Name())
//			}
//		}
//		if subtitlePath != "" {
//			if filepath.Ext(filepath.Base(videoPath)) == ".mkv" {
//				baseName := strings.TrimSuffix(filepath.Base(videoPath), ".mkv")
//				outputFile := filepath.Join(mp4Path, baseName+".mp4")
//				err := util.ConvertMKVToMP4(videoPath, outputFile, "", false)
//				if err != nil {
//					log.Printf("【%s】转换 mp4 失败: %v\n", inputFile, err)
//				} else {
//					log.Printf("【%s】转换 mp4 成功\n", inputFile)
//					os.Remove(videoPath)
//					os.Create(filepath.Join(mkvPath, "a.txt"))
//					videoPath = outputFile
//				}
//			}
//			if err := util.ConvertMKVToMP4(videoPath, "out.mp4", subtitlePath, true); err != nil {
//				log.Printf("合并视频【%s】字幕出错 %v", videoPath, err)
//				os.Rename(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(filepath.Base(videoPath), "[", "."), "]", ""), " ", ""), videoPath)
//				os.Rename(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(filepath.Base(subtitlePath), "[", "."), "]", ""), " ", ""), subtitlePath)
//				return
//			} else {
//				os.Remove(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(filepath.Base(videoPath), "[", "."), "]", ""), " ", ""))
//				os.Remove(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(filepath.Base(subtitlePath), "[", "."), "]", ""), " ", ""))
//				os.Rename("out.mp4", videoPath)
//				os.Create(filepath.Join(mkvPath, "a.txt"))
//			}
//		} else if filepath.Ext(filePath) == ".mkv" {
//			baseName := strings.TrimSuffix(filepath.Base(filePath), ".mkv")
//			outputFile := filepath.Join(mp4Path, baseName+".mp4")
//			err := util.ConvertMKVToMP4(inputFile, outputFile, "", false)
//			if err != nil {
//				log.Printf("【%s】转换 mp4 失败: %v\n", inputFile, err)
//			} else {
//				log.Printf("【%s】转换 mp4 成功\n", inputFile)
//				os.Remove(inputFile)
//				os.Create(filepath.Join(mkvPath, "a.txt"))
//			}
//		}
//	}
//}
