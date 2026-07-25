package services

import (
	"context"
	"difyserver/config"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// ClearLoginErrorRateLimit 清除 Dify 登录失败限流计数
//
// Dify 在连续登录失败达到阈值（默认 5 次 / 24 小时）后，会在 Redis 中记录
// login_error_rate_limit:{email} 计数，并对该账号的所有登录请求返回
// "Too many incorrect password attempts. Please try again later."，
// 即使密码正确也会被拒绝。管理员重置密码后需要一并清除该计数。
func ClearLoginErrorRateLimit(email string) error {
	cfg := config.GlobalConfig.Redis
	if cfg.Host == "" {
		// 未配置 Redis 时跳过（不阻塞密码重置流程）
		return nil
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	defer rdb.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Dify RateLimiter 的 key 前缀为 login_error_rate_limit
	key := fmt.Sprintf("login_error_rate_limit:%s", email)
	if err := rdb.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("清除登录限流计数失败: %w", err)
	}
	return nil
}
