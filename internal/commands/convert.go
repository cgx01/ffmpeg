package commands

import (
	"ffmpeg/util"
	"fmt"
	"github.com/spf13/pflag"
	"os"
	"path/filepath"
	"strings"
)

func RunConvert(args []string) {
	fs := pflag.NewFlagSet("convert", pflag.ExitOnError)
	dirs := fs.StringSliceP("dirs", "d", []string{}, "Directories or files to process (comma separated or multiple flags)")
	targetFormat := fs.String("to", "mp4", "Target format (e.g. mp4, mkv, avi)")
	
	fs.Parse(args)

	// 合并 flag 指定的目录和位置参数指定的目录
	targetPaths := *dirs
	targetPaths = append(targetPaths, fs.Args()...)

	if len(targetPaths) == 0 {
		fmt.Println("Error: No paths specified. Usage: convert [paths...] or convert -d [paths...]")
		fs.Usage()
		return
	}

	// 确保 targetFormat 不带点
	*targetFormat = strings.TrimPrefix(*targetFormat, ".")

	fmt.Printf("Processing paths: %v, Target Format: %s\n", targetPaths, *targetFormat)

	for _, path := range targetPaths {
		info, err := os.Stat(path)
		if err != nil {
			fmt.Printf("Error accessing %s: %v\n", path, err)
			continue
		}

		if info.IsDir() {
			if err := parseDirsVidInfo(path, *targetFormat); err != nil {
				fmt.Printf("Error processing directory %s: %v\n", path, err)
			}
		} else {
			// Process single file
			if isVideoFile(path) {
				if err := processVideoFile(path, *targetFormat); err != nil {
					fmt.Printf("Error converting file %s: %v\n", path, err)
				}
			} else {
				fmt.Printf("Skipping non-video file: %s\n", path)
			}
		}
	}
}

// Defined to avoid re-parsing flags inside recursive calls if we were passing args, 
// but here we just pass the path string.
var excludedDirs = map[string]bool{}
var num int

func parseDirsVidInfo(dir string, targetFormat string) error {
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read dir %s failed: %w", dir, err)
	}

	for _, dirEntry := range dirEntries {
		entryName := dirEntry.Name()
		entryPath := filepath.Join(dir, entryName)
		
		absEntryPath, err := filepath.Abs(entryPath)
		if err != nil {
			return fmt.Errorf("get abs path failed %s: %w", entryPath, err)
		}

		if dirEntry.IsDir() {
			if excludedDirs[absEntryPath] {
				continue
			}
			if err := parseDirsVidInfo(entryPath, targetFormat); err != nil {
				return fmt.Errorf("process subdir %s failed: %w", entryPath, err)
			}
			continue
		} else {
			if isVideoFile(entryPath) {
				if err := processVideoFile(entryPath, targetFormat); err != nil {
					// Don't stop the whole directory process for one failed file, but log it.
					// Or do we return error? Original logic returned error.
					// Let's log and continue? Or return error.
					// Original: return err
					return err
				}
			}
		}
	}
	return nil
}

func isVideoFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return util.VideoExtRegex.MatchString(ext) || ext == ".mp4"
}

func processVideoFile(inputFile, targetFormat string) error {
	ext := strings.ToLower(filepath.Ext(inputFile))
	targetExt := "." + strings.ToLower(targetFormat)

	// --- 自动查找字幕 ---
	var subtitlePath string
	baseName := strings.TrimSuffix(inputFile, filepath.Ext(inputFile))
	dir := filepath.Dir(inputFile)

	// 1. 优先尝试同名匹配 (Exact Match)
	for _, subExt := range []string{".srt", ".ass", ".str"} {
		potentialSubPath := filepath.Join(dir, filepath.Base(baseName)+subExt)
		if _, err := os.Stat(potentialSubPath); err == nil {
			subtitlePath = potentialSubPath
			fmt.Printf("找到同名字幕文件: %s\n", subtitlePath)
			break
		}
	}

	// 2. 如果没找到同名，扫描目录下其他字幕 (Loose Match)
	if subtitlePath == "" {
		entries, _ := os.ReadDir(dir)
		var subFiles []string
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			lowerName := strings.ToLower(e.Name())
			if strings.HasSuffix(lowerName, ".srt") || strings.HasSuffix(lowerName, ".ass") || strings.HasSuffix(lowerName, ".str") {
				subFiles = append(subFiles, filepath.Join(dir, e.Name()))
			}
		}

		if len(subFiles) == 1 {
			subtitlePath = subFiles[0]
			fmt.Printf("找到目录下唯一字幕文件: %s\n", subtitlePath)
		} else if len(subFiles) > 1 {
			vidNameLower := strings.ToLower(filepath.Base(baseName))
			for _, sub := range subFiles {
				subNameLower := strings.ToLower(filepath.Base(sub))
				if strings.Contains(subNameLower, vidNameLower) || strings.Contains(vidNameLower, strings.TrimSuffix(filepath.Base(subNameLower), filepath.Ext(subNameLower))) {
					subtitlePath = sub
					fmt.Printf("通过模糊匹配找到字幕文件: %s\n", subtitlePath)
					break
				}
			}
		}
	}
	// --- 字幕查找结束 ---

	hasSubtitle := subtitlePath != ""

	// 如果已经是目标格式，且没有发现字幕文件，则跳过
	if ext == targetExt && !hasSubtitle {
		return nil
	}

	fmt.Printf("准备处理视频: %s (目标格式: %s, 嵌入字幕: %v)\n", inputFile, targetFormat, hasSubtitle)
	num++
	
	// Temp output file
	tempOutput := filepath.Join(dir, "."+strings.TrimSuffix(filepath.Base(inputFile), filepath.Ext(inputFile))+".tmp"+targetExt)
	
	// Convert or Embed
	if err := util.ConvertVideo(inputFile, tempOutput, subtitlePath, hasSubtitle); err != nil {
		return fmt.Errorf("处理失败: %w", err)
	}
	
	// Success
	// 如果原文件和目标文件后缀相同（如都是 .mp4），重命名时会覆盖原文件。
	// 但我们之前已经用 .tmp 作为中间名，所以安全。
	finalTargetFile := filepath.Join(dir, filepath.Base(baseName)+targetExt)
	
	// 如果目标路径和输入路径一致（例如都是 .mp4），先删除原文件或者直接重命名覆盖
	// 在 Windows 上，Rename 无法直接覆盖已存在的文件，所以我们需要先处理。
	if strings.ToLower(finalTargetFile) == strings.ToLower(inputFile) {
		fmt.Printf("覆盖原文件: %s\n", inputFile)
		os.Remove(inputFile)
	} else {
		fmt.Printf("删除原始文件: %s\n", inputFile)
		os.Remove(inputFile)
	}
	
	if err := os.Rename(tempOutput, finalTargetFile); err != nil {
		return fmt.Errorf("重命名临时文件 %s 失败: %w", tempOutput, err)
	}
	
	// Remove subtitle file as well if it was embedded
	if hasSubtitle {
		fmt.Printf("删除已嵌入的字幕文件: %s\n", subtitlePath)
		os.Remove(subtitlePath)
	}
	
	fmt.Printf("处理成功: %s\n", finalTargetFile)
	return nil
}
