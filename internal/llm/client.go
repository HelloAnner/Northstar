/**
 * LLM 客户端封装
 *
 * @author Anner
 * @since 12.0
 * Created on 2026/2/1
 */
package llm

import (
	"context"
	"fmt"
	"strings"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// Client 大模型客户端
type Client struct {
	model  llms.Model
	prompt string
	tools  []llms.Tool
}

// NewClient 创建客户端
func NewClient(cfg Config, prompt string) (*Client, error) {
	return newClient(cfg, prompt, Tools())
}

// NewTextClient 创建不带工具的纯文本客户端。
func NewTextClient(cfg Config, prompt string) (*Client, error) {
	return newClient(cfg, prompt, nil)
}

func newClient(cfg Config, prompt string, tools []llms.Tool) (*Client, error) {
	llm, err := openai.New(
		openai.WithToken(cfg.APIKey),
		openai.WithModel(cfg.Model),
		openai.WithBaseURL(cfg.BaseURL),
	)
	if err != nil {
		return nil, err
	}
	return &Client{
		model:  llm,
		prompt: prompt,
		tools:  tools,
	}, nil
}

// Chat 执行聊天并返回工具调用结果
func (c *Client) Chat(ctx context.Context, req ChatRequest, stream func(string) error) (ChatResult, error) {
	messages := buildMessages(c.prompt, req.Messages)
	options := make([]llms.CallOption, 0, 2)
	if len(c.tools) > 0 {
		options = append(options, llms.WithTools(c.tools))
	}
	if stream != nil {
		options = append(options, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
			if shouldSkipStreamChunk(chunk) {
				return nil
			}
			return stream(string(chunk))
		}))
	}
	resp, err := c.model.GenerateContent(ctx, messages, options...)
	if err != nil {
		return ChatResult{}, err
	}
	choice, err := firstChoice(resp)
	if err != nil {
		return ChatResult{}, err
	}
	return ChatResult{Content: choice.Content, ToolCalls: choice.ToolCalls}, nil
}

func buildMessages(prompt string, history []ChatMessage) []llms.MessageContent {
	messages := []llms.MessageContent{llms.TextParts(llms.ChatMessageTypeSystem, prompt)}
	for _, msg := range history {
		role := toMessageRole(msg.Role)
		if role == "" || strings.TrimSpace(msg.Content) == "" {
			continue
		}
		messages = append(messages, llms.TextParts(role, msg.Content))
	}
	return messages
}

func toMessageRole(role string) llms.ChatMessageType {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "assistant":
		return llms.ChatMessageTypeAI
	case "user":
		return llms.ChatMessageTypeHuman
	case "system":
		return llms.ChatMessageTypeSystem
	default:
		return ""
	}
}

func firstChoice(resp *llms.ContentResponse) (*llms.ContentChoice, error) {
	if resp == nil || len(resp.Choices) == 0 || resp.Choices[0] == nil {
		return nil, fmt.Errorf("模型未返回内容")
	}
	return resp.Choices[0], nil
}

func shouldSkipStreamChunk(chunk []byte) bool {
	trimmed := strings.TrimSpace(string(chunk))
	if trimmed == "" {
		return true
	}
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		if strings.Contains(trimmed, "\"arguments\"") || strings.Contains(trimmed, "\"tool\"") || strings.Contains(trimmed, "\"function\"") {
			return true
		}
	}
	return false
}
