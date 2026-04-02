package nodes

import (
	"context"

	"github.com/AstroSynapseLab/mar-mar-ai/sdk/glg/graph"
	"github.com/tmc/langchaingo/llms"
)

func AgentNode(llm llms.Model, functions []llms.Tool) graph.NodeFunction {
	return func(ctx context.Context, state []llms.MessageContent, options graph.Options) ([]llms.MessageContent, error) {

		response, err := llm.GenerateContent(ctx, state, llms.WithTools(functions))
		if err != nil {
			return state, err
		}

		msg := llms.TextParts(llms.ChatMessageTypeAI, response.Choices[0].Content)

		if len(response.Choices[0].ToolCalls) > 0 {
			for _, toolCall := range response.Choices[0].ToolCalls {
				msg.Parts = append(msg.Parts, toolCall)
			}
		}

		state = append(state, msg)
		return state, nil
	}
}
