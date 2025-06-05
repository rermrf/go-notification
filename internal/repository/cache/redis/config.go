package redis

import (
	"context"
	"encoding/json"
	"errors"
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

func (c *Cache) Get(ctx context.Context, bizID int64) (domain.BusinessConfig, error) {
	key := cache.ConfigKey(bizID)
	// 从redis中获取数据
	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			// 键不存在
			return domain.BusinessConfig{}, cache.ErrKeyNotFound
		}
		return domain.BusinessConfig{}, err
	}
	// 反序列化
	var businessConfig domain.BusinessConfig
	err = json.Unmarshal([]byte(val), &businessConfig)
	if err != nil {
		return domain.BusinessConfig{}, fmt.Errorf("序列化数据失败：%w", err)
	}
	return businessConfig, nil
}

func (c *Cache) Set(ctx context.Context, cfg domain.BusinessConfig) error {
	key := cache.ConfigKey(cfg.ID)

	// 序列化数据
	data, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("序列化数据失败：%w", err)
	}
	err = c.client.Set(ctx, key, data, cache.DefaultExpriredTime).Err()
	if err != nil {
		return fmt.Errorf("redis 存储数据失败：%w", err)
	}
	return nil
}

func (c *Cache) Delete(ctx context.Context, bizID int64) error {
	return c.client.Del(ctx, cache.ConfigKey(bizID)).Err()
}

func (c *Cache) GetConfigs(ctx context.Context, bizIDs []int64) (map[int64]domain.BusinessConfig, error) {
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

func (c *Cache) SetConfigs(ctx context.Context, cfgs []domain.BusinessConfig) error {
	if len(cfgs) == 0 {
		return nil
	}

	// 使用管道设置，提供性能
	// 这边是一个性能优化的写法
	// 在集群模式下，命中同一个节点的 key 会被打包作为一个 pipeline
	// 确保你的 redis 客户端支持自动分组/智能路由
	pipe := c.client.Pipeline()

	for _, cfg := range cfgs {
		key := cache.ConfigKey(cfg.ID)

		// 序列化数据
		data, err := json.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("序列化数据失败：%w", err)
		}
		pipe.Set(ctx, key, data, cache.DefaultExpriredTime)
	}

	// 执行管道中的所有命令
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("pipeline 执行 set 操作失败：%w", err)
	}
	return nil
}
