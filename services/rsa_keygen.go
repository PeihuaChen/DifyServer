package services

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"difyserver/config"
	"encoding/pem"
	"fmt"

	"golang.org/x/crypto/ssh"
)

// GenerateKeyPair 为 tenant 生成 RSA 密钥对
// 私钥通过 SSH 保存到远程 Dify storage 的 privkeys/{tenant_id}/private.pem
// 返回 PEM 格式的公钥字符串
func GenerateKeyPair(tenantID string) (string, error) {
	cfg := config.GlobalConfig.Dify
	if cfg.StoragePath == "" {
		return "", fmt.Errorf("dify.storage_path 未配置")
	}
	if cfg.SSHHost == "" || cfg.SSHUser == "" || cfg.SSHPassword == "" {
		return "", fmt.Errorf("dify SSH 配置不完整")
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", fmt.Errorf("生成 RSA 密钥失败: %v", err)
	}

	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	pubKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", fmt.Errorf("编码公钥失败: %v", err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	})

	privKeyDir := fmt.Sprintf("%s/privkeys/%s", cfg.StoragePath, tenantID)
	privKeyPath := fmt.Sprintf("%s/private.pem", privKeyDir)

	if err := writeFileViaSSH(privKeyDir, privKeyPath, privPEM); err != nil {
		return "", fmt.Errorf("SSH 写入私钥失败: %v", err)
	}

	fmt.Printf("[RSA] 已为租户 %s 生成密钥对, 远程路径: %s\n", tenantID, privKeyPath)
	return string(pubPEM), nil
}

func writeFileViaSSH(dir, filePath string, content []byte) error {
	cfg := config.GlobalConfig.Dify
	port := cfg.SSHPort
	if port == 0 {
		port = 22
	}

	sshConfig := &ssh.ClientConfig{
		User: cfg.SSHUser,
		Auth: []ssh.AuthMethod{
			ssh.Password(cfg.SSHPassword),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", cfg.SSHHost, port), sshConfig)
	if err != nil {
		return fmt.Errorf("SSH 连接失败: %v", err)
	}
	defer client.Close()

	// 创建目录
	sess1, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("创建 session 失败: %v", err)
	}
	if err := sess1.Run(fmt.Sprintf("mkdir -p %s", dir)); err != nil {
		sess1.Close()
		return fmt.Errorf("创建目录失败: %v", err)
	}
	sess1.Close()

	// 写入文件
	sess2, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("创建 session 失败: %v", err)
	}
	defer sess2.Close()

	sess2.Stdin = bytes.NewReader(content)
	if err := sess2.Run(fmt.Sprintf("cat > %s && chmod 600 %s", filePath, filePath)); err != nil {
		return fmt.Errorf("写入文件失败: %v", err)
	}

	return nil
}
