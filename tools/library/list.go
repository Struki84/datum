package library

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/tmc/langchaingo/tools"
)

var _ tools.Tool = &FileListTool{}

type FileListTool struct {
	filesDir string
	files    []string
}

func NewFileListTool(dir string) (*FileListTool, error) {
	return &FileListTool{
		filesDir: dir,
	}, nil
}

func (tool *FileListTool) Name() string {
	return "ListFiles"
}

func (tool *FileListTool) Description() string {
	return "Lists all files you can read and search."
}

func (tool *FileListTool) Call(ctx context.Context, input string) (string, error) {
	log.Printf("Listing files...")

	entries, err := os.ReadDir(tool.filesDir)
	if err != nil {
		return fmt.Sprintf("error reading files directory %s", err), nil
	}

	tool.files = []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		tool.files = append(tool.files, entry.Name())
	}

	if len(tool.files) == 0 {
		return "No files found", nil
	}

	var str string
	for _, fileName := range tool.files {
		str += "- " + fileName + "\n"
	}

	return str, nil
}
