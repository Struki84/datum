package search

import (
	"context"
	"encoding/json"
	"log"

	"github.com/AstroSynapseLab/mar-mar-ai/internal/ai/tools/search/google"
	"github.com/AstroSynapseLab/mar-mar-ai/sdk/glg/graph"
	"github.com/AstroSynapseLab/mar-mar-ai/sdk/glg/nodes"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/tools"
	"github.com/tmc/langchaingo/tools/duckduckgo"
)

var (
	searchPrimer = `You are an agent that has access to a DuckDuckGo and Google search engine.
	Please provide the user with the information they are looking for by using the search tools provided.`
)

var _ tools.Tool = &WebSearchTool{}

type WebSearchTool struct {
	llm        llms.Model
	functions  []llms.Tool
	serpApiKey string
}

func New(llm llms.Model, serpApiKey string) (*WebSearchTool, error) {
	functions := []llms.Tool{
		{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name:        "secondarySearch",
				Description: "Performs DuckDuckGo web search, use this search tool only if primary search fails.",
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
				Name:        "primarySearch",
				Description: "Performs google web search via serpapi. Use this search tool as primary tool.",
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

	return &WebSearchTool{
		llm:       llm,
		functions: functions,
	}, nil
}

func (search *WebSearchTool) Name() string {
	return "WebSearch"
}

func (search *WebSearchTool) Description() string {
	return "Performs web search using Google and DuckDuckGo, will resolve to DuckDuckGo if Google is unavailable."
}

func (search *WebSearchTool) Call(ctx context.Context, input string) (string, error) {
	log.Println("Performing web search with input: ", input)

	webSearch := func(ctx context.Context, state []llms.MessageContent, options graph.Options) ([]llms.MessageContent, error) {
		lastMsg := state[len(state)-1]

		for _, part := range lastMsg.Parts {
			if toolCall, ok := part.(llms.ToolCall); ok {
				var args struct {
					Query string `json:"query"`
				}

				if err := json.Unmarshal([]byte(toolCall.FunctionCall.Arguments), &args); err != nil {
					return state, err
				}

				var toolResponse string
				if toolCall.FunctionCall.Name == "primarySearch" {

					google, err := google.New(search.serpApiKey, 10)
					if err != nil {
						log.Printf("search error: %v", err)
						return state, err
					}

					toolResponse, err = google.Call(ctx, args.Query)
					if err != nil {
						log.Printf("search error: %v", err)
						return state, err
					}
				}

				if toolCall.FunctionCall.Name == "secondarySearch" {
					search, err := duckduckgo.New(10, duckduckgo.DefaultUserAgent)
					if err != nil {
						log.Printf("search error: %v", err)
						return state, err
					}

					toolResponse, err = search.Call(ctx, args.Query)
					if err != nil {
						log.Printf("search error: %v", err)
						return state, err
					}
				}

				msg := llms.MessageContent{
					Role: llms.ChatMessageTypeTool,
					Parts: []llms.ContentPart{
						llms.ToolCallResponse{
							ToolCallID: toolCall.ID,
							Name:       toolCall.FunctionCall.Name,
							Content:    toolResponse,
						},
					},
				}

				state = append(state, msg)
			}
		}

		return state, nil
	}

	shouldSearch := func(ctx context.Context, state []llms.MessageContent, options graph.Options) string {
		lastMsg := state[len(state)-1]
		for _, part := range lastMsg.Parts {
			if _, ok := part.(llms.ToolCall); ok {
				return "search"
			}
		}

		return graph.END
	}

	workflow := graph.NewMessageGraph()

	workflow.AddNode("agent", nodes.AgentNode(search.llm, search.functions))
	workflow.AddNode("search", webSearch)

	workflow.SetEntryPoint("agent")
	workflow.AddConditionalEdge("agent", shouldSearch)
	workflow.AddEdge("search", "agent")

	graph, err := workflow.Compile()
	if err != nil {
		log.Printf("error: %v", err)
		return "", err
	}

	initialState := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, searchPrimer),
		llms.TextParts(llms.ChatMessageTypeHuman, input),
	}

	response, err := graph.Invoke(ctx, initialState)
	if err != nil {
		return "", err
	}

	lastMsg := response[len(response)-1].Parts[0].(llms.TextContent).Text
	return lastMsg, nil
}
