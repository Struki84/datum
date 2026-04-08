package library

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/struki84/datum/config"
	"github.com/struki84/datum/tools/library/loaders"
	"github.com/tmc/langchaingo/documentloaders"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/textsplitter"
	"github.com/tmc/langchaingo/vectorstores/pinecone"
)

type FileSentry struct {
	store   pinecone.Store
	dirPath string
}

func NewFileSentry(dirPath string) *FileSentry {
	config := config.New()

	llm, err := openai.New(openai.WithEmbeddingModel("text-embedding-3-small"))
	if err != nil {
		log.Fatal(err)
	}

	e, err := embeddings.NewEmbedder(llm)
	if err != nil {
		log.Fatal(err)
	}

	store, err := pinecone.New(
		pinecone.WithHost(config.PineconeHost),
		pinecone.WithEmbedder(e),
		pinecone.WithAPIKey(config.PineconeAPIKey),
		pinecone.WithNameSpace("datum-files"),
	)
	if err != nil {
		log.Fatal(err)
	}

	return &FileSentry{
		store:   store,
		dirPath: dirPath,
	}
}

func (sentry *FileSentry) SaveFile(ctx context.Context, path string) error {
	fileName := filepath.Base(path)
	log.Printf("Indexing file: %s", fileName)

	docs, err := loadAndSplitFile(path)
	if err != nil {
		return fmt.Errorf("failed to load file %s: %w", fileName, err)
	}

	// Tag each chunk with the source file and a unique ID based on filename
	for i := range docs {
		docs[i].Metadata["source"] = fileName
		docs[i].Metadata["chunk"] = i
		docs[i].Metadata["id"] = fmt.Sprintf("%s_chunk_%d", fileName, i)
	}

	_, err = sentry.store.AddDocuments(ctx, docs)
	if err != nil {
		return fmt.Errorf("failed to store documents: %w", err)
	}

	log.Printf("Indexed file: %s (%d chunks)", fileName, len(docs))
	return nil
}

func (sentry *FileSentry) ScanFiles(ctx context.Context) error {
	return filepath.Walk(sentry.dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		fileName := filepath.Base(path)

		// Check if first chunk already exists in Pinecone
		results, searchErr := sentry.store.SimilaritySearch(ctx, fileName, 1)
		if searchErr == nil && len(results) > 0 {
			if source, ok := results[0].Metadata["source"]; ok && source == fileName {
				log.Printf("Skipping already indexed file: %s", fileName)
				return nil
			}
		}

		if err := sentry.SaveFile(ctx, path); err != nil {
			log.Printf("Error indexing file %s: %v", path, err)
		}
		return nil
	})
}

func (sentry *FileSentry) WatchFiles(ctx context.Context) error {
	log.Println("Watching files...")

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("error creating watcher: %w", err)
	}
	defer watcher.Close()

	if err := watcher.Add(sentry.dirPath); err != nil {
		return fmt.Errorf("error watching directory: %w", err)
	}

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Create == fsnotify.Create {
					info, err := os.Stat(event.Name)
					if err != nil {
						log.Println("Error getting file info:", err)
						continue
					}
					if !info.IsDir() {
						log.Println("New file detected:", event.Name)
						if err := sentry.SaveFile(ctx, event.Name); err != nil {
							log.Println("Error indexing file:", err)
						}
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("Watcher error:", err)
			}
		}
	}()

	<-ctx.Done()
	log.Println("Stopping watcher...")
	return ctx.Err()
}

// loadAndSplitFile reads a file from disk and splits it into document chunks.
// Adjust the loader types to match what your app supports.
func loadAndSplitFile(path string) ([]schema.Document, error) {
	fileBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	reader := bytes.NewReader(fileBytes)
	var loader documentloaders.Loader

	switch filepath.Ext(path) {
	case ".pdf":
		loader = loaders.NewPDF(reader, reader.Size())
	case ".docx", ".doc", ".xlsx", ".xls", ".pptx", ".ppt":
		loader = loaders.NewOffice(reader, reader.Size(), filepath.Base(path))
	}

	splitter := textsplitter.NewRecursiveCharacter(
		textsplitter.WithChunkSize(1000),
		textsplitter.WithChunkOverlap(200),
	)

	return loader.LoadAndSplit(context.Background(), splitter)
}
