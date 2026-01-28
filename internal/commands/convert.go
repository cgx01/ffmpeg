package commands

import (
	"context"
	"ffmpeg/util"
	"fmt"
	"github.com/spf13/pflag"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
)

type ConvertJob struct {
	Path         string
	TargetFormat string
	DryRun       bool
	Accel        string
	Force        bool
	VCodec       string
}

func RunConvert(args []string) {
	fs := pflag.NewFlagSet("convert", pflag.ExitOnError)
	dirs := fs.StringSliceP("dirs", "d", []string{}, "Directories or files to process")
	targetFormat := fs.String("to", "mp4", "Target format")
	dryRun := fs.Bool("dry-run", false, "Simulate operation")
	workers := fs.Int("workers", 1, "Number of concurrent workers")
	accel := fs.String("accel", "", "Hardware acceleration (cuda, qsv, amf)")
	force := fs.Bool("force", false, "Force conversion even if format matches")
	vcodec := fs.String("vcodec", "h264", "Video codec (h264, hevc)")
	
	fs.Parse(args)

	targetPaths := *dirs
	targetPaths = append(targetPaths, fs.Args()...)

	if len(targetPaths) == 0 {
		fmt.Println("Error: No paths specified.")
		fs.Usage()
		return
	}

	*targetFormat = strings.TrimPrefix(*targetFormat, ".")

	// 1. 设置信号监听 (优雅退出)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n🛑 接到停止信号，正在取消所有任务并清理...")
		cancel()
	}()

	// 2. 初始化 Worker Pool
	jobChan := make(chan ConvertJob, len(targetPaths)*100) 
	var wg sync.WaitGroup

	numWorkers := *workers
	if numWorkers < 1 {
		numWorkers = 1
	}
	
	fmt.Printf("🚀 启动 %d 个并发 Worker (加速: %s, 目标编码: %s)\n", numWorkers, *accel, *vcodec)

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for job := range jobChan {
				select {
				case <-ctx.Done():
					return
				default:
				}
				if err := processVideoJob(ctx, job); err != nil {
					if ctx.Err() == nil {
						fmt.Printf("[Worker %d] ❌ 处理失败 %s: %v\n", workerID, job.Path, err)
					}
				}
			}
		}(i)
	}

	// 3. 收集任务
	go func() {
		for _, path := range targetPaths {
			if ctx.Err() != nil {
				break
			}
			info, err := os.Stat(path)
			if err != nil {
				continue
			}
			if info.IsDir() {
				walkDir(ctx, path, *targetFormat, *dryRun, *accel, *force, *vcodec, jobChan)
			} else if isVideoFile(path) {
				jobChan <- ConvertJob{Path: path, TargetFormat: *targetFormat, DryRun: *dryRun, Accel: *accel, Force: *force, VCodec: *vcodec}
			}
		}
		close(jobChan)
	}()

	wg.Wait()
	fmt.Println("✅ 所有任务已完成")
}

func walkDir(ctx context.Context, dir, targetFormat string, dryRun bool, accel string, force bool, vcodec string, jobChan chan<- ConvertJob) {
	entries, _ := os.ReadDir(dir)
	for _, entry := range entries {
		if ctx.Err() != nil {
			return
		}
		fullPath := filepath.Join(dir, entry.Name())
		if entry.IsDir() {
			walkDir(ctx, fullPath, targetFormat, dryRun, accel, force, vcodec, jobChan)
		} else if isVideoFile(fullPath) {
			jobChan <- ConvertJob{Path: fullPath, TargetFormat: targetFormat, DryRun: dryRun, Accel: accel, Force: force, VCodec: vcodec}
		}
	}
}

func isVideoFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return util.VideoExtRegex.MatchString(ext) || ext == ".mp4"
}

func processVideoJob(ctx context.Context, job ConvertJob) error {
	inputFile := job.Path
	targetFormat := job.TargetFormat
	dryRun := job.DryRun
	accel := job.Accel
	force := job.Force
	vcodec := job.VCodec

	ext := strings.ToLower(filepath.Ext(inputFile))
	targetExt := "." + strings.ToLower(targetFormat)

	// 字幕查找
	var subtitlePath string
	baseName := strings.TrimSuffix(inputFile, filepath.Ext(inputFile))
	dir := filepath.Dir(inputFile)
	for _, subExt := range []string{".srt", ".ass"} {
		potential := filepath.Join(dir, filepath.Base(baseName)+subExt)
		if _, err := os.Stat(potential); err == nil {
			subtitlePath = potential
			break
		}
	}
	if subtitlePath == "" {
		if entries, err := os.ReadDir(dir); err == nil {
			var subFiles []string
			for _, e := range entries {
				if !e.IsDir() {
					name := strings.ToLower(e.Name())
					if strings.HasSuffix(name, ".srt") || strings.HasSuffix(name, ".ass") {
						subFiles = append(subFiles, filepath.Join(dir, e.Name()))
					}
				}
			}
			if len(subFiles) == 1 {
				subtitlePath = subFiles[0]
			}
		}
	}

	hasSubtitle := subtitlePath != ""
	
	// 如果不强制，且已经是目标格式，且没有发现字幕文件，则跳过
	if !force && ext == targetExt && !hasSubtitle {
		fmt.Printf("⏭️  跳过: %s (已是目标格式且无字幕，使用 --force 强制)\n", filepath.Base(inputFile))
		return nil 
	}

	tempOutput := filepath.Join(dir, "."+filepath.Base(baseName)+".tmp"+targetExt)
	defer func() {
		if ctx.Err() != nil {
			os.Remove(tempOutput)
		}
	}()

	if err := util.ConvertVideo(ctx, inputFile, tempOutput, subtitlePath, hasSubtitle, dryRun, accel, vcodec); err != nil {
		return err
	}

	if dryRun {
		return nil
	}

	finalTargetFile := filepath.Join(dir, filepath.Base(baseName)+targetExt)
	if strings.ToLower(finalTargetFile) == strings.ToLower(inputFile) {
		os.Remove(inputFile)
	} else {
		os.Remove(inputFile)
	}
	if err := util.SafeRename(tempOutput, finalTargetFile); err != nil {
		return err
	}
	if hasSubtitle {
		os.Remove(subtitlePath)
	}
	fmt.Printf("✅ 完成: %s\n", filepath.Base(finalTargetFile))
	return nil
}
