package agents

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/lokutor-ai/lokutor-orchestrator/pkg/orchestrator"
	"github.com/struki84/datum/clipt/storage"
	"github.com/struki84/datum/clipt/tui/schema"
	"github.com/struki84/datum/config"
	"github.com/struki84/datum/sdk/glg/graph"
	"github.com/struki84/datum/sdk/glg/nodes"
	"github.com/struki84/datum/tools/library"
	scraper "github.com/struki84/datum/tools/scrapper"
	"github.com/struki84/datum/tools/search"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/tools"
)

var (
	primerMsg = `
You are Datum, an intelligent voice assistant powered by a knowledge base.
Your role is to answer user questions accurately using available tools and retrieved context.

When answering, prefer information from the knowledge base over general knowledge.
If the knowledge base doesn't have what you need, fall back to web search.
If neither source has the answer, say so honestly rather than guessing.

Keep responses concise and conversational — they will be spoken aloud, so avoid
markdown formatting, bullet lists, or code blocks in your final answer.
You may call multiple tools if needed before giving your final response.`

//	primerMsg = `
//
// You are Datum, an intelligent voice assistant powered by a knowledge base.
// Your role is to answer user questions accurately using available tools and retrieved context.
//
// You follow a structured reasoning process:
//
// 1. THINK - Analyze the user's question and determine what information you need.
// 2. ACT - Use available tools (web search, knowledge base retrieval) to gather relevant information.
// 3. OBSERVE - Review the results from your tools.
// 4. RESPOND - Synthesize a clear, concise answer based on the gathered context.
//
// Guidelines:
// - Always prefer information from the knowledge base over general knowledge.
// - If the knowledge base doesn't have relevant information, use web search.
// - If neither source has the answer, say so honestly rather than guessing.
// - Keep responses concise and conversational since they will be spoken aloud.
// - Avoid markdown formatting, bullet lists, or code blocks in your final response as the output is converted to speech.
// - When using tools, you may call multiple tools if needed before giving your final answer.
// - Cite your reasoning briefly when the source matters.`
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

func (handler *ChatCallback) HandleNodeStart(ctx context.Context, node string, state []llms.MessageContent) {
	if node == "Chat" {
		msg := schema.Msg{
			Stream:    true,
			Role:      schema.AIMsg,
			Content:   "",
			Timestamp: time.Now().Unix(),
		}

		handler.stream(ctx, msg)
	}

	if node == "ToolNode" {
		lastMsg := state[len(state)-1]

		var toolName string
		for _, part := range lastMsg.Parts {
			if toolCall, ok := part.(llms.ToolCall); ok {
				toolName = toolCall.FunctionCall.Name
			}
		}

		content := fmt.Sprintf("Executing %s tool...", toolName)
		msg := schema.Msg{
			Stream:    false,
			Role:      schema.SysMsg,
			Content:   content,
			Timestamp: time.Now().Unix(),
		}

		handler.stream(ctx, msg)
	}
}

func (handler *ChatCallback) HandleNodeEnd(ctx context.Context, node string, finalState []llms.MessageContent) {

}

func (handler *ChatCallback) HandleNodeStream(ctx context.Context, node string, chunk []byte) {
	if node == "Chat" {
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
	LLM            *openai.LLM
	streamHandler  func(ctx context.Context, msg schema.Msg) error
	currentModel   string
	storage        storage.SQLite
	functions      []llms.Tool
	tools          []tools.Tool
	cycle          int
	maxCycles      int
	config         *config.Config
	currentSession schema.ChatSession
}

func NewChatAgent(model string, storage storage.SQLite) *ChatAgent {
	functions := []llms.Tool{
		{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name:        "WebSearch",
				Description: "Performs web search using Google and DuckDuckGo, will resolve to DuckDuckGo if Google is unavailable.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"query": map[string]any{
							"type":        "string",
							"description": "The search query",
						},
					},
				},
			},
		},
		{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name:        "WebBrowser",
				Description: "Will read and summarize the contents of a web page and return it as string output, input should be a full website url",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"url": map[string]any{
							"type":        "string",
							"description": "Url of the web page",
						},
					},
				},
			},
		},
		{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name:        "ListFiles",
				Description: "Lists all files you can read and search.",
				Parameters: map[string]any{
					"type":       "object",
					"properties": map[string]any{},
				},
			},
		},
		{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name:        "ReadFiles",
				Description: "Lists all files you can read and search.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"file": map[string]any{
							"type":        "string",
							"description": "Name of the file for reading.",
						},
						"query": map[string]any{
							"type":        "string",
							"description": "The search query",
						},
					},
				},
			},
		},
		{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name:        "SearchFiles",
				Description: "Searches trough the files for relevant information.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"query": map[string]any{
							"type":        "string",
							"description": "The search query",
						},
					},
				},
			},
		},
	}

	cnf := config.New()

	llm, err := openai.New(
		openai.WithModel(model),
		openai.WithBaseURL("https://openrouter.ai/api/v1"),
		openai.WithToken(cnf.OpenrouterAPIKey),
	)

	if err != nil {
		fmt.Println("Can't create model:", err)
		return nil
	}

	filesDir := "./data"

	// Init the tools that will be used
	webSearch, _ := search.New(llm, cnf.SerpAPIKey)
	webBrowser, _ := scraper.New()
	listFiles, _ := library.NewFileListTool(filesDir)
	searchFiles, _ := library.NewFileSearchTool()
	readFiles, _ := library.NewReadFileTool(llm, filesDir)

	return &ChatAgent{
		LLM:          llm,
		currentModel: model,
		storage:      storage,
		functions:    functions,
		cycle:        1,
		maxCycles:    6,
		config:       cnf,

		tools: []tools.Tool{
			webSearch,
			webBrowser,
			listFiles,
			searchFiles,
			readFiles,
		},
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

func (agent *ChatAgent) SetSession(session schema.ChatSession) {
	agent.currentSession = session
}

func (agent *ChatAgent) Stream(ctx context.Context, callback func(ctx context.Context, msg schema.Msg) error) {
	agent.streamHandler = callback
}

func (agent *ChatAgent) Run(ctx context.Context, input string, session schema.ChatSession) (string, error) {
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

	useTool := func(ctx context.Context, state []llms.MessageContent, options graph.Options) string {
		if agent.cycle >= agent.maxCycles {
			agent.cycle = 0
			return "chat"
		}

		lastMsg := state[len(state)-1]

		for _, part := range lastMsg.Parts {
			if _, ok := part.(llms.ToolCall); ok {
				agent.cycle++
				return "tool"
			}
		}

		return "chat"
	}

	buffer, err := agent.storage.LoadMsgs(session.ID)
	if err != nil {
		log.Println(err)
		return "", err
	}

	userMsg := schema.Msg{
		Role:      schema.UserMsg,
		Content:   input,
		Timestamp: time.Now().Unix(),
	}

	err = agent.storage.SaveMsg(session.ID, userMsg)
	if err != nil {
		log.Println(err)
		return "", err
	}

	callback := NewChatCallback()
	callback.ReadStreamFunc(ctx, agent.streamHandler)

	workflow := graph.NewMessageGraph(graph.WithCallback(callback))

	workflow.AddNode("agent", nodes.AgentNode(agent.LLM, agent.functions))
	workflow.AddNode("tool", nodes.ToolNode(agent.tools))
	workflow.AddNode("chat", chat)

	workflow.SetEntryPoint("agent")
	workflow.AddConditionalEdge("agent", useTool)
	workflow.AddEdge("tool", "agent")
	workflow.AddEdge("chat", graph.END)

	initialState := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, primerMsg),
		llms.TextParts(llms.ChatMessageTypeSystem, "CHAT HISTORY: \n"+buffer),
		llms.TextParts(llms.ChatMessageTypeHuman, input),
	}

	chatAgent, err := workflow.Compile()
	if err != nil {
		log.Printf("Failed to create workflow %v", err)
		return "", err
	}

	resp, err := chatAgent.Invoke(ctx, initialState)
	if err != nil {
		log.Printf("Failed to invoke workflow %v", err)
		return "", err
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
		return "", err
	}

	return text, nil
}

func (agent *ChatAgent) Complete(ctx context.Context, messages []orchestrator.Message, tools []orchestrator.Tool) (string, error) {
	var userInput string
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			userInput = messages[i].Content
			break
		}
	}

	var fallbackResponses = []string{
		"I found the information, but I'm having trouble putting it into words. Could you try asking again?",
		"Something got lost on my end. I processed your request but couldn't form a response — give me another shot.",
		"My thoughts seem to have wandered off. Want to try that again?",
	}

	// runCtx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	// defer cancel()

	response, err := agent.Run(ctx, userInput, agent.currentSession)
	if err != nil {
		log.Printf("Error running chat agent Complete(): %v", err)
		return fallbackResponses[rand.Intn(len(fallbackResponses))], nil
	}

	return response, nil
}
