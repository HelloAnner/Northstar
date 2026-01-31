package config

import (
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

// AppConfig 应用配置
type AppConfig struct {
	Server   ServerConfig   `toml:"server"`
	Data     DataConfig     `toml:"data"`
	Business BusinessConfig `toml:"business"`
	Excel    ExcelConfig    `toml:"excel"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port    int  `toml:"port"`
	DevMode bool `toml:"dev_mode"`
}

// DataConfig 数据配置
type DataConfig struct {
	DataDir    string `toml:"data_dir"`
	AutoBackup bool   `toml:"auto_backup"`
}

// BusinessConfig 业务配置
type BusinessConfig struct {
	DefaultMonth int     `toml:"default_month"`
	MaxGrowth    float64 `toml:"max_growth"`
	MinGrowth    float64 `toml:"min_growth"`
}

// ExcelConfig Excel 导出相关配置
type ExcelConfig struct {
	TemplatePath string `toml:"template_path"`
}

// LoadConfigInfo 配置加载元信息
type LoadConfigInfo struct {
	PortSpecified bool
}

// DefaultConfig 默认配置
func DefaultConfig() *AppConfig {
	return &AppConfig{
		Server: ServerConfig{
			Port:    20261,
			DevMode: false,
		},
		Data: DataConfig{
			DataDir:    "data",
			AutoBackup: true,
		},
		Business: BusinessConfig{
			DefaultMonth: 1,
			MaxGrowth:    0.5,
			MinGrowth:    -0.3,
		},
		Excel: ExcelConfig{
			TemplatePath: "",
		},
	}
}

func isPortSpecifiedInToml(data []byte) bool {
	var raw map[string]any
	if err := toml.Unmarshal(data, &raw); err != nil {
		return false
	}

	serverAny, ok := raw["server"]
	if !ok {
		return false
	}

	serverMap, ok := serverAny.(map[string]any)
	if !ok {
		return false
	}

	_, ok = serverMap["port"]
	return ok
}

// GetExeDir 获取可执行文件所在目录
func GetExeDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(exe), nil
}

// LoadConfigWithInfo 从 config.toml 加载配置并返回元信息
func LoadConfigWithInfo() (*AppConfig, LoadConfigInfo, error) {
	info := LoadConfigInfo{}
	config := DefaultConfig()

	exeDir, err := GetExeDir()
	if err != nil {
		// 无法获取可执行文件目录，使用当前目录
		exeDir = "."
	}

	configPath := filepath.Join(exeDir, "config.toml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// 配置文件不存在，使用默认配置
			return config, info, nil
		}
		return nil, info, err
	}

	info.PortSpecified = isPortSpecifiedInToml(data)

	if err := toml.Unmarshal(data, config); err != nil {
		return nil, info, err
	}

	// 环境变量覆盖（用于 E2E / 本地运行）
	if v := os.Getenv("NORTHSTAR_EXCEL_TEMPLATE_PATH"); v != "" {
		config.Excel.TemplatePath = v
	}
	if config.Excel.TemplatePath == "" {
		if v := os.Getenv("NS_MONTH_REPORT_TEMPLATE_XLSX"); v != "" {
			config.Excel.TemplatePath = v
		}
	}

	return config, info, nil
}

// LoadConfig 从 config.toml 加载配置
// 配置文件位于可执行文件同目录下
func LoadConfig() (*AppConfig, error) {
	config, _, err := LoadConfigWithInfo()
	return config, err
}

// SaveConfig 保存配置到 config.toml
func SaveConfig(config *AppConfig) error {
	exeDir, err := GetExeDir()
	if err != nil {
		exeDir = "."
	}

	configPath := filepath.Join(exeDir, "config.toml")

	data, err := toml.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

// EnsureDataDir 确保数据目录存在
// 数据目录位于可执行文件同目录下
func EnsureDataDir(config *AppConfig) (string, error) {
	exeDir, err := GetExeDir()
	if err != nil {
		exeDir = "."
	}

	dataDir := filepath.Join(exeDir, config.Data.DataDir)

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return "", err
	}

	// 创建子目录
	subdirs := []string{"uploads", "exports", "backups"}
	for _, subdir := range subdirs {
		path := filepath.Join(dataDir, subdir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return "", err
		}
	}

	return dataDir, nil
}

// GetDataPath 获取数据文件路径
func GetDataPath(config *AppConfig, subdir, filename string) string {
	exeDir, _ := GetExeDir()
	if exeDir == "" {
		exeDir = "."
	}
	return filepath.Join(exeDir, config.Data.DataDir, subdir, filename)
}
