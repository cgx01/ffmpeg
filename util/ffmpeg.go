package util

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"image/gif"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

const (
	//ffmpegBin = "D:\\software\\ffmpeg-7.0.2-full_build-shared\\bin\\ffmpeg.exe"
	ffmpegBin = "ffmpeg"
	//GIFPARAM  = "fps=10,scale=1280:720:flags=lanczos,split[s0][s1];[s0]palettegen=stats_mode=single[p];[s1][p]paletteuse=dither=bayer:bayer_scale=3"
	//MP4PARAM  = "fps=10,scale=1280:720:flags=lanczos,split[s0][s1];[s0]palettegen[p];[s1][p]paletteuse"
)

var (
	ffmpegSpecialChars = regexp.MustCompile(`[][(){}?*%#&'"\t]`)
	sing               = make(chan struct{}, 1)
)

// mp4转为gif、压缩gif
func CompressGif(inputFile, outputFile, filesize string, isMP4 bool) error {
	var cmd *exec.Cmd
	if isMP4 {
		width, height := getMP4Stream(inputFile)
		MP4PARAM := fmt.Sprintf("fps=10,scale=%d:%d:flags=lanczos,split[s0][s1];[s0]palettegen[p];[s1][p]paletteuse", width, height)
		//ffmpeg -i Join_file_082107340.mp4 -vf "fps=10,scale=1280:720:flags=lanczos,split[s0][s1];[s0]palettegen[p];[s1][p]paletteuse" -fs 9M Join_file_082107340.gif
		cmd = exec.Command(ffmpegBin, "-i", inputFile, "-vf", fmt.Sprintf("%s", MP4PARAM), "-fs", filesize, outputFile)
	} else {
		open, err := os.Open(inputFile)
		if err != nil {
			return fmt.Errorf("无法打开文件: %v\n", err)
		}
		defer open.Close()
		img, err := gif.DecodeConfig(open)
		if err != nil {
			return fmt.Errorf("无法解码图像: %v filename is %s\n", err, inputFile)
		}
		GIFPARAM := fmt.Sprintf("fps=10,scale=%d:%d:flags=lanczos,split[s0][s1];[s0]palettegen=stats_mode=single[p];[s1][p]paletteuse=dither=bayer:bayer_scale=3", img.Width, img.Height)
		//ffmpeg -i 51967511.gif -vf  -fs 9M -max_muxing_queue_size 9999 o.gif
		cmd = exec.Command(ffmpegBin, "-i", inputFile, "-vf", fmt.Sprintf("%s", GIFPARAM), "-fs", filesize, "-max_muxing_queue_size", "9999", outputFile)
	}
	fmt.Printf("执行命令: %v\n", cmd.Args)

	// 创建字节缓冲区来捕获命令的标准输出和标准错误
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	defer func() {
		sing <- struct{}{}
	}()
	go printProgress(inputFile, outputFile, sing)
	// 执行命令
	err := cmd.Run()
	if err != nil {
		// 打印标准错误输出
		fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
		return err
	}
	return nil
}

type probeResult struct {
	Streams []struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"streams"`
}

func getMP4Stream(inputFile string) (width, height int) {
	// 执行ffprobe命令
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height",
		"-of", "json",
		inputFile)

	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("执行ffprobe失败: %v\n", err)
		fmt.Println("请确保已安装FFmpeg并将其添加到系统PATH中")
		os.Exit(1)
	}

	// 解析JSON输出
	var result probeResult
	if err := json.Unmarshal(output, &result); err != nil {
		fmt.Printf("解析输出失败: %v\n", err)
		os.Exit(1)
	}

	if len(result.Streams) == 0 {
		fmt.Println("未找到视频流")
		os.Exit(1)
	}

	return result.Streams[0].Width, result.Streams[0].Height
}

// ConvertMKVToMP4 函数用于将 MKV 文件转换为 MP4 文件
func ConvertMKVToMP4(inputFile, outputFile, subtitle string, isSub bool) error {
	// 构建 ffmpeg 命令
	var cmd *exec.Cmd
	if isSub && subtitle != "" {
		cmd = exec.Command(ffmpegBin, "-i", inputFile, "-vf", fmt.Sprintf("subtitles=%s", subtitle), outputFile)
	} else {
		cmd = exec.Command(ffmpegBin, "-i", inputFile, "-c:v", "libx264", "-c:a", "aac", outputFile)
	}

	// 创建字节缓冲区来捕获命令的标准输出和标准错误
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	go printProgress(inputFile, outputFile, sing)
	defer func() {
		sing <- struct{}{}
	}()
	// 执行命令
	err := cmd.Run()

	if err != nil {
		// 打印标准错误输出
		fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
		return err
	}
	return nil
}

func printProgress(inp, out string, sing <-chan struct{}) {
	var lastSize int64
	var lastPrintTime time.Time
	fileInfo, err := os.Stat(inp)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("文件不存在")
		} else {
			fmt.Printf("获取文件信息失败: %v\n", err)
		}
		return
	}

	// 获取文件大小（单位：字节）
	fileSize := fileInfo.Size()
	// 创建带颜色的进度条输出
	progressColor := color.New(color.FgGreen).SprintFunc()
	percentageColor := color.New(color.FgCyan, color.Bold).SprintfFunc()

	// 使用 ticker 替代 timer，更适合周期性任务
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	// 初始输出
	fmt.Printf("开始处理文件[%s]...\n", inp)
	var barWidth, barFilled, barEmpty int
	var percent float64
	for {
		select {
		case <-sing:
			// 生成满进度条（100%）
			percent = 100.0
			barWidth = 50
			barFilled = int(percent / 100 * float64(barWidth))
			barEmpty = barWidth - barFilled

			// 设置颜色函数
			progressColor := color.New(color.FgGreen).SprintFunc()
			percentageColor := color.New(color.FgCyan, color.Bold).SprintfFunc()

			// 生成完整进度条
			progressBar := fmt.Sprintf(
				"\r%s [%s%s] %s",
				progressColor("处理文件进度:"),
				strings.Repeat("█", barFilled),
				strings.Repeat(" ", barEmpty),
				percentageColor("%.2f%%", percent),
			)
			fmt.Println(progressBar)
			return
		case <-ticker.C:
			// 读取输出文件大小
			info, err := os.Stat(out)
			if err != nil {
				// 文件可能尚未创建，或者发生其他错误
				continue
			}

			currentSize := info.Size()

			// 只有当大小变化或超过1秒未更新时才打印
			if currentSize != lastSize || time.Since(lastPrintTime) > time.Second {
				lastSize = currentSize
				lastPrintTime = time.Now()

				// 计算百分比
				percent = float64(currentSize) / float64(fileSize) * 100

				// 生成进度条
				barWidth = 50
				barFilled = int(percent / 100 * float64(barWidth))
				barEmpty = barWidth - barFilled

				progressBar := fmt.Sprintf(
					"\r%s [%s%s] %s",
					progressColor("处理文件进度:"),
					strings.Repeat("█", barFilled),
					strings.Repeat(" ", barEmpty),
					percentageColor("%.2f%%", percent),
				)

				fmt.Print(progressBar)
			}
		}
	}
}

// ExtractSubtitles 从视频中提取字幕
func ExtractSubtitles(inputVideoPath, outputSubtitlePath string) error {
	// 构建 FFmpeg 命令
	cmd := exec.Command(ffmpegBin, "-i", inputVideoPath, "-map", "0:s:0", "-c:s", "srt", outputSubtitlePath)
	fmt.Printf("执行命令: %v\n", cmd.Args)

	// 创建缓冲区用于存储命令的标准输出和标准错误输出
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	// 执行命令
	err := cmd.Run()
	if err != nil {
		// 若命令执行出错，打印错误信息和标准错误输出内容
		log.Printf("执行 FFmpeg 命令时出错: %v\n", err)
		log.Printf("标准错误输出: %s\n", stderr.String())
		return err
	}

	// 打印标准输出内容
	fmt.Printf("命令标准输出: %s\n", out.String())
	return nil
}

// RemoveSubtitles 去掉视频中的字幕
func RemoveSubtitles(inputVideoPath, outputVideoPath string) error {
	// 构建 FFmpeg 命令
	cmd := exec.Command(ffmpegBin, "-i", inputVideoPath, "-map", "0:v", "-map", "0:a", "-c", "copy", outputVideoPath)
	//ffmpeg -i brazzersexxtra.24.12.13.angela.white.this.flight.attendant.fucks.part.1.mp4 -map 0:v -map 0:a -c copy output.mp4
	fmt.Printf("执行命令: %v\n", cmd.Args)

	// 创建缓冲区用于存储命令的标准输出和标准错误输出
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	// 执行命令
	err := cmd.Run()
	if err != nil {
		// 若命令执行出错，打印错误信息和标准错误输出内容
		log.Printf("执行 FFmpeg 命令时出错: %v\n", err)
		log.Printf("标准错误输出: %s\n", stderr.String())
		return err
	}

	// 打印标准输出内容
	fmt.Printf("命令标准输出: %s\n", out.String())
	return nil
}

// CheckVideoHasSubtitles 检查视频文件是否包含字幕
func CheckVideoHasSubtitles(videoPath string) (bool, error) {
	// 构建 FFmpeg 命令
	cmd := exec.Command("ffmpeg", "-i", videoPath)

	// 获取命令的标准错误输出管道
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return false, err
	}

	// 启动命令
	if err := cmd.Start(); err != nil {
		return false, err
	}

	// 读取标准错误输出
	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		line := scanner.Text()
		// 检查输出行中是否包含 "Subtitle" 关键字
		if strings.Contains(line, "Subtitle") {
			return true, nil
		}
	}

	// 等待命令执行完成
	if err := cmd.Wait(); err != nil {
		return false, err
	}

	// 如果没有找到 "Subtitle" 关键字，则认为视频不包含字幕
	return false, nil
}

//func moveFile(src, dst string) error {
//	// 复制文件
//	in, err := os.Open(src)
//	if err != nil {
//		return err
//	}
//	defer in.Close()
//
//	out, err := os.Create(dst)
//	if err != nil {
//		return err
//	}
//	defer func() {
//		out.Close()
//		os.Remove(dst) // 复制失败时清理
//	}()
//
//	_, err = io.Copy(out, in)
//	if err != nil {
//		return err
//	}
//
//	// 关闭文件并删除原文件
//	if err = out.Close(); err != nil {
//		return err
//	}
//	return os.Remove(src)
//}

func ReplaceChar(name string) string {
	if ffmpegSpecialChars.MatchString(name) {
		return ffmpegSpecialChars.ReplaceAllString(name, "")
	}
	return name
}
