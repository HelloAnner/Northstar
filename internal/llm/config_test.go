/**
 * LLM 配置单元测试
 *
 * @author Anner
 * @since 12.0
 * Created on 2026/2/1
 */
package llm

import (
	"testing"
)

// TestConfigValidate 测试配置验证
func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "完整配置",
			cfg: Config{
				BaseURL: "https://api.deepseek.com/v1",
				Model:   "deepseek-chat",
				APIKey:  "sk-test123",
			},
			wantErr: false,
		},
		{
			name: "空 BaseURL",
			cfg: Config{
				BaseURL: "",
				Model:   "deepseek-chat",
				APIKey:  "sk-test123",
			},
			wantErr: true,
			errMsg:  "llm_base_url 不能为空",
		},
		{
			name: "空 Model",
			cfg: Config{
				BaseURL: "https://api.deepseek.com/v1",
				Model:   "",
				APIKey:  "sk-test123",
			},
			wantErr: true,
			errMsg:  "llm_model 不能为空",
		},
		{
			name: "空 APIKey",
			cfg: Config{
				BaseURL: "https://api.deepseek.com/v1",
				Model:   "deepseek-chat",
				APIKey:  "",
			},
			wantErr: true,
			errMsg:  "llm_api_key 不能为空",
		},
		{
			name: "全空配置",
			cfg: Config{
				BaseURL: "",
				Model:   "",
				APIKey:  "",
			},
			wantErr: true,
			errMsg:  "llm_base_url",
		},
		{
			name: "BaseURL 带空格",
			cfg: Config{
				BaseURL: "  https://api.deepseek.com/v1  ",
				Model:   "deepseek-chat",
				APIKey:  "sk-test123",
			},
			wantErr: false, // 验证前会被 trim
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				if err == nil {
					t.Error("期望错误但未返回")
				} else if tt.errMsg != "" {
					if err.Error() != tt.errMsg && !containsStr(err.Error(), tt.errMsg) {
						t.Errorf("错误消息期望包含 '%s'，实际 '%s'", tt.errMsg, err.Error())
					}
				}
			} else {
				if err != nil {
					t.Errorf("不期望错误但返回: %v", err)
				}
			}
		})
	}
}

// TestConfigStruct 测试配置结构
func TestConfigStruct(t *testing.T) {
	cfg := Config{
		BaseURL: "https://api.deepseek.com/v1",
		Model:   "deepseek-chat",
		APIKey:  "sk-test123",
	}

	// 验证字段可访问
	if cfg.BaseURL == "" {
		t.Error("BaseURL 不应为空")
	}
	if cfg.Model == "" {
		t.Error("Model 不应为空")
	}
	if cfg.APIKey == "" {
		t.Error("APIKey 不应为空")
	}
}

// TestConfigWithDeepSeekDefaults 测试使用 DeepSeek 默认配置
func TestConfigWithDeepSeekDefaults(t *testing.T) {
	cfg := Config{
		BaseURL: "https://api.deepseek.com/v1",
		Model:   "deepseek-chat",
		APIKey:  "sk-1a33358662ac4e62a69effd2cdc046be",
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("DeepSeek 配置应有效，但返回错误: %v", err)
	}
}

// TestConfigValidateErrorMessages 测试错误消息
func TestConfigValidateErrorMessages(t *testing.T) {
	tests := []struct {
		name     string
		cfg      Config
		contains string
	}{
		{
			name:     "BaseURL 错误消息",
			cfg:      Config{BaseURL: "", Model: "test", APIKey: "test"},
			contains: "llm_base_url",
		},
		{
			name:     "Model 错误消息",
			cfg:      Config{BaseURL: "https://test.com", Model: "", APIKey: "test"},
			contains: "llm_model",
		},
		{
			name:     "APIKey 错误消息",
			cfg:      Config{BaseURL: "https://test.com", Model: "test", APIKey: ""},
			contains: "llm_api_key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if err == nil {
				t.Fatal("期望错误但未返回")
			}
			if !containsStr(err.Error(), tt.contains) {
				t.Errorf("错误消息应包含 '%s'，实际: %s", tt.contains, err.Error())
			}
		})
	}
}

func containsStr(s, substr string) bool {
	return len(substr) <= len(s) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
