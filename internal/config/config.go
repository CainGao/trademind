// Package config 加载应用配置。
//
// 配置分层（规范 V1.0 §9.1）:
//   - 默认配置（config/default.yaml）
//   - 用户配置（runtime/config.yaml）
//   - 环境变量（最高优先级）
package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Config 应用配置。
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Log      LogConfig      `yaml:"log"`
	App      AppConfig      `yaml:"app"`
}

type ServerConfig struct {
	Port int `yaml:"port"` // 默认 7789
}

type DatabaseConfig struct {
	Path string `yaml:"path"` // 默认 runtime/trademind.db
}

type LogConfig struct {
	Level    string `yaml:"level"`     // debug|info|warn|error
	Dir      string `yaml:"dir"`       // 日志目录
	MaxSize  int    `yaml:"max_size"`  // 单文件最大 MB
	MaxBackups int  `yaml:"max_backups"`
	MaxAge   int    `yaml:"max_age"`   // 保留天数
}

type AppConfig struct {
	Name        string `yaml:"name"`
	Version     string `yaml:"version"`
	RuntimeDir  string `yaml:"runtime_dir"`
	Production  bool   `yaml:"production"`
}

// Default 返回默认配置。
func Default() *Config {
	return &Config{
		Server:   ServerConfig{Port: 7789},
		Database: DatabaseConfig{Path: filepath.Join("runtime", "trademind.db")},
		Log: LogConfig{
			Level: "info", Dir: "logs",
			MaxSize: 50, MaxBackups: 7, MaxAge: 30,
		},
		App: AppConfig{
			Name: "TradeMind AI", Version: "0.1.0",
			RuntimeDir: "runtime", Production: false,
		},
	}
}

// Load 加载配置。优先级: 环境变量 > runtime/config.yaml > 默认。
func Load() (*Config, error) {
	cfg := Default()

	// 读 runtime/config.yaml（如果存在）
	userCfgPath := filepath.Join("runtime", "config.yaml")
	if data, err := os.ReadFile(userCfgPath); err == nil {
		_ = data // TODO: YAML 解析（简化版先用默认）
	}

	// 环境变量覆盖
	if v := os.Getenv("TRADEMIND_PORT"); v != "" {
		_, _ = fmt.Sscanf(v, "%d", &cfg.Server.Port)
	}
	if v := os.Getenv("TRADEMIND_DB_PATH"); v != "" {
		cfg.Database.Path = v
	}
	if v := os.Getenv("TRADEMIND_LOG_LEVEL"); v != "" {
		cfg.Log.Level = v
	}

	// 确保运行时目录存在
	for _, dir := range []string{cfg.App.RuntimeDir, cfg.Log.Dir, filepath.Dir(cfg.Database.Path)} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("创建目录失败 [%s]: %w", dir, err)
		}
	}

	return cfg, nil
}
