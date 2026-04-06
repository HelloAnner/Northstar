/**
 * 服务启动初始化测试
 *
 * @author Anner
 * Created on 2026/3/14
 */

package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"northstar/internal/config"
	"northstar/internal/store"
)

func TestNewServerInitializesCleanState(t *testing.T) {
	dataDir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Data.DataDir = dataDir
	cfg.Server.DevMode = false

	s := NewServer(cfg)
	t.Cleanup(func() {
		_ = s.GetStore().Close()
	})

	// 验证 store 能正常加载约束（空列表）
	constraints, err := s.GetStore().ListAdjustmentConstraints(false)
	if err != nil {
		t.Fatalf("list constraints: %v", err)
	}
	if len(constraints) != 0 {
		t.Fatalf("expected 0 default constraints, got %d", len(constraints))
	}

	// 验证约束 API 可用
	httpServer := httptest.NewServer(s.router)
	t.Cleanup(httpServer.Close)

	resp, err := http.Get(httpServer.URL + "/api/v1/constraints")
	if err != nil {
		t.Fatalf("get constraints: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected constraints status: %d", resp.StatusCode)
	}

	var items []store.AdjustmentConstraint
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		t.Fatalf("decode constraints: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected empty constraints, got %d", len(items))
	}

	// 验证自然语言规则 API 可用
	rulesResp, err := http.Get(httpServer.URL + "/api/v1/natural-rules")
	if err != nil {
		t.Fatalf("get natural-rules: %v", err)
	}
	defer rulesResp.Body.Close()
	if rulesResp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected natural-rules status: %d", rulesResp.StatusCode)
	}
}
