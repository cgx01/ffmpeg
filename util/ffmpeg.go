package util

import (
	"bufio"
	"bytes"
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

// ConvertVideo 函数用于通用的视频格式转换
func ConvertVideo(inputFile, outputFile, subtitle string, isSub bool) error {
	// 构建 ffmpeg 命令
	var args []string
	args = append(args, "-i", inputFile)

	if isSub && subtitle != "" {
		// 如果需要烧录字幕
		// 注意：烧录字幕通常需要重编码，因此不能使用 -c copy
		// 这里的 subtitles=filename 滤镜会自动处理视频流
		
		// 修复 Windows 路径问题：FFmpeg 滤镜中路径的 : 和 \ 需要特殊处理
		// 1. 将反斜杠 \ 替换为正斜杠 /
		// 2. 将盘符后的冒号 : 转义为 \:
		
		sanitizedSub := strings.ReplaceAll(subtitle, "\\", "/")
		sanitizedSub = strings.ReplaceAll(sanitizedSub, ":", "\\:")
		
		// 使用单引号包裹路径以处理可能存在的特殊字符（如空格、括号）
		// 但注意：exec.Command 参数中的单引号会被原样传给 ffmpeg，ffmpeg 解析器会处理它
		// 为了保险，我们既然已经转义了关键字符，可以直接传值，或者加引号。
		// 最稳妥的方式是：转义关键字符 + 单引号包裹
		
		// 这里我们采用最稳妥的 FFmpeg 推荐方式：subtitles='filename'
		// 并确保 filename 里的单引号被转义（虽然 windows 路径里一般没有单引号）
		sanitizedSub = strings.ReplaceAll(sanitizedSub, "'", "'\\''")
		
		args = append(args, "-vf", fmt.Sprintf("subtitles='%s'", sanitizedSub))
	} else {
		// 如果不烧录字幕，根据输出格式决定是否指定编码器
		// 为了保持对旧逻辑的兼容性（高质量转换），如果目标是 mp4/mkv，我们默认使用 libx264 + aac
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(outputFile), "."))
		if ext == "mp4" || ext == "mkv" {
			args = append(args, "-c:v", "libx264", "-c:a", "aac")
		} else {
			// 其他格式让 ffmpeg 自动选择最佳编码器，或者用户可以通过其他方式指定（目前暂未暴露）
			// 也可以默认尝试 -c:v libx264 如果容器支持
		}
	}
	
	args = append(args, outputFile)
	cmd := exec.Command(ffmpegBin, args...)
	fmt.Printf("%v\n", cmd.Args)

	// 获取命令的标准错误输出管道
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	// 启动命令
	if err := cmd.Start(); err != nil {
		return err
	}

	// 读取标准错误输出并解析进度
	go printProgress(stderr, inputFile)

	// 等待命令执行完成
	if err := cmd.Wait(); err != nil {
		return err
	}
	// 处理完成，显示100%进度条
	fmt.Println(generateProgressBar(100.0, barWidth))
	return nil
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
