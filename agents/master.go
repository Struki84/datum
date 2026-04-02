package agents

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/struki84/datum/clipt/storage"
	"github.com/struki84/datum/clipt/tui/schema"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

type MasterAgent struct {
	LLM           *openai.LLM
	streamHandler func(ctx context.Context, chunk []byte) error
	currentModel  string
	storage       storage.SQLite
}

func NewMasterAgent(model string, storage storage.SQLite) *MasterAgent {

	llm, err := openai.New(
		openai.WithModel(model),
		openai.WithBaseURL("https://openrouter.ai/api/v1"),
		openai.WithToken(os.Getenv("OPENROUTER_API_KEY")),
	)

	if err != nil {
		fmt.Println("Can't create model:", err)
		return nil
	}

	return &MasterAgent{
		LLM:          llm,
		currentModel: model,
		storage:      storage,
	}
}

func (agent *MasterAgent) Name() string {
	return "Datum Agent"
}

func (agent *MasterAgent) Type() schema.ProviderType {
	return schema.LLM
}

func (agent *MasterAgent) Description() string {

	return "Datum Test Voice Agent"
}

func (agent *MasterAgent) Run(ctx context.Context, input string, session schema.ChatSession) error {

	buffer, err := agent.storage.LoadMsgs(session.ID)
	if err != nil {
		log.Println(err)
		return err
	}

	userMsg := schema.Msg{
		Role:      schema.UserMsg,
		Content:   input,
		Timestamp: time.Now().Unix(),
	}

	err = agent.storage.SaveMsg(session.ID, userMsg)
	if err != nil {
		log.Println(err)
		return err
	}

	content := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, "You are a helpful assistant!"),
		llms.TextParts(llms.ChatMessageTypeSystem, "CHAT HISTORY: \n"+buffer),
		llms.TextParts(llms.ChatMessageTypeHuman, input),
	}

	response, err := agent.LLM.GenerateContent(
		ctx,
		content,
		llms.WithStreamingFunc(agent.streamHandler),
	)

	if err != nil {
		fmt.Println(err)
		return err
	}

	aiMsg := schema.Msg{
		Role:      schema.AIMsg,
		Content:   response.Choices[0].Content,
		Timestamp: time.Now().Unix(),
	}

	err = agent.storage.SaveMsg(session.ID, aiMsg)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func (agent *MasterAgent) Stream(ctx context.Context, callback func(ctx context.Context, msg schema.Msg) error) {
	agent.streamHandler = func(ctx context.Context, chunk []byte) error {
		callback(ctx, schema.Msg{
			Stream:    true,
			Role:      schema.AIMsg,
			Content:   string(chunk),
			Timestamp: time.Now().Unix(),
		})

		return nil
	}
}
