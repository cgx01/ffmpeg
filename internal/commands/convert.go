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
}

func RunConvert(args []string) {
	fs := pflag.NewFlagSet("convert", pflag.ExitOnError)
	dirs := fs.StringSliceP("dirs", "d", []string{}, "Directories or files to process")
	targetFormat := fs.String("to", "mp4", "Target format")
	dryRun := fs.Bool("dry-run", false, "Simulate operation")
	workers := fs.Int("workers", 1, "Number of concurrent workers")
	accel := fs.String("accel", "", "Hardware acceleration (cuda, qsv, amf)")
	force := fs.Bool("force", false, "Force conversion even if format matches")
	
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
	jobChan := make(chan ConvertJob, len(targetPaths)*100) // 缓冲稍大一点
	var wg sync.WaitGroup

	// 限制 worker 数量，至少为 1
	numWorkers := *workers
	if numWorkers < 1 {
		numWorkers = 1
	}
	
	fmt.Printf("🚀 启动 %d 个并发 Worker (加速模式: %s)\n", numWorkers, *accel)
	if *dryRun {
		fmt.Println("🚧 DRY-RUN 模式: 不会修改任何文件")
	}

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for job := range jobChan {
				// 检查是否已取消
				select {
				case <-ctx.Done():
					return
				default:
				}

				// 执行任务
				if err := processVideoJob(ctx, job); err != nil {
					// 如果是取消导致的错误，不打印常规错误日志
					if ctx.Err() == nil {
						fmt.Printf("[Worker %d] ❌ 处理失败 %s: %v\n", workerID, job.Path, err)
					}
				}
			}
		}(i)
	}

	// 3. 收集并发送任务
	go func() {
		for _, path := range targetPaths {
			// 检查 ctx
			if ctx.Err() != nil {
				break
			}
			
			info, err := os.Stat(path)
			if err != nil {
				fmt.Printf("Error accessing %s: %v\n", path, err)
				continue
			}

			if info.IsDir() {
				walkDir(ctx, path, *targetFormat, *dryRun, *accel, *force, jobChan)
			} else {
				if isVideoFile(path) {
					jobChan <- ConvertJob{Path: path, TargetFormat: *targetFormat, DryRun: *dryRun, Accel: *accel, Force: *force}
				}
			}
		}
		close(jobChan)
	}()

	// 4. 等待所有任务完成
	wg.Wait()
	fmt.Println("✅ 所有任务已完成 (或已停止)")
}

func walkDir(ctx context.Context, dir, targetFormat string, dryRun bool, accel string, force bool, jobChan chan<- ConvertJob) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Printf("Read dir failed %s: %v\n", dir, err)
		return
	}

	for _, entry := range entries {
		if ctx.Err() != nil {
			return
		}
		
		fullPath := filepath.Join(dir, entry.Name())
		
		if entry.IsDir() {
			walkDir(ctx, fullPath, targetFormat, dryRun, accel, force, jobChan)
		} else {
			if isVideoFile(fullPath) {
				jobChan <- ConvertJob{Path: fullPath, TargetFormat: targetFormat, DryRun: dryRun, Accel: accel, Force: force}
			}
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

	ext := strings.ToLower(filepath.Ext(inputFile))
	targetExt := "." + strings.ToLower(targetFormat)

	// 字幕查找逻辑
	var subtitlePath string
	baseName := strings.TrimSuffix(inputFile, filepath.Ext(inputFile))
	dir := filepath.Dir(inputFile)

	// 1. 同名
	for _, subExt := range []string{".srt", ".ass", ".str"} {
		potential := filepath.Join(dir, filepath.Base(baseName)+subExt)
		if _, err := os.Stat(potential); err == nil {
			subtitlePath = potential
			break
		}
	}
	// 2. 目录下唯一
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
		return nil // 跳过
	}

	// Temp file
	tempOutput := filepath.Join(dir, "."+strings.TrimSuffix(filepath.Base(inputFile), filepath.Ext(inputFile))+".tmp"+targetExt)
	
	// 确保 temp 被清理
	defer func() {
		if ctx.Err() != nil {
			os.Remove(tempOutput)
		}
	}()

	if err := util.ConvertVideo(ctx, inputFile, tempOutput, subtitlePath, hasSubtitle, dryRun, accel); err != nil {
		return err
	}

	if dryRun {
		return nil
	}

	// Rename
	finalTargetFile := filepath.Join(dir, filepath.Base(baseName)+targetExt)
	
	if strings.ToLower(finalTargetFile) == strings.ToLower(inputFile) {
		os.Remove(inputFile)
	} else {
		os.Remove(inputFile)
	}
	
	if err := util.SafeRename(tempOutput, finalTargetFile); err != nil {
		return fmt.Errorf("rename failed: %w", err)
	}

	if hasSubtitle {
		os.Remove(subtitlePath)
	}
	
	fmt.Printf("✅ 完成: %s\n", filepath.Base(finalTargetFile))
	return nil
}
