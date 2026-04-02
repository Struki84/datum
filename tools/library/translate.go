package library

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/AstroSynapseLab/mar-mar-ai/internal/models"
	"github.com/AstroSynapseLab/mar-mar-ai/internal/repositories"
	"github.com/AstroSynapseLab/mar-mar-ai/sdk/crud/database"
	"github.com/AstroSynapseLab/mar-mar-ai/utils/storage"
	"github.com/AstroSynapseLab/mar-mar-ai/utils/translator"
	"github.com/tmc/langchaingo/tools"
)

var _ tools.Tool = &TranslatefileTool{}

type TranslatefileTool struct {
	translator translator.GoogleTranslator
	documents  []models.Document
	mimeTypes  map[string]string
	storage    *storage.AWS
	docsRepo   *repositories.DocumentsRepository
	chroma     *storage.ChromaStore
	apiKey     string
}

func NewTranslateFileTool(docs []models.Document, db *database.Database, apiKey string) (*TranslatefileTool, error) {
	host := os.Getenv("CHROMA_HOST")
	port := os.Getenv("CHROMA_PORT")
	url := fmt.Sprintf("http://%s:%s", host, port)

	chroma, err := storage.NewChromaStore(url)
	if err != nil {
		log.Println("Error creating chroma client:", err)
		return nil, err
	}

	chroma.SetupDB(context.Background())

	translator, err := translator.NewGoogleTranslator()
	if err != nil {
		log.Println("Error creating translator:", err)
		return nil, err
	}

	storage, err := storage.NewAWS()
	if err != nil {
		log.Println("Error creating storage:", err)
		return nil, err
	}

	return &TranslatefileTool{
		documents:  docs,
		chroma:     chroma,
		storage:    storage,
		docsRepo:   repositories.NewDocumentsRepository(db),
		translator: *translator,
		apiKey:     apiKey,
		mimeTypes: map[string]string{
			".txt":  "text/plain",
			".pdf":  "application/pdf",
			".doc":  "application/msword",
			".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			".xls":  "application/vnd.ms-excel",
			".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			".ppt":  "application/vnd.ms-powerpoint",
			".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		},
	}, nil
}

func (tool *TranslatefileTool) Name() string {
	return "TranslateFile"
}

func (tool *TranslatefileTool) Description() string {
	return "Translates a file."
}

func (tool *TranslatefileTool) Call(ctx context.Context, input string) (string, error) {
	log.Println("Translating file with input:", input)
	var toolInput struct {
		File string `json:"file,omitempty"`
		Lang string `json:"lang,omitempty"`
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

	mimeType := tool.mimeTypes[filepath.Ext(requestedDoc.Name)]
	inputDoc := map[string]any{
		"targetLang": toolInput.Lang,
		"content":    requestedDoc.Contents,
		"mimeType":   mimeType,
	}

	result, err := tool.translator.TranslateDocument(inputDoc)
	if err != nil {
		fmt.Println(err)
		return fmt.Sprintf("%v: %s", "failed to translate document", err), nil
	}

	newFilename := fmt.Sprintf("%s.%s", toolInput.Lang, requestedDoc.Name)
	key := fmt.Sprintf("%v.%s", requestedDoc.UserID, newFilename)

	location, err := tool.storage.UploadFile(key, result["content"].([]byte))
	if err != nil {
		log.Println("Error uploading file:", err)
		return "", err
	}

	newDoc := &models.Document{
		Path:        location,
		ParentID:    requestedDoc.ID,
		Name:        newFilename,
		Description: "translated",
		Language:    toolInput.Lang,
		Status:      "COMPLETED",
		IsPublic:    false,
		UserID:      requestedDoc.UserID,
		User:        requestedDoc.User,
		Contents:    result["content"].([]byte),
	}

	err = tool.docsRepo.SaveDocument(newDoc)
	if err != nil {
		log.Println("Error saving document:", err)
		return "", err
	}

	user := fmt.Sprintf("%d-%s", newDoc.UserID, requestedDoc.User.Username)
	err = tool.chroma.SaveFile(newDoc, user, tool.apiKey)
	if err != nil {
		log.Println("Error saving file to chroma DB:", err)
		return "", err
	}

	URL := tool.storage.GetTempURL(location)

	response := fmt.Sprintf("%s translated to %s, saved as %s, URL is: %s", requestedDoc.Name, toolInput.Lang, newFilename, URL)
	return response, nil
}
