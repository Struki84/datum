package library

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/AstroSynapseLab/mar-mar-ai/internal/models"
	"github.com/AstroSynapseLab/mar-mar-ai/utils/storage"
	"github.com/tmc/langchaingo/tools"
)

var _ tools.Tool = &FileSearchTool{}

type FileSearchTool struct {
	Chroma storage.VectorStore
	Docs   []models.Document
	apiKey string
	user   string
}

func NewSerchFilesTool(docs []models.Document, openAIAPIKey string, user string) (*FileSearchTool, error) {
	host := os.Getenv("CHROMA_HOST")
	port := os.Getenv("CHROMA_PORT")
	url := fmt.Sprintf("http://%s:%s", host, port)

	chroma, err := storage.NewChromaStore(url)
	if err != nil {
		return nil, err
	}

	chroma.SetupDB(context.Background())

	return &FileSearchTool{
		Chroma: chroma,
		Docs:   docs,
		apiKey: openAIAPIKey,
		user:   user,
	}, nil
}

func (tool *FileSearchTool) Name() string {
	return "SearchFiles"
}

func (tool *FileSearchTool) Description() string {
	return "Searches for files."
}

func (tool *FileSearchTool) Call(ctx context.Context, input string) (string, error) {
	log.Println("Searching for files with input:", input)

	var toolInput struct {
		Query string `json:"query,omitempty"`
	}

	err := json.Unmarshal([]byte(input), &toolInput)
	if err != nil {
		return fmt.Sprintf("%v: %s", "invalid input", err), nil
	}

	results, err := tool.Chroma.SearchFiles(tool.Docs, toolInput.Query, tool.user, tool.apiKey)
	if err != nil {
		return "", err
	}

	type foundDoc struct {
		ID          uint   `json:"id,omitempty"`
		Name        string `json:"name,omitempty"`
		URL         string `json:"url,omitempty"`
		Description string `json:"description,omitempty"`
		Language    string `json:"language,omitempty"`
		IsPublic    bool   `json:"isPublic,omitempty"`
		Content     string `json:"content,omitempty"`
		Page        string `json:"page,omitempty"`
		TotalPages  string `json:"totalPages,omitempty"`
	}

	SearchResults := []foundDoc{}

	awsStorage, _ := storage.NewAWS()

	for _, result := range results {
		ID, _ := strconv.ParseUint(result["docID"].(string), 10, 64)

		for _, doc := range tool.Docs {
			if doc.ID == uint(ID) {
				SearchResults = append(SearchResults, foundDoc{
					ID:          doc.ID,
					Name:        doc.Name,
					URL:         awsStorage.GetTempURL(doc.Path),
					Description: doc.Description,
					Language:    doc.Language,
					IsPublic:    doc.IsPublic,
					Content:     result["content"].(string),
					Page:        result["page"].(string),
					TotalPages:  result["totalPages"].(string),
				})
			}
		}
	}

	jsonBytes, err := json.Marshal(SearchResults)
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}
