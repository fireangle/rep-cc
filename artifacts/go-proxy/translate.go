package main

import (
	"encoding/json"
	"strings"
	"time"
)

// --- OpenAI → Anthropic ---

func openAIToolsToAnthropic(tools []OpenAITool) []AnthropicTool {
	out := make([]AnthropicTool, len(tools))
	for i, t := range tools {
		out[i] = AnthropicTool{
			Name:        t.Function.Name,
			Description: t.Function.Description,
			InputSchema: t.Function.Parameters,
		}
	}
	return out
}

func openAIToolChoiceToAnthropic(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}

	var s string
	if json.Unmarshal(raw, &s) == nil {
		switch s {
		case "auto":
			return json.RawMessage(`{"type":"auto"}`)
		case "none":
			return json.RawMessage(`{"type":"auto"}`)
		case "required":
			return json.RawMessage(`{"type":"any"}`)
		}
		return nil
	}

	var obj struct {
		Function struct {
			Name string `json:"name"`
		} `json:"function"`
	}
	if json.Unmarshal(raw, &obj) == nil && obj.Function.Name != "" {
		b, _ := json.Marshal(map[string]string{"type": "tool", "name": obj.Function.Name})
		return b
	}
	return nil
}

func openAIMessagesToAnthropic(msgs []OpenAIMessage) (system string, anthropicMsgs []AnthropicMessage) {
	for _, msg := range msgs {
		switch msg.Role {
		case "system":
			var s string
			if json.Unmarshal(msg.Content, &s) == nil {
				system = s
			}

		case "tool":
			block := AnthropicContentBlock{
				Type:      "tool_result",
				ToolUseID: msg.ToolCallID,
			}
			var s string
			if json.Unmarshal(msg.Content, &s) == nil {
				block.Content = json.RawMessage(`"` + strings.ReplaceAll(s, `"`, `\"`) + `"`)
			}

			// Merge into last user message if possible
			if len(anthropicMsgs) > 0 {
				last := &anthropicMsgs[len(anthropicMsgs)-1]
				if last.Role == "user" {
					var blocks []AnthropicContentBlock
					if json.Unmarshal(last.Content, &blocks) == nil {
						blocks = append(blocks, block)
						last.Content, _ = json.Marshal(blocks)
						continue
					}
				}
			}
			content, _ := json.Marshal([]AnthropicContentBlock{block})
			anthropicMsgs = append(anthropicMsgs, AnthropicMessage{
				Role:    "user",
				Content: content,
			})

		case "assistant":
			var blocks []AnthropicContentBlock
			var s string
			if json.Unmarshal(msg.Content, &s) == nil && s != "" {
				blocks = append(blocks, AnthropicContentBlock{Type: "text", Text: s})
			}
			for _, tc := range msg.ToolCalls {
				var input json.RawMessage
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &input); err != nil {
					input = json.RawMessage(`{}`)
				}
				blocks = append(blocks, AnthropicContentBlock{
					Type:  "tool_use",
					ID:    tc.ID,
					Name:  tc.Function.Name,
					Input: input,
				})
			}
			content, _ := json.Marshal(blocks)
			anthropicMsgs = append(anthropicMsgs, AnthropicMessage{
				Role:    "assistant",
				Content: content,
			})

		case "user":
			anthropicMsgs = append(anthropicMsgs, AnthropicMessage{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}
	}
	return
}

func buildAnthropicRequest(req *OpenAIChatRequest) *AnthropicRequest {
	system, msgs := openAIMessagesToAnthropic(req.Messages)

	maxTokens := 8192
	if req.MaxTokens != nil {
		maxTokens = *req.MaxTokens
	}

	ar := &AnthropicRequest{
		Model:     req.Model,
		MaxTokens: maxTokens,
		Messages:  msgs,
	}

	if system != "" {
		ar.System = json.RawMessage(`"` + strings.ReplaceAll(system, `"`, `\"`) + `"`)
	}
	if len(req.Tools) > 0 {
		tools := openAIToolsToAnthropic(req.Tools)
		ar.Tools = tools
	}
	if len(req.ToolChoice) > 0 {
		ar.ToolChoice = openAIToolChoiceToAnthropic(req.ToolChoice)
	}
	if req.Temperature != nil {
		ar.Temperature = req.Temperature
	}
	if req.TopP != nil {
		ar.TopP = req.TopP
	}

	return ar
}

// --- Anthropic → OpenAI ---

func anthropicResponseToOpenAI(resp *AnthropicResponse) *OpenAIChatCompletion {
	var text string
	var toolCalls []OpenAIToolCall

	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			text += block.Text
		case "tool_use":
			args, _ := json.Marshal(block.Input)
			toolCalls = append(toolCalls, OpenAIToolCall{
				ID:   block.ID,
				Type: "function",
				Function: OpenAIToolCallFunc{
					Name:      block.Name,
					Arguments: string(args),
				},
			})
		}
	}

	finishReason := "stop"
	if resp.StopReason != nil && *resp.StopReason == "tool_use" {
		finishReason = "tool_calls"
	}

	var contentPtr *string
	if text != "" {
		contentPtr = &text
	}

	return &OpenAIChatCompletion{
		ID:      resp.ID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   resp.Model,
		Choices: []OpenAIChoice{
			{
				Index: 0,
				Message: &OpenAIRespMsg{
					Role:      "assistant",
					Content:   contentPtr,
					ToolCalls: toolCalls,
				},
				FinishReason: &finishReason,
			},
		},
		Usage: &OpenAIUsage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
	}
}
