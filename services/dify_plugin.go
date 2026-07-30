package services

import (
	"bytes"
	"difyserver/config"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

// innerAPIHint 内部接口不通时的环境变量 / NGINX 配置提示
const innerAPIHint = "请确认 Dify 内部接口已开启：" +
	"1) 在 api 服务增加环境变量 INNER_API=true、INNER_API_KEY=sk-***；" +
	"2) 在 NGINX 增加 /inner/api 路由：location /inner/api { proxy_pass http://api:5001; include proxy.conf; }；" +
	"3) 在 config.yaml 的 dify.inner_api_key 中填写与 INNER_API_KEY 一致的密钥"

// CreateWorkspaceResponse Dify 内部接口创建工作空间的响应
type CreateWorkspaceResponse struct {
	Message string `json:"message"`
	Tenant  struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"tenant"`
}

// PluginInstallResponse 插件安装响应
type PluginInstallResponse struct {
	AllInstalled bool   `json:"all_installed"`
	TaskID       string `json:"task_id"`
}

// DifyClient Dify API 客户端
type DifyClient struct {
	BaseURL   string
	CSRFToken string // 登录后从 cookie 中获取的 CSRF token
	HTTP      *http.Client
}

func NewDifyClient() *DifyClient {
	jar, _ := cookiejar.New(nil)
	return &DifyClient{
		BaseURL: config.GlobalConfig.Dify.ConsoleAPIURL,
		HTTP: &http.Client{
			Timeout: 60 * time.Second,
			Jar:     jar,
		},
	}
}

// doPost 发送 POST 请求，自动附带 CSRF token
func (c *DifyClient) doPost(url string, body interface{}) (*http.Response, error) {
	jsonBody, _ := json.Marshal(body)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.CSRFToken != "" {
		req.Header.Set("X-CSRF-Token", c.CSRFToken)
	}
	return c.HTTP.Do(req)
}

// LoginToDify 登录 Dify，token 和 csrf_token 通过 Set-Cookie 返回
func (c *DifyClient) LoginToDify(email, password string) error {
	loginURL := fmt.Sprintf("%s/console/api/login", c.BaseURL)
	// Dify 新版本要求 password 字段经过 Base64 编码传输
	encodedPassword := base64.StdEncoding.EncodeToString([]byte(password))
	body := map[string]interface{}{
		"email":       email,
		"password":    encodedPassword,
		"remember_me": true,
	}
	jsonBody, _ := json.Marshal(body)

	resp, err := c.HTTP.Post(loginURL, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("请求 Dify 登录接口失败: %v", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return fmt.Errorf("Dify 登录失败, status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	// 从 Set-Cookie 中提取 csrf_token
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "csrf_token" {
			c.CSRFToken = cookie.Value
		}
	}

	// 也从 cookie jar 中查找（某些情况下 cookie 可能已被 jar 吸收）
	if c.CSRFToken == "" && c.HTTP.Jar != nil {
		parsedURL, _ := url.Parse(c.BaseURL)
		for _, cookie := range c.HTTP.Jar.Cookies(parsedURL) {
			if cookie.Name == "csrf_token" {
				c.CSRFToken = cookie.Value
			}
		}
	}

	// 验证登录成功
	hasAccessToken := false
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "access_token" {
			hasAccessToken = true
			break
		}
	}
	if !hasAccessToken && c.HTTP.Jar != nil {
		parsedURL, _ := url.Parse(c.BaseURL)
		for _, cookie := range c.HTTP.Jar.Cookies(parsedURL) {
			if cookie.Name == "access_token" {
				hasAccessToken = true
				break
			}
		}
	}

	if !hasAccessToken {
		return fmt.Errorf("登录成功但未获取到 access_token, cookies=%v", resp.Header.Values("Set-Cookie"))
	}

	fmt.Printf("[DifyClient] 登录成功, csrf_token=%s\n", c.CSRFToken)
	return nil
}

// SwitchWorkspace 切换当前用户的活跃 workspace
func (c *DifyClient) SwitchWorkspace(tenantID string) error {
	switchURL := fmt.Sprintf("%s/console/api/workspaces/switch", c.BaseURL)
	body := map[string]string{
		"tenant_id": tenantID,
	}

	resp, err := c.doPost(switchURL, body)
	if err != nil {
		return fmt.Errorf("请求切换工作空间接口失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("切换工作空间失败, status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	return nil
}

// InstallPluginsFromMarketplace 为指定 tenant 安装 marketplace 插件
func (c *DifyClient) InstallPluginsFromMarketplace(tenantID string, pluginIdentifiers []string) (*PluginInstallResponse, error) {
	if len(pluginIdentifiers) == 0 {
		return nil, fmt.Errorf("插件列表为空")
	}

	// 先切换到目标 workspace
	if err := c.SwitchWorkspace(tenantID); err != nil {
		return nil, fmt.Errorf("切换工作空间失败: %v", err)
	}

	// 调用插件安装接口
	installURL := fmt.Sprintf("%s/console/api/workspaces/current/plugin/install/marketplace", c.BaseURL)
	body := map[string]interface{}{
		"plugin_unique_identifiers": pluginIdentifiers,
	}

	resp, err := c.doPost(installURL, body)
	if err != nil {
		return nil, fmt.Errorf("请求插件安装接口失败: %v", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("插件安装失败, status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	var installResp PluginInstallResponse
	if err := json.Unmarshal(respBody, &installResp); err != nil {
		return nil, fmt.Errorf("解析插件安装响应失败: %v, body=%s", err, string(respBody))
	}

	return &installResp, nil
}

// CreateWorkspace 通过 Dify 内部接口(Inner API) 创建工作空间。
// owner_email 必须是已存在的 Dify 账号，成功后返回新建 tenant 的 id。
// 对应接口: POST /inner/api/enterprise/workspace
func (c *DifyClient) CreateWorkspace(name, ownerEmail string) (string, error) {
	return c.doCreateWorkspace("/inner/api/enterprise/workspace", map[string]string{
		"name":        name,
		"owner_email": ownerEmail,
	})
}

// CreateWorkspaceOwnerless 通过 Dify 内部接口创建无 owner 的工作空间，仅需 name。
// 对应接口: POST /inner/api/enterprise/workspace/ownerless
func (c *DifyClient) CreateWorkspaceOwnerless(name string) (string, error) {
	return c.doCreateWorkspace("/inner/api/enterprise/workspace/ownerless", map[string]string{
		"name": name,
	})
}

// doCreateWorkspace 向 Dify 内部接口发送创建工作空间请求的通用实现。
// 该接口依赖 api 服务的 INNER_API=true / INNER_API_KEY 环境变量以及 NGINX /inner/api 路由，
// 接口不通时返回带有配置提示的错误。
func (c *DifyClient) doCreateWorkspace(path string, body map[string]string) (string, error) {
	cfg := config.GlobalConfig.Dify

	if cfg.InnerAPIKey == "" {
		return "", fmt.Errorf("未配置 Dify 内部接口密钥(dify.inner_api_key)。%s", innerAPIHint)
	}

	baseURL := cfg.InnerAPIURL
	if baseURL == "" {
		baseURL = cfg.ConsoleAPIURL
	}
	if baseURL == "" {
		return "", fmt.Errorf("未配置 Dify 接口地址(dify.inner_api_url 或 dify.console_api_url)")
	}

	wsURL := fmt.Sprintf("%s%s", strings.TrimRight(baseURL, "/"), path)
	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", wsURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Inner-Api-Key", cfg.InnerAPIKey)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		// 网络层面不通（连接被拒、超时、DNS 等）
		return "", fmt.Errorf("请求创建工作空间接口失败(%s): %v。%s", wsURL, err, innerAPIHint)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	switch resp.StatusCode {
	case 200, 201:
		// 创建成功，继续解析
	case 401, 403:
		return "", fmt.Errorf("创建工作空间鉴权失败(status=%d)：X-Inner-Api-Key 与 api 服务的 INNER_API_KEY 不一致，或未开启 INNER_API。%s (body=%s)", resp.StatusCode, innerAPIHint, string(respBody))
	case 404:
		return "", fmt.Errorf("创建工作空间接口不可用(404)：未找到 %s（可能 owner 账号不存在，或 NGINX 未配置 /inner/api 路由、api 服务未开启 INNER_API）。%s (body=%s)", path, innerAPIHint, string(respBody))
	default:
		return "", fmt.Errorf("创建工作空间失败, status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	var wsResp CreateWorkspaceResponse
	if err := json.Unmarshal(respBody, &wsResp); err != nil {
		return "", fmt.Errorf("解析创建工作空间响应失败: %v, body=%s", err, string(respBody))
	}
	if wsResp.Tenant.ID == "" {
		return "", fmt.Errorf("创建工作空间成功但未返回 tenant id, body=%s", string(respBody))
	}

	fmt.Printf("[DifyClient] 工作空间创建成功: id=%s, name=%s\n", wsResp.Tenant.ID, wsResp.Tenant.Name)
	return wsResp.Tenant.ID, nil
}

// InstallDefaultPlugins 为新创建的 tenant 安装默认插件
func InstallDefaultPlugins(tenantID string) error {
	cfg := config.GlobalConfig.Dify
	if cfg.ConsoleAPIURL == "" || cfg.AdminEmail == "" || cfg.AdminPassword == "" {
		return fmt.Errorf("Dify 配置不完整，跳过插件安装")
	}
	if len(cfg.DefaultPlugins) == 0 {
		return nil
	}

	client := NewDifyClient()

	if err := client.LoginToDify(cfg.AdminEmail, cfg.AdminPassword); err != nil {
		return fmt.Errorf("登录 Dify 失败: %v", err)
	}

	result, err := client.InstallPluginsFromMarketplace(tenantID, cfg.DefaultPlugins)
	if err != nil {
		return fmt.Errorf("安装插件失败: %v", err)
	}

	if result.AllInstalled {
		fmt.Printf("[DifyPlugin] 租户 %s 的默认插件已全部安装完成\n", tenantID)
	} else {
		fmt.Printf("[DifyPlugin] 租户 %s 的插件安装任务已创建, task_id=%s\n", tenantID, result.TaskID)
	}

	return nil
}
