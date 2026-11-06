package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config 只放进程启动参数。服务号和订阅方保存在数据库里。
type Config struct {
	Addr       string
	DBPath     string
	AdminToken string
}

func FromEnv() (Config, error) {
	cfg := Config{
		Addr:       getenv("ADDR", ":8080"),
		DBPath:     getenv("DB_PATH", filepath.Join("data", "dispatcher.db")),
		AdminToken: strings.TrimSpace(os.Getenv("ADMIN_TOKEN")),
	}
	if cfg.AdminToken == "" {
		return Config{}, fmt.Errorf("必须设置环境变量 ADMIN_TOKEN")
	}
	if cfg.Addr == "" {
		return Config{}, fmt.Errorf("ADDR 不能为空")
	}
	if cfg.DBPath == "" {
		return Config{}, fmt.Errorf("DB_PATH 不能为空")
	}
	return cfg, nil
}

func getenv(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}
