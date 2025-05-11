package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

var password = "momo.moe"
var filePath = "F:\\NPXVIP(Namprikk) – 作品大合集 [100GB] 更新： 2月18日"
var zfile = "D:\\BaiduNetdiskDownload\\NPXVIP(Namprikk) – 作品大合集 [97.5GB] 更新： 2月2日"

func main() {
	for {
		dir, _ := os.ReadDir(filePath)
		for _, f := range dir {
			if !f.IsDir() {
				join := filepath.Join(filePath, f.Name())
				cmd := exec.Command("D:\\software\\7-Zip\\7z.exe", "x", join, "-p"+password, fmt.Sprintf("-o%s", filePath))
				if err := Command(cmd); err != nil {
					mvCmd := exec.Command("move", join, "D:\\error.txt")
					Command(mvCmd)
					sprintf := fmt.Sprintf("【%s】解压失败\n", join)
					file, _ := os.OpenFile("err.txt", os.O_APPEND|os.O_WRONLY, 0666)
					file.WriteString(sprintf)
				} else {
					os.Remove(join)
				}
			} else {
				return
			}
		}
	}
}

func Command(cmd *exec.Cmd) error {
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
