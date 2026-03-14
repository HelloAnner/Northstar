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
	"os"
	"path/filepath"
	"testing"

	"northstar/internal/config"
	"northstar/internal/rules"
)

func TestNewServerInitializesRuleManagement(t *testing.T) {
	dataDir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Data.DataDir = dataDir
	cfg.Server.DevMode = false

	s := NewServer(cfg)
	t.Cleanup(func() {
		_ = s.GetStore().Close()
	})

	rulesPath := filepath.Join(dataDir, "config", "rules.md")
	raw, err := os.ReadFile(rulesPath)
	if err != nil {
		t.Fatalf("read rules.md: %v", err)
	}
	if len(raw) == 0 {
		t.Fatal("rules.md should not be empty")
	}

	roleJSONPath := filepath.Join(dataDir, "config", "role.json")
	roleRaw, err := os.ReadFile(roleJSONPath)
	if err != nil {
		t.Fatalf("read role.json: %v", err)
	}
	if len(roleRaw) == 0 {
		t.Fatal("role.json should not be empty")
	}
	loadedRules, err := rules.Load(roleJSONPath)
	if err != nil {
		t.Fatalf("load role.json: %v", err)
	}
	if loadedRules == nil {
		t.Fatal("loaded rules should not be nil")
	}
	if total := len(loadedRules.Clamps) + len(loadedRules.Filters) + len(loadedRules.Compensates); total != 17 {
		t.Fatalf("expected 17 default role.json rules, got %d", total)
	}

	assertConfigValue(t, s, "rules_convert_status", "idle")
	assertConfigValue(t, s, "rules_convert_at", "")
	assertConfigValue(t, s, "rules_convert_error", "")
	assertConfigValue(t, s, "llm_user_prompt", "")

	httpServer := httptest.NewServer(s.router)
	t.Cleanup(httpServer.Close)

	resp, err := http.Get(httpServer.URL + "/api/v1/rules")
	if err != nil {
		t.Fatalf("get rules: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected rules status: %d", resp.StatusCode)
	}

	var items []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		t.Fatalf("decode rules: %v", err)
	}
	// 默认内置 3 条值类非负约束 + 13 条增速范围约束 + 1 条联动约束
	if len(items) != 17 {
		t.Fatalf("expected 17 default rules, got %d", len(items))
	}
}

func assertConfigValue(t *testing.T, s *Server, key string, expected string) {
	t.Helper()

	value, err := s.GetStore().GetConfig(key)
	if err != nil {
		t.Fatalf("get config %s: %v", key, err)
	}
	if value != expected {
		t.Fatalf("unexpected config %s=%q expected=%q", key, value, expected)
	}
}
