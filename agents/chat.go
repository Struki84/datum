package agents

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/struki84/datum/clipt/storage"
	"github.com/struki84/datum/clipt/tui/schema"
	"github.com/struki84/datum/sdk/glg/graph"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

type ChatCallback struct {
	graph.SimpleCallback
	stream func(ctx context.Context, msg schema.Msg)
}

func NewChatCallback() *ChatCallback {
	return &ChatCallback{}
}

func (handler *ChatCallback) ReadStreamFunc(ctx context.Context, callback func(context.Context, schema.Msg) error) {
	handler.stream = func(ctx context.Context, msg schema.Msg) {
		callback(ctx, msg)
	}
}

func (handler *ChatCallback) HandleNodeStart(ctx context.Context, node string, initialState []llms.MessageContent) {
	// if node == "Content" {
	// 	msg := schema.Msg{
	// 		Stream:    false,
	// 		Role:      schema.AIMsg,
	// 		Content:   "",
	// 		Timestamp: time.Now().Unix(),
	// 	}
	//
	// 	handler.stream(ctx, msg)
	// }
}

func (handler *ChatCallback) HandleNodeEnd(ctx context.Context, node string, finalState []llms.MessageContent) {

}

func (handler *ChatCallback) HandleNodeStream(ctx context.Context, node string, chunk []byte) {
	if node == "Content" {
		msg := schema.Msg{
			Stream:    true,
			Role:      schema.AIMsg,
			Content:   string(chunk),
			Timestamp: time.Now().Unix(),
		}

		handler.stream(ctx, msg)
	}
}

type ChatAgent struct {
	LLM           *openai.LLM
	streamHandler func(ctx context.Context, msg schema.Msg) error
	currentModel  string
	storage       storage.SQLite
}

func NewChatAgent(model string, storage storage.SQLite) *ChatAgent {
	llm, err := openai.New(
		openai.WithModel(model),
		openai.WithBaseURL("https://openrouter.ai/api/v1"),
		openai.WithToken(os.Getenv("OPENROUTER_API_KEY")),
	)

	if err != nil {
		fmt.Println("Can't create model:", err)
		return nil
	}

	return &ChatAgent{
		LLM:          llm,
		currentModel: model,
		storage:      storage,
	}
}

func (agent *ChatAgent) Name() string {
	return "Datum Agent"
}

func (agent *ChatAgent) Type() schema.ProviderType {
	return schema.Agent
}

func (agent *ChatAgent) Description() string {
	return "Datum Test Voice Agent"
}

func (agent *ChatAgent) Run(ctx context.Context, input string, session schema.ChatSession) error {
	chat := func(ctx context.Context, state []llms.MessageContent, options graph.Options) ([]llms.MessageContent, error) {
		options.CallbackHandler.HandleNodeStart(ctx, "Chat", state)

		response, _ := agent.LLM.GenerateContent(ctx, state,
			llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
				options.CallbackHandler.HandleNodeStream(ctx, "Chat", chunk)
				return nil
			}),
		)

		state = append(state, llms.TextParts(llms.ChatMessageTypeAI, response.Choices[0].Content))
		options.CallbackHandler.HandleNodeEnd(ctx, "Chat", state)

		return state, nil

	}

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

	callback := NewChatCallback()
	callback.ReadStreamFunc(ctx, agent.streamHandler)

	workflow := graph.NewMessageGraph(graph.WithCallback(callback))

	workflow.AddNode("chat", chat)
	workflow.AddEdge("chat", graph.END)
	workflow.SetEntryPoint("chat")

	chatAgent, err := workflow.Compile()
	if err != nil {
		log.Printf("Failed to create workflow %v", err)
		return err
	}

	initialState := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, "You are a helpful assistant!"),
		llms.TextParts(llms.ChatMessageTypeSystem, "CHAT HISTORY: \n"+buffer),
		llms.TextParts(llms.ChatMessageTypeHuman, input),
	}

	resp, err := chatAgent.Invoke(ctx, initialState)
	if err != nil {
		log.Printf("Failed to invoke workflow %v", err)
		return err
	}

	text := resp[len(resp)-1].Parts[0].(llms.TextContent).Text

	aiMsg := schema.Msg{
		Role:      schema.AIMsg,
		Content:   text,
		Timestamp: time.Now().Unix(),
	}

	err = agent.storage.SaveMsg(session.ID, aiMsg)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func (agent *ChatAgent) Stream(ctx context.Context, callback func(ctx context.Context, msg schema.Msg) error) {
	agent.streamHandler = callback
}
