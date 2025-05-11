package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func embedSubtitles(inputVideoPath, outputVideoPath, subtitlePath string) error {
	// 检查文件是否存在
	_, err := os.Stat(inputVideoPath)
	if os.IsNotExist(err) {
		log.Printf("输入视频文件 %s 不存在", inputVideoPath)
		return err
	}
	_, err = os.Stat(subtitlePath)
	if os.IsNotExist(err) {
		log.Printf("字幕文件 %s 不存在", subtitlePath)
		return err
	}

	////将路径中的反斜杠替换为正斜杠
	//inputVideoPath = strings.ReplaceAll(inputVideoPath, "\\", "\"")
	//subtitlePath = strings.ReplaceAll(subtitlePath, "\\", "\"")
	//outputVideoPath = strings.ReplaceAll(outputVideoPath, "\\", "\"")
	//
	//// 对路径进行引号处理
	//inputVideoPath = fmt.Sprintf("\"%s\"", inputVideoPath)
	//subtitlePath = fmt.Sprintf("\"%s\"", subtitlePath)
	//outputVideoPath = fmt.Sprintf("\"%s\"", outputVideoPath)
	//ffmpeg -i D:\BaiduNetdiskDownload\这个空姐很带劲\EP 1\BrazzersExxtra.24.12.13.Angela.White.This.Flight.Attendant.Fucks.Part.1.XXX.1080p.MP4-WRB[XC]\brazzersexxtra.24.12.13.angela.white.this.flight.attendant.fucks.part.1.mp4 -vf subtitles=D:\BaiduNetdiskDownload\这个空姐很带劲\EP 1\BrazzersExxtra.24.12.13.Angela.White.This.Flight.Attendant.Fucks.Part.1.XXX.1080p.MP4-WRB[XC]\brazzersexxtra.srt D:\BaiduNetdiskDownload\这个空姐很带劲\EP 1\brazzersexxtra.24.12.13.angela.white.this.flight.attendant.fucks.part.1.mp4
	// 构建 FFmpeg 命令
	cmd := exec.Command("ffmpeg", "-i", inputVideoPath, "-vf", fmt.Sprintf("'subbtitles=%s'", subtitlePath), outputVideoPath)
	fmt.Printf("执行命令: %v\n", cmd.Args)
	// 创建缓冲区用于存储命令的标准输出和标准错误输出
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	// 执行命令
	err = cmd.Run()
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

func main() {
	//subtitles, _ := util.CheckVideoHasSubtitles("F:\\影视\\篠田優\\ACHJ-002-uncensored-HD\\ACHJ-002-uncensored-nyap2p.com.mp4")
	//fmt.Printf("%v\n", subtitles)

	inputFile := "F:\\影视\\篠田優\\ACHJ-002-uncensored-HD\\ACHJ-002-uncensored-nyap2p.com.mkv"
	baseName := strings.TrimSuffix(inputFile, filepath.Ext(inputFile))
	outputFile := baseName + ".mp4"
	fmt.Printf("%s\n%s\n%s\n", baseName, outputFile, filepath.Ext(inputFile))
}
