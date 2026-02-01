/**
 * LLM 交互类型定义
 *
 * @author Anner
 * @since 12.0
 * Created on 2026/2/1
 */
package llm

import (
	"encoding/json"
	"fmt"

	"github.com/tmc/langchaingo/llms"
)

// ChatMessage 会话消息
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest 聊天请求
type ChatRequest struct {
	Messages []ChatMessage
	Year     int
	Month    int
}

// ChatResult 聊天结果
type ChatResult struct {
	Content   string
	ToolCalls []llms.ToolCall
}

// CompanyUpdate 企业更新指令
type CompanyUpdate struct {
	ID    string                 `json:"id"`
	Patch map[string]interface{} `json:"patch"`
}

type indicatorTargetsArgs struct {
	Targets map[string]float64 `json:"targets"`
}

type updateCompaniesArgs struct {
	Updates []CompanyUpdate `json:"updates"`
}

// ParsedToolCalls 解析后的工具调用
type ParsedToolCalls struct {
	IndicatorTargets map[string]float64
	CompanyUpdates   []CompanyUpdate
}

// ParseToolCalls 解析工具调用参数
func ParseToolCalls(calls []llms.ToolCall) (ParsedToolCalls, error) {
	out := ParsedToolCalls{IndicatorTargets: map[string]float64{}}
	for _, call := range calls {
		if err := applyToolCall(&out, call); err != nil {
			return out, err
		}
	}
	return out, nil
}

func applyToolCall(out *ParsedToolCalls, call llms.ToolCall) error {
	if call.FunctionCall == nil {
		return nil
	}
	switch call.FunctionCall.Name {
	case ToolSetIndicatorTargets:
		return mergeIndicatorTargets(out, call.FunctionCall.Arguments)
	case ToolUpdateCompanies:
		return appendCompanyUpdates(out, call.FunctionCall.Arguments)
	default:
		return nil
	}
}

func mergeIndicatorTargets(out *ParsedToolCalls, raw string) error {
	var args indicatorTargetsArgs
	if err := json.Unmarshal([]byte(raw), &args); err != nil {
		return fmt.Errorf("解析指标目标失败: %w", err)
	}
	if args.Targets == nil {
		return nil
	}
	for k, v := range args.Targets {
		out.IndicatorTargets[k] = v
	}
	return nil
}

func appendCompanyUpdates(out *ParsedToolCalls, raw string) error {
	var args updateCompaniesArgs
	if err := json.Unmarshal([]byte(raw), &args); err != nil {
		return fmt.Errorf("解析企业更新失败: %w", err)
	}
	if len(args.Updates) == 0 {
		return nil
	}
	out.CompanyUpdates = append(out.CompanyUpdates, args.Updates...)
	return nil
}
