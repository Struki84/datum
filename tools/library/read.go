package library

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"

	"github.com/AstroSynapseLab/mar-mar-ai/internal/models"
	"github.com/AstroSynapseLab/mar-mar-ai/utils/loader"
	"github.com/AstroSynapseLab/mar-mar-ai/utils/storage"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/documentloaders"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/textsplitter"
	"github.com/tmc/langchaingo/tools"
)

var _ tools.Tool = &ReadFileTool{}

type ReadFileTool struct {
	llm       llms.Model
	splitter  textsplitter.TextSplitter
	documents []models.Document
}

func NewReadFileTool(llm llms.Model, docs []models.Document) (*ReadFileTool, error) {
	splitter := textsplitter.NewRecursiveCharacter()
	splitter.ChunkSize = 500
	splitter.ChunkOverlap = 50

	return &ReadFileTool{
		llm:       llm,
		documents: docs,
		splitter:  splitter,
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

	var requestedDoc models.Document
	for _, doc := range tool.documents {
		if doc.Name == toolInput.File {
			requestedDoc = doc
			break
		}
	}

	file := bytes.NewReader(requestedDoc.Contents)
	var fileLoader documentloaders.Loader

	switch filepath.Ext(requestedDoc.Name) {
	case ".pdf":
		fileLoader = loader.NewPDF(file, file.Size())
	case ".docx", ".doc", ".xlsx", ".xls", ".pptx", ".ppt":
		fileLoader = loader.NewOffice(file, file.Size(), requestedDoc.Name)
	}

	log.Println("Loading file:", requestedDoc.Name)
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

	aws, _ := storage.NewAWS()

	type toolResponse struct {
		Answer   string `json:"answer"`
		Metadata struct {
			FileName string `json:"file_name"`
			Language string `json:"language"`
			URL      string `json:"url"`
			IsPublic bool   `json:"isPublic"`
		} `json:"metadata"`
	}

	response := toolResponse{
		Answer: answer["text"].(string),
		Metadata: struct {
			FileName string `json:"file_name"`
			Language string `json:"language"`
			URL      string `json:"url"`
			IsPublic bool   `json:"isPublic"`
		}{
			FileName: requestedDoc.Name,
			Language: requestedDoc.Language,
			URL:      aws.GetTempURL(requestedDoc.Path),
			IsPublic: requestedDoc.IsPublic,
		},
	}

	jsonResponse, err := json.Marshal(response)
	if err != nil {
		return "", err
	}

	return string(jsonResponse), nil
}
