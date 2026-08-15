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

	"gopkg.in/yaml.v3"
)

// AppVersion 可被 ldflags 覆盖（Makefile: -X 'config.AppVersion=1.0.0'）。
var AppVersion = "0.1.0"

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
	Production  bool   `yaml:"production"` // 默认 true：私有化桌面部署即生产形态；开发时设 TRADEMIND_PRODUCTION=false
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
			Name: "TradeMind AI", Version: AppVersion,
			RuntimeDir: "runtime", Production: true,
		},
	}
}

// Load 加载配置。优先级: 环境变量 > runtime/config.yaml > 默认。
// 向后兼容包装：LoadFromPath("")。
func Load() (*Config, error) {
	return LoadFromPath("")
}

// LoadFromPath 从指定路径加载 YAML 配置（空路径则读 runtime/config.yaml）。
// 优先级: 环境变量 > config.yaml > 默认值。
func LoadFromPath(customPath string) (*Config, error) {
	cfg := Default()

	// 读 YAML 配置文件（如果存在），解析后覆盖默认值
	cfgPath := customPath
	if cfgPath == "" {
		cfgPath = filepath.Join("runtime", "config.yaml")
	}
	if data, err := os.ReadFile(cfgPath); err == nil {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("解析配置文件失败 [%s]: %w", cfgPath, err)
		}
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
	// 生产模式开关：TRADEMIND_PRODUCTION=false/0 关闭（开发模式，Gin debug 日志）
	if v := os.Getenv("TRADEMIND_PRODUCTION"); v != "" {
		cfg.App.Production = v != "false" && v != "0"
	}

	// 确保运行时目录存在
	for _, dir := range []string{cfg.App.RuntimeDir, cfg.Log.Dir, filepath.Dir(cfg.Database.Path)} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("创建目录失败 [%s]: %w", dir, err)
		}
	}

	return cfg, nil
}
