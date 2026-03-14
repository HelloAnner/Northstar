/**
 * AI 对话意图解析测试
 *
 * @author Anner
 * Created on 2026/3/14
 */
package llm

import (
	"context"
	"strings"
	"testing"
)

type fakeIntentClient struct {
	result  ChatResult
	err     error
	lastReq ChatRequest
}

func (f *fakeIntentClient) Chat(_ context.Context, req ChatRequest, _ func(string) error) (ChatResult, error) {
	f.lastReq = req
	return f.result, f.err
}

func TestParseIntentReturnsAdjustmentPlan(t *testing.T) {
	client := &fakeIntentClient{
		result: ChatResult{
			Content: "```json\n{\"actions\":[{\"type\":\"set_target\",\"indicatorId\":\"wholesale_month_rate\",\"value\":15}]}\n```",
		},
	}

	plan, err := ParseIntent(client, "把批发当月增速调整到 15%", map[string]float64{
		"wholesale_month_rate": 12,
		"retail_month_rate":    8,
	})
	if err != nil {
		t.Fatalf("parse intent: %v", err)
	}
	if len(plan.Actions) != 1 {
		t.Fatalf("expected 1 action, got %+v", plan.Actions)
	}
	if plan.Actions[0].IndicatorID != "wholesale_month_rate" || plan.Actions[0].Value != 15 {
		t.Fatalf("unexpected action: %+v", plan.Actions[0])
	}
	if len(client.lastReq.Messages) != 1 {
		t.Fatalf("expected a single prompt message, got %+v", client.lastReq.Messages)
	}
	if !strings.Contains(client.lastReq.Messages[0].Content, "批发当月增速调整到 15%") {
		t.Fatalf("expected request message to include user message, got %s", client.lastReq.Messages[0].Content)
	}
	if !strings.Contains(client.lastReq.Messages[0].Content, "wholesale_month_rate") {
		t.Fatalf("expected request message to include indicators, got %s", client.lastReq.Messages[0].Content)
	}
}

func TestParseIntentAllowsEmptyActions(t *testing.T) {
	client := &fakeIntentClient{
		result: ChatResult{
			Content: `{"actions":[]}`,
		},
	}

	plan, err := ParseIntent(client, "当前零售增速为什么偏低？", map[string]float64{
		"retail_month_rate": 8,
	})
	if err != nil {
		t.Fatalf("parse intent: %v", err)
	}
	if plan == nil || len(plan.Actions) != 0 {
		t.Fatalf("expected empty actions, got %+v", plan)
	}
}

func TestParseIntentRejectsInvalidIndicator(t *testing.T) {
	client := &fakeIntentClient{
		result: ChatResult{
			Content: `{"actions":[{"type":"set_target","indicatorId":"bad_indicator","value":15}]}`,
		},
	}

	_, err := ParseIntent(client, "调整指标", map[string]float64{
		"wholesale_month_rate": 12,
	})
	if err == nil || !strings.Contains(err.Error(), "bad_indicator") {
		t.Fatalf("expected invalid indicator error, got %v", err)
	}
}

func TestParseIntentAcceptsSetIndicatorAlias(t *testing.T) {
	client := &fakeIntentClient{
		result: ChatResult{
			Content: `{"actions":[{"type":"set_indicator","indicatorId":"wholesale_month_rate","value":18}]}`,
		},
	}

	plan, err := ParseIntent(client, "把批发当月增速调到 18%", map[string]float64{
		"wholesale_month_rate": 12,
	})
	if err != nil {
		t.Fatalf("parse intent: %v", err)
	}
	if len(plan.Actions) != 1 {
		t.Fatalf("expected 1 action, got %+v", plan.Actions)
	}
	if plan.Actions[0].Type != "set_target" {
		t.Fatalf("expected alias to normalize to set_target, got %+v", plan.Actions[0])
	}
}
