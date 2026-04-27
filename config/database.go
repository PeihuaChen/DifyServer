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
	Dify   struct {
		ConsoleAPIURL  string   `yaml:"console_api_url"`
		AdminEmail     string   `yaml:"admin_email"`
		AdminPassword  string   `yaml:"admin_password"`
		DefaultPlugins []string `yaml:"default_plugins"`
		StoragePath    string   `yaml:"storage_path"`  // Dify storage 路径
		SSHHost        string   `yaml:"ssh_host"`      // SSH 主机地址（远程写入私钥用）
		SSHPort        int      `yaml:"ssh_port"`      // SSH 端口
		SSHUser        string   `yaml:"ssh_user"`      // SSH 用户名
		SSHPassword    string   `yaml:"ssh_password"`  // SSH 密码
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
