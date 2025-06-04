package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/redis/go-redis/v9"
	"go-notification/internal/domain"
	"go-notification/internal/pkg/logger"
	"go-notification/internal/repository/cache"
)

type Cache struct {
	client redis.Cmdable
	logger logger.Logger
}

func NewCache(client redis.Cmdable, logger logger.Logger) cache.ConfigCache {
	return &Cache{}
}

func (c Cache) Get(ctx context.Context, bizID int64) (domain.BusinessConfig, error) {
	//TODO implement me
	panic("implement me")
}

func (c Cache) Set(ctx context.Context, cfg domain.BusinessConfig) error {
	//TODO implement me
	panic("implement me")
}

func (c Cache) Delete(ctx context.Context, bizID int64) error {
	return c.client.Del(ctx, cache.ConfigKey(bizID)).Err()
}

func (c Cache) GetConfigs(ctx context.Context, bizIDs []int64) (map[int64]domain.BusinessConfig, error) {
	if len(bizIDs) == 0 {
		return nil, nil
	}
	// 准备所有的键
	keys := make([]string, len(bizIDs))
	for i, bizID := range bizIDs {
		keys[i] = cache.ConfigKey(bizID)
	}

	// 使用 MGET 批量获取数据
	vals, err := c.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("redis 执行 MGET 失败: %w", err)
	}

	// 处理结果
	res := make(map[int64]domain.BusinessConfig)
	for i, val := range vals {
		if val == nil {
			continue // 跳过
		}

		// 将字符串转换为结构体
		strVal, ok := val.(string)
		if !ok {
			continue
		}

		var cfg domain.BusinessConfig
		if err := json.Unmarshal([]byte(strVal), &cfg); err != nil {
			// 解析错误，记录错误但是继续处理其他配置
			c.logger.Error("从 redis 序列化数据失败", logger.Error(err), logger.String("key", keys[i]), logger.String("val", strVal))
			continue
		}
		res[bizIDs[i]] = cfg
	}
	return res, nil
}

func (c Cache) SetConfigs(ctx context.Context, cfgs []domain.BusinessConfig) error {
	//TODO implement me
	panic("implement me")
}
