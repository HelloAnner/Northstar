/**
 * LLM 配置读取与校验
 *
 * @author Anner
 * @since 12.0
 * Created on 2026/2/1
 */
package llm

import (
	"fmt"
	"strings"

	"northstar/internal/store"
)

// Config 大模型配置
type Config struct {
	BaseURL string
	Model   string
	APIKey  string
}

// LoadConfig 从数据库读取大模型配置
func LoadConfig(st *store.Store) (Config, error) {
	baseURL, err := st.GetConfig("llm_base_url")
	if err != nil {
		return Config{}, err
	}
	model, err := st.GetConfig("llm_model")
	if err != nil {
		return Config{}, err
	}
	apiKey, err := st.GetConfig("llm_api_key")
	if err != nil {
		return Config{}, err
	}
	cfg := Config{
		BaseURL: strings.TrimSpace(baseURL),
		Model:   strings.TrimSpace(model),
		APIKey:  strings.TrimSpace(apiKey),
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Validate 校验配置是否完整
func (c Config) Validate() error {
	if c.BaseURL == "" {
		return fmt.Errorf("llm_base_url 不能为空")
	}
	if c.Model == "" {
		return fmt.Errorf("llm_model 不能为空")
	}
	if c.APIKey == "" {
		return fmt.Errorf("llm_api_key 不能为空")
	}
	return nil
}
