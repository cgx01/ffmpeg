package main

import (
	"ffmpeg/internal/commands"
	"fmt"
	"github.com/spf13/pflag"
	"log"
	"os"
	"os/exec"
)

var ffmpegBin string

func main() {
	// 定义全局 flag
	pflag.StringVar(&ffmpegBin, "ffmpeg", "", "指定 ffmpeg 的 bin 目录路径 (将添加到 PATH)")
	
	// 自定义 Usage，避免 pflag 默认输出误导用户以为只能传 flag
	pflag.Usage = printUsage

	// 解析 flag，注意：pflag 默认遇到非 flag 参数就会停止解析（如果 Interspersed 为 false），
	// 或者解析所有 flag（默认为 true）。我们希望解析全局 flag，剩下的作为子命令。
	pflag.Parse()

	// 处理 ffmpeg 路径设置
	if ffmpegBin != "" {
		if err := setupFFmpeg(ffmpegBin); err != nil {
			log.Fatalf("配置 ffmpeg 失败: %v", err)
		}
	}

	// 获取剩余参数（非全局 flag 的参数）
	args := pflag.Args()
	if len(args) < 1 {
		printUsage()
		os.Exit(1)
	}

	subCmd := args[0]
	subArgs := args[1:]

	switch subCmd {
	case "convert":
		commands.RunConvert(subArgs)
	case "unzip":
		commands.RunUnzip(subArgs)
	case "gif":
		commands.RunGif(subArgs)
	case "flatten":
		commands.RunFlatten(subArgs)
	case "stats":
		commands.RunStats(subArgs)
	case "help":
		printUsage()
	default:
		fmt.Printf("未知命令: %s\n", subCmd)
		printUsage()
		os.Exit(1)
	}
}

func setupFFmpeg(binPath string) error {
	// 将 ffmpeg 目录添加到 PATH
	currentPath := os.Getenv("PATH")
	newPath := currentPath + string(os.PathListSeparator) + binPath
	if err := os.Setenv("PATH", newPath); err != nil {
		return fmt.Errorf("设置环境变量失败: %w", err)
	}

	// 验证 ffmpeg 是否可用
	cmd := exec.Command("ffmpeg", "-version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("验证 ffmpeg 命令失败: %v\n输出: %s", err, string(output))
	}
	
	// 验证 ffprobe 是否可用 (util 包中也会用到)
	// 虽然 ffmpeg.go 只检查了 ffmpeg，但 util 包里有 ffprobe 调用
	cmdProbe := exec.Command("ffprobe", "-version")
	if err := cmdProbe.Run(); err != nil {
		fmt.Println("警告: 未能在该目录下找到 ffprobe，部分功能可能受限。")
	}

	return nil
}

func printUsage() {
	fmt.Println("用法: ffmpeg-tool [global-flags] <command> [command-args]")
	fmt.Println("\n全局参数:")
	pflag.PrintDefaults()
	fmt.Println("\n可用命令:")
	fmt.Println("  convert   视频格式转换 (自动嵌入同名字幕)")
	fmt.Println("  unzip     递归解压 7z 文件")
	fmt.Println("  gif       GIF 压缩与生成")
	fmt.Println("  flatten   扁平化目录结构")
	fmt.Println("  stats     统计目录文件类型")
	fmt.Println("\n示例:")
	fmt.Println("  ffmpeg-tool --ffmpeg \"C:\\ffmpeg\\bin\" convert -d \"C:\\Video\"")
}
