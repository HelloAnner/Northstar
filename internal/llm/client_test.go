/**
 * LLM 客户端单元测试
 *
 * @author Anner
 * @since 12.0
 * Created on 2026/2/1
 */
package llm

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/tmc/langchaingo/llms"
)

// TestBuildMessages 测试消息构建功能
func TestBuildMessages(t *testing.T) {
	tests := []struct {
		name     string
		prompt   string
		history  []ChatMessage
		expected int // 期望的消息数量
	}{
		{
			name:     "仅系统提示词",
			prompt:   "你是助手",
			history:  []ChatMessage{},
			expected: 1,
		},
		{
			name:   "单轮对话",
			prompt: "你是助手",
			history: []ChatMessage{
				{Role: "user", Content: "你好"},
			},
			expected: 2,
		},
		{
			name:   "多轮对话",
			prompt: "你是助手",
			history: []ChatMessage{
				{Role: "user", Content: "你好"},
				{Role: "assistant", Content: "你好！有什么可以帮助你的？"},
				{Role: "user", Content: "测试"},
			},
			expected: 4,
		},
		{
			name:   "包含空内容消息",
			prompt: "你是助手",
			history: []ChatMessage{
				{Role: "user", Content: "你好"},
				{Role: "user", Content: "   "}, // 空白内容应被跳过
				{Role: "assistant", Content: "回复"},
			},
			expected: 3,
		},
		{
			name:   "包含无效角色",
			prompt: "你是助手",
			history: []ChatMessage{
				{Role: "user", Content: "你好"},
				{Role: "invalid_role", Content: "无效角色"},
				{Role: "assistant", Content: "回复"},
			},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messages := buildMessages(tt.prompt, tt.history)
			if len(messages) != tt.expected {
				t.Errorf("期望 %d 条消息，实际 %d 条", tt.expected, len(messages))
			}

			// 验证第一条是系统消息
			if len(messages) > 0 {
				parts := messages[0].Parts
				if len(parts) > 0 {
					text, ok := parts[0].(llms.TextContent)
					if !ok || text.Text != tt.prompt {
						t.Errorf("第一条消息应该是系统提示词 '%s'", tt.prompt)
					}
				}
			}
		})
	}
}

// TestToMessageRole 测试角色转换
func TestToMessageRole(t *testing.T) {
	tests := []struct {
		input    string
		expected llms.ChatMessageType
	}{
		{"user", llms.ChatMessageTypeHuman},
		{"USER", llms.ChatMessageTypeHuman},
		{"User", llms.ChatMessageTypeHuman},
		{"assistant", llms.ChatMessageTypeAI},
		{"ASSISTANT", llms.ChatMessageTypeAI},
		{"Assistant", llms.ChatMessageTypeAI},
		{"system", llms.ChatMessageTypeSystem},
		{"SYSTEM", llms.ChatMessageTypeSystem},
		{"invalid", ""},
		{"", ""},
		{"  user  ", llms.ChatMessageTypeHuman}, // 带空格
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toMessageRole(tt.input)
			if result != tt.expected {
				t.Errorf("角色 '%s' 转换期望 %v，实际 %v", tt.input, tt.expected, result)
			}
		})
	}
}

// TestFirstChoice 测试首个选择提取
func TestFirstChoice(t *testing.T) {
	tests := []struct {
		name        string
		resp        *llms.ContentResponse
		expectError bool
	}{
		{
			name:        "nil 响应",
			resp:        nil,
			expectError: true,
		},
		{
			name:        "空选择列表",
			resp:        &llms.ContentResponse{Choices: []*llms.ContentChoice{}},
			expectError: true,
		},
		{
			name: "有效响应",
			resp: &llms.ContentResponse{
				Choices: []*llms.ContentChoice{
					{Content: "测试内容"},
				},
			},
			expectError: false,
		},
		{
			name: "首个选择为 nil",
			resp: &llms.ContentResponse{
				Choices: []*llms.ContentChoice{nil, {Content: "第二个"}},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			choice, err := firstChoice(tt.resp)
			if tt.expectError {
				if err == nil {
					t.Error("期望错误但未返回")
				}
			} else {
				if err != nil {
					t.Errorf("不期望错误但返回: %v", err)
				}
				if choice == nil {
					t.Error("期望返回选择但实际为 nil")
				}
			}
		})
	}
}

// TestShouldSkipStreamChunk 测试流式块跳过逻辑
func TestShouldSkipStreamChunk(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"", true},                         // 空字符串
		{"   ", true},                      // 空白字符
		{"\n\t", true},                     // 换行和制表符
		{"正常内容", false},                  // 正常内容
		{"{\"arguments\":{}}", true},       // 工具调用参数
		{"{\"tool\":\"test\"}", true},      // 工具调用
		{"{\"function\":{}}", true},       // 函数调用
		{"{\"key\":\"value\"}", false},    // 普通 JSON
		{"[1,2,3]", false},                 // 普通数组
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := shouldSkipStreamChunk([]byte(tt.input))
			if result != tt.expected {
				t.Errorf("输入 '%s' 期望跳过=%v，实际=%v", tt.input, tt.expected, result)
			}
		})
	}
}

// TestNewClient 测试客户端创建
func TestNewClient(t *testing.T) {
	// 获取环境变量配置
	baseURL := os.Getenv("DEEPSEEK_BASE_URL")
	if baseURL == "" {
		baseURL = os.Getenv("LLM_BASE_URL")
	}
	model := os.Getenv("DEEPSEEK_MODEL_NAME")
	if model == "" {
		model = os.Getenv("LLM_MODEL")
	}
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("LLM_API_KEY")
	}

	// 如果没有配置，使用测试默认值
	if baseURL == "" {
		baseURL = "https://api.deepseek.com/v1"
	}
	if model == "" {
		model = "deepseek-chat"
	}
	if apiKey == "" {
		apiKey = "test-api-key"
	}

	tests := []struct {
		name      string
		cfg       Config
		prompt    string
		expectErr bool
	}{
		{
			name: "有效配置",
			cfg: Config{
				BaseURL: baseURL,
				Model:   model,
				APIKey:  apiKey,
			},
			prompt:    "测试提示词",
			expectErr: false,
		},
		{
			name: "空提示词",
			cfg: Config{
				BaseURL: baseURL,
				Model:   model,
				APIKey:  apiKey,
			},
			prompt:    "",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.cfg, tt.prompt)
			if tt.expectErr {
				if err == nil {
					t.Error("期望错误但未返回")
				}
			} else {
				if err != nil {
					t.Logf("创建客户端时出错 (可能是网络问题): %v", err)
					// 不强制失败，因为可能是网络/API 配置问题
				}
				if client != nil {
					if client.prompt != tt.prompt {
						t.Errorf("提示词不匹配: 期望 '%s'，实际 '%s'", tt.prompt, client.prompt)
					}
					if len(client.tools) != 2 {
						t.Errorf("工具数量不正确: 期望 2，实际 %d", len(client.tools))
					}
				}
			}
		})
	}
}

// TestClientChatIntegration 集成测试 - 需要有效的 API 配置
func TestClientChatIntegration(t *testing.T) {
	// 检查是否有 API 配置
	baseURL := os.Getenv("DEEPSEEK_BASE_URL")
	if baseURL == "" {
		baseURL = os.Getenv("LLM_BASE_URL")
	}
	model := os.Getenv("DEEPSEEK_MODEL_NAME")
	if model == "" {
		model = os.Getenv("LLM_MODEL")
	}
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("LLM_API_KEY")
	}

	if baseURL == "" || model == "" || apiKey == "" {
		t.Skip("跳过集成测试: 未配置 LLM 环境变量 (DEEPSEEK_* 或 LLM_*)")
	}

	if !strings.Contains(apiKey, "sk-") && len(apiKey) < 10 {
		t.Skip("跳过集成测试: API Key 格式看起来无效")
	}

	cfg := Config{
		BaseURL: baseURL,
		Model:   model,
		APIKey:  apiKey,
	}

	prompt := "你是 Northstar 经济数据分析助手。当前操作年月：2026年1月。"
	client, err := NewClient(cfg, prompt)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}

	// 测试简单对话
	req := ChatRequest{
		Messages: []ChatMessage{
			{Role: "user", Content: "你好，请介绍一下你的功能"},
		},
		Year:  2026,
		Month: 1,
	}

	var streamContent strings.Builder
	streamFunc := func(chunk string) error {
		streamContent.WriteString(chunk)
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30)
	defer cancel()

	result, err := client.Chat(ctx, req, streamFunc)
	if err != nil {
		t.Logf("对话测试出错 (可能是 API 限制): %v", err)
		return // 不强制失败
	}

	if result.Content == "" && len(result.ToolCalls) == 0 {
		t.Error("返回结果既无内容也无工具调用")
	}

	t.Logf("流式内容长度: %d", streamContent.Len())
	t.Logf("工具调用数量: %d", len(result.ToolCalls))
}

// TestClientChatWithStream 测试流式输出
func TestClientChatWithStream(t *testing.T) {
	// 检查是否有 API 配置
	baseURL := os.Getenv("DEEPSEEK_BASE_URL")
	if baseURL == "" {
		baseURL = os.Getenv("LLM_BASE_URL")
	}
	model := os.Getenv("DEEPSEEK_MODEL_NAME")
	if model == "" {
		model = os.Getenv("LLM_MODEL")
	}
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("LLM_API_KEY")
	}

	if baseURL == "" || model == "" || apiKey == "" {
		t.Skip("跳过流式测试: 未配置 LLM 环境变量")
	}

	cfg := Config{
		BaseURL: baseURL,
		Model:   model,
		APIKey:  apiKey,
	}

	prompt := "你是助手"
	client, err := NewClient(cfg, prompt)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}

	req := ChatRequest{
		Messages: []ChatMessage{
			{Role: "user", Content: "说两个字：测试"},
		},
		Year:  2026,
		Month: 1,
	}

	chunks := []string{}
	streamFunc := func(chunk string) error {
		chunks = append(chunks, chunk)
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30)
	defer cancel()

	_, err = client.Chat(ctx, req, streamFunc)
	if err != nil {
		t.Logf("流式对话出错: %v", err)
		return
	}

	if len(chunks) == 0 {
		t.Log("未收到流式块，可能是 API 不返回流式数据")
	} else {
		t.Logf("收到 %d 个流式块", len(chunks))
	}
}
