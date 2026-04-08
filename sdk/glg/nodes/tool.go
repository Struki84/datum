package nodes

import (
	"context"
	"log"

	"github.com/struki84/datum/sdk/glg/graph"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/tools"
)

func ToolNode(nodeTools []tools.Tool) graph.NodeFunction {
	return func(ctx context.Context, state []llms.MessageContent, options graph.Options) ([]llms.MessageContent, error) {

		if options.CallbackHandler != nil {
			options.CallbackHandler.HandleNodeStart(ctx, "ToolNode", state)
		}

		lastMsg := state[len(state)-1]

		for _, part := range lastMsg.Parts {
			if toolCall, ok := part.(llms.ToolCall); ok {
				var toolNotFound bool
				var toolResponse string
				var err error

				toolNotFound = true
				for _, tool := range nodeTools {
					log.Printf("trying to execute tool node for tool name %s, and function call %s", tool.Name(), toolCall.FunctionCall.Name)
					if tool.Name() == toolCall.FunctionCall.Name {
						toolNotFound = false
						toolResponse, err = tool.Call(ctx, toolCall.FunctionCall.Arguments)
						if err != nil {
							toolResponse = "Error calling tool: " + err.Error()
							return state, err
						}
					}

					if toolNotFound {
						toolResponse = "Requested tool was not found. The reason might be that the tool is not enabled or properly configured."
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

		if options.CallbackHandler != nil {
			options.CallbackHandler.HandleNodeEnd(ctx, "ToolNode", state)
		}
		return state, nil
	}
}
