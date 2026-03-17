package config

import (
	"os"

	"github.com/joho/godotenv"
)

// Config 配置结构体  (我直接照搬了，要改可以删掉也可以留)
type Config struct {
	ModelScopeKey string
}

// AppConfig 全局配置
var AppConfig *Config

// LoadConfig 加载配置
func LoadConfig() error {
	err := godotenv.Load()
	if err != nil {
	}

	AppConfig = &Config{
		ModelScopeKey: os.Getenv("MODELSCOPE_KEY"),
	}

	return nil
}
