package local

import (
	"context"
	"encoding/json"
	"errors"
	ca "github.com/patrickmn/go-cache"
	"github.com/redis/go-redis/v9"
	"go-notification/internal/domain"
	"go-notification/internal/pkg/logger"
	"go-notification/internal/repository/cache"
	"strings"
	"time"
)

const (
	defaultTimeout = time.Second * 5
)

type Cache struct {
	rdb    *redis.Client
	logger logger.Logger
	c      *ca.Cache
}

func NewLocalCache(rdb *redis.Client, logger logger.Logger, c *ca.Cache) cache.ConfigCache {
	localCache := &Cache{
		rdb:    rdb,
		logger: logger,
		c:      c,
	}
	// 开启监控redis中的内容
	// 在这监听 redis 的 key 变更，更新本地缓存
	go localCache.loop(context.Background())
	return localCache
}

func (c *Cache) Get(_ context.Context, bizID int64) (domain.BusinessConfig, error) {
	val, ok := c.c.Get(cache.ConfigKey(bizID))
	if !ok {
		return domain.BusinessConfig{}, cache.ErrKeyNotFound
	}
	res, ok := val.(domain.BusinessConfig)
	if !ok {
		return domain.BusinessConfig{}, errors.New("数据类型不正确")
	}
	return res, nil
}

func (c *Cache) Set(_ context.Context, cfg domain.BusinessConfig) error {
	key := cache.ConfigKey(cfg.ID)
	c.c.Set(key, cfg, cache.DefaultExpriredTime)
	return nil
}

func (c *Cache) Delete(_ context.Context, bizID int64) error {
	key := cache.ConfigKey(bizID)
	c.c.Delete(key)
	return nil
}

func (c *Cache) GetConfigs(ctx context.Context, bizIDs []int64) (map[int64]domain.BusinessConfig, error) {
	configMap := make(map[int64]domain.BusinessConfig)
	for _, bizID := range bizIDs {
		val, ok := c.c.Get(cache.ConfigKey(bizID))
		if ok {
			res, ok := val.(domain.BusinessConfig)
			if !ok {
				return configMap, errors.New("数据类型不正确")
			}
			configMap[bizID] = res
		}
	}
	return configMap, nil
}

func (c *Cache) SetConfigs(_ context.Context, cfgs []domain.BusinessConfig) error {
	for _, cfg := range cfgs {
		c.c.Set(cache.ConfigKey(cfg.ID), cfg, cache.DefaultExpriredTime)
	}
	return nil
}

// 监控 redis 中的数据
func (c *Cache) loop(ctx context.Context) {
	pubsub := c.rdb.PSubscribe(ctx, "__keyspace@*__:config:*")
	defer pubsub.Close()
	ch := pubsub.Channel()
	for msg := range ch {
		// 在线上环境，小心别把敏感数据打出来了
		// 比如说你的 channel 里面包含了手机号码，你就别打了
		c.logger.Info("监听到 Redis 更新消息", logger.String("key", msg.Channel), logger.String("data", msg.Payload))
		const channelMinLen = 2
		channel := msg.Channel
		channelStrList := strings.SplitN(channel, ":", channelMinLen)
		if len(channelStrList) < channelMinLen {
			c.logger.Error("监听到非法 Redis key：", logger.String("channel", channel))
			continue
		}
		// config:133 => 133
		const keyIdx = 1
		key := channelStrList[keyIdx]
		ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
		eventType := msg.Payload
		c.handlerConfigChange(ctx, key, eventType)
		cancel()
	}
}

func (c *Cache) handlerConfigChange(ctx context.Context, key string, eventType string) {
	// 自定义业务逻辑（如动态更新配置）
	switch eventType {
	case "set":
		res := c.rdb.Get(ctx, key)
		if res.Err() != nil {
			c.logger.Error("订阅完获取键失败", logger.String("key", key))
		}
		var cfg domain.BusinessConfig
		err := json.Unmarshal([]byte(res.Val()), &cfg)
		if err != nil {
			c.logger.Error("序列化数据失败", logger.String("key", key))
		}
		c.c.Set(key, cfg, cache.DefaultExpriredTime)
	case "del":
		c.c.Delete(key)
	}
}
