package config

import (
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

type Config struct {
	Database struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		DBName   string `yaml:"dbname"`
	} `yaml:"database"`
	Admins []string `yaml:"admins"` // 添加管理员邮箱列表
	Redis  struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		Password string `yaml:"password"`
		DB       int    `yaml:"db"`
	} `yaml:"redis"` // Dify 使用的 Redis（用于清除登录失败限流计数）
	Dify   struct {
		ConsoleAPIURL  string   `yaml:"console_api_url"`
		AdminEmail     string   `yaml:"admin_email"`
		AdminPassword  string   `yaml:"admin_password"`
		DefaultPlugins []string `yaml:"default_plugins"`
		InnerAPIURL    string   `yaml:"inner_api_url"`  // Dify 内部接口地址（留空则使用 console_api_url）
		InnerAPIKey    string   `yaml:"inner_api_key"`  // Dify 内部接口密钥（对应 api 服务的 INNER_API_KEY 环境变量）
	} `yaml:"dify"`
}

var GlobalConfig Config

func LoadConfig() error {
	// 首先尝试在当前工作目录读取 config.yaml
	configPath := "config.yaml"
	data, err := os.ReadFile(configPath)
	if err != nil {
		// 如果失败，尝试在程序执行路径读取
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		configPath = filepath.Join(filepath.Dir(exe), "config.yaml")
		data, err = os.ReadFile(configPath)
		if err != nil {
			return err
		}
	}

	// 解析配置文件
	err = yaml.Unmarshal(data, &GlobalConfig)
	if err != nil {
		return err
	}

	return nil
}
