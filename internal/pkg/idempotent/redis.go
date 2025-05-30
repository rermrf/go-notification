package idempotent

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"time"
)

type RedisIdempotencyService struct {
	client redis.Cmdable
	expiry time.Duration // 添加过期时间配置
}

// NewRedisIdempotencyService 创建一个新的 redis 幂等性服务
func NewRedisIdempotencyService(client redis.Cmdable, expiry time.Duration) *RedisIdempotencyService {
	return &RedisIdempotencyService{client: client, expiry: expiry}
}

func (s *RedisIdempotencyService) getKey(key string) string {
	return fmt.Sprintf("idempotency:%s", key)
}

func (s *RedisIdempotencyService) Exists(ctx context.Context, key string) (bool, error) {
	result, err := s.client.SetNX(ctx, s.getKey(key), "1", s.expiry).Result()
	if err != nil {
		return false, err
	}
	return !result, nil
}

func (s *RedisIdempotencyService) MExists(ctx context.Context, keys ...string) ([]bool, error) {
	// 使用管道批量执行SetNX命令
	pipe := s.client.Pipeline()
	cmds := make([]*redis.BoolCmd, len(keys))

	for i, key := range keys {
		cmds[i] = pipe.SetNX(ctx, s.getKey(key), "1", s.expiry)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, err
	}
	results := make([]bool, len(keys))
	for i, cmd := range cmds {
		exists, err := cmd.Result()
		if err != nil {
			return nil, err
		}
		results[i] = !exists
	}
	return results, nil
}
