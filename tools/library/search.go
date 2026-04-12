package library

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/struki84/datum/config"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/tools"
	"github.com/tmc/langchaingo/vectorstores"
	"github.com/tmc/langchaingo/vectorstores/pinecone"
)

var _ tools.Tool = &FileSearchTool{}

type SearchResult struct {
	Source  string  `json:"source"`
	Chunk   int     `json:"chunk"`
	Content string  `json:"content"`
	Score   float32 `json:"score"`
}

type FileSearchTool struct {
	store pinecone.Store
}

func NewFileSearchTool() (*FileSearchTool, error) {
	config := config.New()

	llm, err := openai.New(openai.WithEmbeddingModel("text-embedding-3-small"))
	if err != nil {
		return nil, fmt.Errorf("failed to create embedder LLM: %w", err)
	}

	e, err := embeddings.NewEmbedder(llm)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedder: %w", err)
	}

	store, err := pinecone.New(
		pinecone.WithHost(config.PineconeHost),
		pinecone.WithEmbedder(e),
		pinecone.WithAPIKey(config.PineconeAPIKey),
		pinecone.WithNameSpace("datum-files"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Pinecone: %w", err)
	}

	return &FileSearchTool{
		store: store,
	}, nil
}

func (tool *FileSearchTool) Name() string {
	return "SearchFiles"
}

func (tool *FileSearchTool) Description() string {
	return "Searches the knowledge base for relevant information. Input should be a JSON object with a 'query' field."
}

func (tool *FileSearchTool) Call(ctx context.Context, input string) (string, error) {
	log.Println("Searching files with input:", input)

	var toolInput struct {
		Query string `json:"query,omitempty"`
	}

	if err := json.Unmarshal([]byte(input), &toolInput); err != nil {
		return fmt.Sprintf("invalid input: %s", err), nil
	}

	if toolInput.Query == "" {
		return "no query provided", nil
	}

	docs, err := tool.store.SimilaritySearch(ctx, toolInput.Query, 5,
		vectorstores.WithScoreThreshold(0.7),
	)
	if err != nil {
		return fmt.Sprintf("search failed: %s", err), nil
	}

	if len(docs) == 0 {
		return "no relevant results found", nil
	}

	results := make([]SearchResult, len(docs))
	for i, doc := range docs {
		source, _ := doc.Metadata["source"].(string)
		chunk, _ := doc.Metadata["chunk"].(float64) // JSON numbers decode as float64

		results[i] = SearchResult{
			Source:  source,
			Chunk:   int(chunk),
			Content: doc.PageContent,
			Score:   doc.Score,
		}
	}

	jsonBytes, err := json.Marshal(results)
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}
