package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

var path = "D:\\BaiduNetdiskDownload\\NPXVIP(Namprikk) – 作品大合集 [97.5GB] 更新： 2月2日"

func main() {
	dir, _ := os.ReadDir(path)
	for _, f := range dir {
		if f.IsDir() {
			dirs, _ := os.ReadDir(filepath.Join(path, f.Name()))
			for _, d := range dirs {
				src := filepath.Join(path, f.Name(), d.Name())
				dst := filepath.Join(path, d.Name())
				fmt.Printf("src: %s, dst: %s\n", src, dst)
				os.Rename(src, dst)
				os.Remove(filepath.Join(path, f.Name()))
			}
		}
	}
}
func command(cmd *exec.Cmd) error {
	fmt.Printf("command:%s\n", cmd)
	// 执行命令并捕获输出
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("解压失败: %v\n", err)
		fmt.Println("命令输出:", string(output))
		return err
	}
	return nil
}
