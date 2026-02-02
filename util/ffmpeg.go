package util

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"image/gif"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	ffmpegBin  = "ffmpeg"
	ffprobEBin = "ffprobe"
)

var (
	ffmpegSpecialChars = regexp.MustCompile(`[][(){}?*%#&'"\t, ]`)
	VideoExtRegex      = regexp.MustCompile(`(?i)\.(mkv|avi|mov|mpeg|mpg|3gp|asf|divx|xvid|m2ts|ts|f4v|swf|mxf|prores|vfw|nut|ivf|m1v|m2v|mj2|mjp2|mpv2|qt|yuv|amv|drc|fli|flv|gvi|gxf|m2t|m4v|mjp|mk3d|mks|mpv|mpeg1|mpeg2|mpeg4|mts|nsv|nuv|ogm|ogv|ogx|ps|rec|rm|rmvb|roq|svi|vob|webm|wm|wmv|wtv|xesc)$`)
)

// mp4转为gif、压缩gif
func CompressGif(inputFile, outputFile, filesize string, isMP4 bool) error {
	var cmd *exec.Cmd
	if isMP4 {
		width, height := getMP4Stream(inputFile)
		MP4PARAM := fmt.Sprintf("fps=10,scale=%d:%d:flags=lanczos,split[s0][s1];[s0]palettegen[p];[s1][p]paletteuse", width, height)
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
		cmd = exec.Command(ffmpegBin, "-i", inputFile, "-vf", fmt.Sprintf("%s", GIFPARAM), "-fs", filesize, "-max_muxing_queue_size", "9999", outputFile)
	}
	fmt.Printf("执行命令: %v\n", cmd.Args)

	// 创建字节缓冲区来捕获命令的标准输出和标准错误
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
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

// GetVideoCodec 获取视频流的编码格式 (如 h264, hevc)
func GetVideoCodec(inputFile string) (string, error) {
	cmd := exec.Command(ffprobEBin,
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=codec_name",
		"-of", "json",
		inputFile)

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("ffprobe error: %v", err)
	}

	type probeCodec struct {
		Streams []struct {
			CodecName string `json:"codec_name"`
		} `json:"streams"`
	}

	var result probeCodec
	if err := json.Unmarshal(output, &result); err != nil {
		return "", err
	}

	if len(result.Streams) == 0 {
		return "", fmt.Errorf("no video stream found")
	}

	return strings.ToLower(result.Streams[0].CodecName), nil
}

// ConvertVideo 函数用于通用的视频格式转换
func ConvertVideo(ctx context.Context, inputFile, outputFile, subtitle string, isSub, dryRun bool, accel string, vcodec string) error {
	var args []string
	args = append(args, "-i", inputFile)

	// 获取源视频编码
	srcCodec, err := GetVideoCodec(inputFile)
	if err != nil {
		fmt.Printf("警告: 无法探测编码 (%v)，将进行重编码\n", err)
		srcCodec = ""
	}

	// 1. 字幕烧录逻辑 (必须重编码)
	if isSub && subtitle != "" {
		// 创建一个临时且文件名简单的字幕副本，规避特殊字符路径问题
		tempSubPath, err := createSafeTempSubtitle(subtitle)
		if err != nil {
			return fmt.Errorf("创建临时字幕文件失败: %w", err)
		}
		defer os.Remove(tempSubPath)

		// 由于是临时文件，路径必定规范，只需要处理 Windows 路径分隔符
		// FFmpeg filter 里的路径需要使用 /，且冒号需要转义（如 C\:）
		sanitizedSub := filepath.ToSlash(tempSubPath)
		sanitizedSub = strings.ReplaceAll(sanitizedSub, ":", "\\:")

		args = append(args, "-vf", fmt.Sprintf("subtitles='%s'", sanitizedSub))

		// 使用指定的编码器
		args = append(args, buildCodecArgs(vcodec, accel)...)
	} else {
		// 2. 智能判断
		targetExt := strings.ToLower(strings.TrimPrefix(filepath.Ext(outputFile), "."))

		// 只有目标编码与源编码一致，且容器兼容时，才进行流复制
		// 比如：用户要求 h264，源也是 h264，则 copy
		// 比如：用户要求 h264，源是 hevc，则必须转码
		shouldCopyVideo := (targetExt == "mp4" || targetExt == "mkv") && (srcCodec == strings.ToLower(vcodec))

		if shouldCopyVideo {
			fmt.Printf("⚡ 触发智能流复制 (源编码 %s 符合目标要求) -> -c:v copy\n", srcCodec)
			args = append(args, "-c:v", "copy", "-c:a", "aac")
		} else {
			// 重编码
			args = append(args, buildCodecArgs(vcodec, accel)...)
		}
	}

	args = append(args, outputFile)

	if dryRun {
		fmt.Printf("[Dry-Run] Would execute: %s %s\n", ffmpegBin, strings.Join(args, " "))
		return nil
	}

	cmd := exec.CommandContext(ctx, ffmpegBin, args...)

	// 同时捕获 stderr 用于错误展示和进度条
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	// 用于保存完整的错误日志
	var stderrLog bytes.Buffer
	// 使用 TeeReader 将 stderrPipe 的内容分流：一份给 stderrLog，一份给 printProgress
	teeReader := io.TeeReader(stderrPipe, &stderrLog)

	if err := cmd.Start(); err != nil {
		return err
	}

	// 这里的 printProgress 需要读取 teeReader
	// 并且需要在 cmd.Wait() 之前完成读取，否则 Wait 会 hang 或者 pipe 不全
	// printProgress 内部是持续读取直到 EOF 的，所以没问题
	go func() {
		printProgress(io.NopCloser(teeReader), inputFile)
	}()

	if err := cmd.Wait(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		// 转换失败，打印完整的 ffmpeg 错误日志
		fmt.Printf("\n❌ FFmpeg 报错输出:\n%s\n", stderrLog.String())
		return fmt.Errorf("ffmpeg process failed: %w", err)
	}
	fmt.Println(generateProgressBar(100.0, barWidth))
	return nil
}

// buildCodecArgs 根据加速模式和目标编码返回参数
func buildCodecArgs(vcodec, accel string) []string {
	vcodec = strings.ToLower(vcodec)
	isNVENC := strings.ToLower(accel) == "cuda" || strings.ToLower(accel) == "nvenc"

	switch vcodec {
	case "h264", "x264":
		if isNVENC {
			fmt.Println("🚀 使用 NVIDIA 加速: h264_nvenc")
			return []string{"-c:v", "h264_nvenc", "-preset", "p4", "-c:a", "aac"}
		}
		fmt.Println("🐌 使用 CPU 编码: libx264")
		return []string{"-c:v", "libx264", "-c:a", "aac"}
	case "hevc", "x265", "h265":
		if isNVENC {
			fmt.Println("🚀 使用 NVIDIA 加速: hevc_nvenc")
			return []string{"-c:v", "hevc_nvenc", "-preset", "p4", "-c:a", "aac"}
		}
		fmt.Println("🐌 使用 CPU 编码: libx265")
		return []string{"-c:v", "libx265", "-c:a", "aac"}
	default:
		// 默认回退
		return []string{"-c:v", "libx264", "-c:a", "aac"}
	}
}

// createSafeTempSubtitle 读取字幕文件并将其内容写入临时目录下的一个简单命名文件
// 这样可以避免 FFmpeg 因路径包含特殊字符（空格、括号、中文等）而报错
func createSafeTempSubtitle(srcPath string) (string, error) {
	content, err := os.ReadFile(srcPath)
	if err != nil {
		return "", err
	}

	// 这里可以添加 UTF-8 转换逻辑，如果需要的话。
	// 目前直接写入，假设大多数现代字幕已经是 UTF-8 或 FFmpeg 能自动识别。

	// 创建临时文件，使用简单的命名前缀
	tmpFile, err := os.CreateTemp("", "ffsub_*.srt")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	if _, err := tmpFile.Write(content); err != nil {
		return "", err
	}

	return tmpFile.Name(), nil
}

// SafeRename Windows 安全重命名 (覆盖目标)
func SafeRename(src, dst string) error {
	// 在 Windows 上，如果 dst 存在，Rename 会失败。
	// 策略：先删 dst，再移。
	if src == dst {
		return nil
	}

	// 尝试直接重命名
	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}

	// 如果失败，可能是目标存在
	// 检查目标是否存在
	if _, err := os.Stat(dst); err == nil {
		// 存在，先删除目标
		if err := os.Remove(dst); err != nil {
			return fmt.Errorf("无法删除目标文件 %s: %w", dst, err)
		}
		// 再次尝试
		return os.Rename(src, dst)
	}

	return err
}

var (
	// 设置颜色函数
	progressColor   = color.New(color.FgGreen).SprintFunc()
	percentageColor = color.New(color.FgCyan, color.Bold).SprintfFunc()
	barWidth        = 50
)

func printProgress(stderrPipe io.ReadCloser, inp string) {
	defer stderrPipe.Close()
	duration, _ := getTotalDuration(inp)
	fmt.Printf("开始处理文件[%s]...\n", inp)

	reader := bufio.NewReaderSize(stderrPipe, 1024)
	for {
		line, err := reader.ReadString('\r')
		if err != nil {
			break
		}
		if processedTime, ok := parseFFmpegOutput(line); ok {
			// 这里可以进一步解析时间并计算进度
			percent := (processedTime / duration.Seconds()) * 100
			fmt.Print(generateProgressBar(percent, barWidth))
		}
	}
}

// parseFFmpegOutput 解析FFmpeg输出行，提取时间信息
func parseFFmpegOutput(line string) (float64, bool) {
	re := regexp.MustCompile(`time=([0-9]{2}):([0-9]{2}):([0-9]{2}.[0-9]{2})`)
	matches := re.FindStringSubmatch(line)
	if len(matches) != 4 {
		return 0, false
	}

	hours, _ := strconv.ParseFloat(matches[1], 64)
	minutes, _ := strconv.ParseFloat(matches[2], 64)
	seconds, _ := strconv.ParseFloat(matches[3], 64)

	return hours*3600 + minutes*60 + seconds, true
}

// 获取总时长以计算精确进度
func getTotalDuration(inputFile string) (time.Duration, error) {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		inputFile)

	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	duration, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	if err != nil {
		return 0, err
	}

	return time.Duration(duration * float64(time.Second)), nil
}

// 生成进度条字符串
func generateProgressBar(percent float64, barWidth int) string {
	if percent >= 100 {
		percent = 100
	}
	barFilled := int(percent / 100 * float64(barWidth))
	barEmpty := barWidth - barFilled

	return fmt.Sprintf(
		"\r%s [%s%s] %s",
		progressColor("处理文件进度:"),
		strings.Repeat("█", barFilled),
		strings.Repeat(" ", barEmpty),
		percentageColor("%.2f%%", percent),
	)
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

func ReplaceChar(name string) string {
	if ffmpegSpecialChars.MatchString(name) {
		return ffmpegSpecialChars.ReplaceAllString(name, "")
	}
	return name
}
