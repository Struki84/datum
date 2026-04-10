package library

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/struki84/datum/tools/library/loaders"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/documentloaders"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/textsplitter"
	"github.com/tmc/langchaingo/tools"
)

var _ tools.Tool = &ReadFileTool{}

type ReadFileTool struct {
	llm      llms.Model
	splitter textsplitter.TextSplitter
	files    []string
}

func NewReadFileTool(llm llms.Model, files []string) (*ReadFileTool, error) {
	splitter := textsplitter.NewRecursiveCharacter()
	splitter.ChunkSize = 500
	splitter.ChunkOverlap = 50

	return &ReadFileTool{
		llm:      llm,
		splitter: splitter,
		files:    files,
	}, nil
}

func (*ReadFileTool) Name() string {
	return "ReadFile"
}

func (tool *ReadFileTool) Description() string {
	return "Reads a file."
}

func (tool *ReadFileTool) Call(ctx context.Context, input string) (string, error) {
	log.Println("Reading file with input:", input)

	var toolInput struct {
		File  string `json:"file,omitempty"`
		Query string `json:"query,omitempty"`
	}

	err := json.Unmarshal([]byte(input), &toolInput)
	if err != nil {
		fmt.Println(err)
		return fmt.Sprintf("%v: %s", "invalid input", err), nil
	}

	var requestedFile string
	for _, fileName := range tool.files {
		if fileName == toolInput.File {
			requestedFile = fileName
			break
		}
	}

	if requestedFile == "" {
		return fmt.Sprintf("file not found: %s", toolInput.File), nil
	}

	// tmp files location basePath
	basePath := "./files"

	filePath := filepath.Join(basePath, requestedFile)
	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Sprintf("error reading file: %s", err), nil
	}

	file := bytes.NewReader(fileBytes)

	var fileLoader documentloaders.Loader
	switch filepath.Ext(requestedFile) {
	case ".pdf":
		fileLoader = loaders.NewPDF(file, file.Size())
	case ".docx", ".doc", ".xlsx", ".xls", ".pptx", ".ppt":
		fileLoader = loaders.NewOffice(file, file.Size(), requestedFile)
	}

	log.Println("Loading file:", requestedFile)
	docs, err := fileLoader.LoadAndSplit(ctx, tool.splitter)
	if err != nil {
		log.Println("Error reading file:", err)
		return "", err
	}

	if toolInput.Query == "" {
		toolInput.Query = "Provide the summary of the file."
	}

	QAChain := chains.LoadStuffQA(tool.llm)
	answer, err := chains.Call(ctx, QAChain, map[string]any{
		"input_documents": docs,
		"question":        toolInput.Query,
	})

	if err != nil {
		return "", err
	}

	return answer["text"].(string), nil
}
