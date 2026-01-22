package commands

import (
	"fmt"
	"github.com/spf13/pflag"
	"os"
	"path/filepath"
)

func RunFlatten(args []string) {
	fs := pflag.NewFlagSet("flatten", pflag.ExitOnError)
	dirPath := fs.StringP("dir", "d", "", "Root directory to flatten")
	
	fs.Parse(args)

	if *dirPath == "" {
		fmt.Println("Error: --dir is required")
		fs.Usage()
		return
	}

	fmt.Printf("Flattening directory: %s\n", *dirPath)
	if err := flattenDir(*dirPath); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}

func flattenDir(root string) error {
	dirEntries, err := os.ReadDir(root)
	if err != nil {
		return err
	}

	for _, entry := range dirEntries {
		if entry.IsDir() {
			subDir := filepath.Join(root, entry.Name())
			
			// Process subdir first (recursive)
			// Wait, original logic just moved files from subdirs to root?
			// "dirs, _ := os.ReadDir(filepath.Join(path, f.Name())); for _, d := range dirs { src := ...; dst := ...; os.Rename }"
			// It moved files from Immediate Subdir -> Root.
			// Let's replicate that logic as it's safer than full recursive flatten without understanding intent.
			
			subDirEntries, err := os.ReadDir(subDir)
			if err != nil {
				fmt.Printf("Error reading subdir %s: %v\n", subDir, err)
				continue
			}
			
			for _, subEntry := range subDirEntries {
				src := filepath.Join(subDir, subEntry.Name())
				dst := filepath.Join(root, subEntry.Name())
				
				fmt.Printf("Moving %s -> %s\n", src, dst)
				if err := os.Rename(src, dst); err != nil {
					fmt.Printf("Failed to move %s: %v\n", src, err)
				}
			}
			// Remove subdir if empty (or just try to remove it)
			os.Remove(subDir)
		}
	}
	return nil
}
