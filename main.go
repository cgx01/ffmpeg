package main

import (
	"ffmpeg-tool/internal/commands"
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

	// 禁止交叉解析：遇到第一个非 flag 参数（即子命令）后停止解析
	// 这样子命令的参数（如 --dry-run）就不会被 main 的 parser 误读或报错
	pflag.CommandLine.SetInterspersed(false)

	// 解析 flag
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
	fmt.Println("=========================================================")
	fmt.Println("      ffmpeg-tool - 自动化多媒体处理工具集")
	fmt.Println("=========================================================")
	fmt.Println("\n用法:")
	fmt.Println("  ffmpeg-tool [全局参数] <命令> [命令参数]")

	fmt.Println("\n全局参数:")
	fmt.Println("  --ffmpeg <路径>    指定 ffmpeg bin 目录 (例如 C:\\ffmpeg\\bin)")
	fmt.Println("                     程序会将其添加到 PATH，并验证 ffmpeg/ffprobe 是否可用")

	fmt.Println("\n可用命令及参数说明:")

	fmt.Println("\n  convert [路径...] [参数]")
	fmt.Println("    视频格式转换及字幕自动嵌入")
	fmt.Println("    参数:")
	fmt.Println("      -d, --dirs <路径>    指定目录或文件 (支持多个，也可直接作为位置参数)")
	fmt.Println("      --to <格式>          目标容器格式 (默认: mp4)")
	fmt.Println("      --vcodec <编码>      视频编码: h264, hevc (默认: h264)")
	fmt.Println("      --accel <模式>       硬件加速: cuda (N卡), qsv (Intel), amf (AMD)")
	fmt.Println("      --workers <数量>     并发处理任务数 (默认: 1)")
	fmt.Println("      --force              强制重新转换 (即使格式已匹配且无字幕)")
	fmt.Println("      --dry-run            预演模式，不实际修改文件")

	fmt.Println("\n  unzip [参数]")
	fmt.Println("    自动化递归解压 7z/zip/rar 压缩包")
	fmt.Println("    参数:")
	fmt.Println("      -d, --dir <路径>     指定解压根目录")
	fmt.Println("      -p, --password <密>  指定解压密码 (默认: momo.moe)")

	fmt.Println("\n  gif [参数]")
	fmt.Println("    GIF 压缩或视频转 GIF")
	fmt.Println("    参数:")
	fmt.Println("      -d, --dir <路径>     指定处理目录")
	fmt.Println("      -s, --size <大小>    限制最大体积 (默认: 9M)")

	fmt.Println("\n  flatten -d <路径>")
	fmt.Println("    将子目录中的文件移动到当前目录（扁平化结构）")

	fmt.Println("\n  stats -d <路径>")
	fmt.Println("    统计目录下的图片和视频数量，并清理种子文件")

	fmt.Println("\n使用示例:")
	fmt.Println("  1. 基础转换 (单线程):")
	fmt.Println("     ffmpeg-tool convert \"D:\\Videos\"")
	fmt.Println("\n  2. 显卡加速 + 3并发 + 强制 H.264 编码:")
	fmt.Println("     ffmpeg-tool convert --accel cuda --workers 3 --vcodec h264 \"D:\\Movies\"")
	fmt.Println("\n  3. 预演模式 (查看哪些文件会被处理):")
	fmt.Println("     ffmpeg-tool convert --dry-run \"D:\\Test\"")
	fmt.Println("\n  4. 指定密码解压目录:")
	fmt.Println("     ffmpeg-tool unzip -d \"E:\\Downloads\" -p \"123456\"")
	fmt.Println("\n  5. 指定 ffmpeg 路径运行:")
	fmt.Println("     ffmpeg-tool --ffmpeg \"C:\\ffmpeg\\bin\" convert \"E:\\NewFiles\"")
	fmt.Println("=========================================================")
}
