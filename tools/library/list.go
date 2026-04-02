package library

import (
	"context"
	"log"

	"github.com/AstroSynapseLab/mar-mar-ai/internal/models"
	"github.com/AstroSynapseLab/mar-mar-ai/utils/storage"
	"github.com/tmc/langchaingo/tools"
)

var _ tools.Tool = &FileListTool{}

type FileListTool struct {
	documents []models.Document
}

func NewFileListTool(docs []models.Document) (*FileListTool, error) {
	return &FileListTool{
		documents: docs,
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

	if len(tool.documents) == 0 {
		return "No files found", nil
	}

	awsStorage, _ := storage.NewAWS()

	var str string
	for _, doc := range tool.documents {
		URL := awsStorage.GetTempURL(doc.Path)
		str += "- " + doc.Name + ", URL: " + URL + "\n"
	}

	return str, nil
}
