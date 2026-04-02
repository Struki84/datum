package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/struki84/clipt/storage"
	"github.com/struki84/clipt/tui/schema"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
)

type Anthropic struct {
	LLM           *anthropic.LLM
	streamHandler func(ctx context.Context, chunk []byte) error
	currentModel  string
	storage       storage.SQLite
}

func NewAnthropic(model string, storage storage.SQLite) *Anthropic {
	llm, err := anthropic.New(anthropic.WithModel(model))
	if err != nil {
		log.Printf("can't create model: %v", err)
		return nil
	}

	return &Anthropic{
		LLM:          llm,
		currentModel: model,
		storage:      storage,
	}
}

func (model *Anthropic) Type() schema.ProviderType {
	return schema.LLM
}

func (model *Anthropic) Name() string {
	return model.currentModel
}

func (model *Anthropic) Description() string {
	desc := fmt.Sprintf("%s by Anthropic", model.currentModel)
	return desc
}

func (model *Anthropic) Stream(ctx context.Context, callback func(ctx context.Context, msg schema.Msg) error) {
	model.streamHandler = func(ctx context.Context, chunk []byte) error {
		callback(ctx, schema.Msg{
			Stream:    true,
			Role:      schema.AIMsg,
			Content:   string(chunk),
			Timestamp: time.Now().Unix(),
		})

		return nil
	}
}

func (model *Anthropic) Run(ctx context.Context, input string, session schema.ChatSession) error {
	buffer, err := model.storage.LoadMsgs(session.ID)
	if err != nil {
		log.Println(err)
		return err
	}

	userMsg := schema.Msg{
		Role:      schema.UserMsg,
		Content:   input,
		Timestamp: time.Now().Unix(),
	}

	err = model.storage.SaveMsg(session.ID, userMsg)
	if err != nil {
		log.Println(err)
		return err
	}

	content := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, "You are a helpful assistant!"),
		llms.TextParts(llms.ChatMessageTypeSystem, "CHAT HISTORY: \n"+buffer),
		llms.TextParts(llms.ChatMessageTypeHuman, input),
	}

	response, err := model.LLM.GenerateContent(ctx, content, llms.WithStreamingFunc(model.streamHandler))
	if err != nil {
		fmt.Println(err)
		return err
	}

	aiMsg := schema.Msg{
		Role:      schema.AIMsg,
		Content:   response.Choices[0].Content,
		Timestamp: time.Now().Unix(),
	}

	err = model.storage.SaveMsg(session.ID, aiMsg)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}
