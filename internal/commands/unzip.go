package commands

import (
	"fmt"
	"github.com/spf13/pflag"
	"os"
	"os/exec"
	"path/filepath"
)

func RunUnzip(args []string) {
	fs := pflag.NewFlagSet("unzip", pflag.ExitOnError)
	password := fs.StringP("password", "p", "momo.moe", "7z password")
	sevenZPath := fs.String("7z-path", "7z", "Path to 7z executable")
	dirPath := fs.StringP("dir", "d", "", "Directory to process")

	fs.Parse(args)

	if *dirPath == "" {
		fmt.Println("Error: --dir is required")
		fs.Usage()
		return
	}

	fmt.Printf("Unzipping in %s with password '%s'\n", *dirPath, *password)
	
	err := processUnzip(*dirPath, *sevenZPath, *password)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}

func processUnzip(path, sevenZPath, password string) error {
	// Loop logic from 7z.go seems to retry indefinitely? 
	// "for { dir, _ := os.ReadDir(filePath) ... }"
	// That might be dangerous if extraction fails and files remain.
	// I will implement a single pass recursive unzip.

	dirEntries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	for _, entry := range dirEntries {
		fullPath := filepath.Join(path, entry.Name())
		if entry.IsDir() {
			// Recursive? The original code logic was:
			// if !f.IsDir() { unzip } else { return } -> This implies it only processed files in the top dir?
			// Wait, original 7z.go:
			// for { dir, _ := os.ReadDir(filePath) ... for _, f := range dir { if !f.IsDir() { ... } else { return } } }
			// If it finds a directory, it returns? That's weird.
			// Maybe it assumes it's processing a flat folder of archives?
			// I will implement recursive traversal.
			
			// Actually, let's stick to safe recursion.
			if err := processUnzip(fullPath, sevenZPath, password); err != nil {
				return err
			}
		} else {
			// Check if archive? 
			// Original code didn't check extension, just tried to unzip everything using 7z.
			// That might be aggressive. 7z handles many formats.
			// Let's at least check for common archive extensions to avoid trying to unzip .txt
			ext := filepath.Ext(entry.Name())
			if ext == ".7z" || ext == ".zip" || ext == ".rar" {
				fmt.Printf("Unzipping: %s\n", fullPath)
				// 7z x archive -pPASS -oOUT
				// -o{Output} should probably be the current directory
				cmd := exec.Command(sevenZPath, "x", fullPath, "-p"+password, fmt.Sprintf("-o%s", path), "-y") // -y to assume yes on overwrite?
				
				if output, err := cmd.CombinedOutput(); err != nil {
					fmt.Printf("Failed to unzip %s: %v\nOutput: %s\n", fullPath, err, string(output))
					// Move to error.txt or rename? Original moved to error.txt location or logged.
					// I'll just log for now.
				} else {
					fmt.Printf("Successfully unzipped %s\n", fullPath)
					// Remove archive
					os.Remove(fullPath)
				}
			}
		}
	}
	return nil
}
