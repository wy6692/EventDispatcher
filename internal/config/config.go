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
	if err := loadDotEnv(".env"); err != nil {
		return Config{}, err
	}
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

// loadDotEnv 读取工作目录下的 .env。文件不存在时忽略。
// 已经存在于进程环境中的变量不会被覆盖。
func loadDotEnv(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("读取 %s: %w", path, err)
	}
	text := strings.TrimPrefix(string(data), "\uFEFF")
	for i, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf("%s 第 %d 行缺少 '='", path, i+1)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			return fmt.Errorf("%s 第 %d 行变量名为空", path, i+1)
		}
		if len(value) >= 2 {
			q := value[0]
			if (q == '"' || q == '\'') && value[len(value)-1] == q {
				value = value[1 : len(value)-1]
			}
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("设置环境变量 %s: %w", key, err)
		}
	}
	return nil
}
