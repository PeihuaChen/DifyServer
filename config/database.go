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
