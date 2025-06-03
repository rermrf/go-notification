package redis

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"go-notification/internal/domain"
	"go-notification/internal/pkg/logger"
	"go-notification/internal/repository/cache"
)

var (
	ErrQuotaLessThenZero = errors.New("额度小于0")
	//go:embed lua/quota.lua
	quotaScript string
	//go:embed lua/batch_decr_quota.lua
	batchDecrQuotaScript string
	//go:embed lua/batch_incr_quota.lua
	batchIncrQuotaScript string
)

type quotaCache struct {
	client redis.Cmdable
	logger logger.Logger
}

func NewQuotaCache(client redis.Cmdable, logger logger.Logger) cache.QuotaCache {
	return &quotaCache{
		client: client,
		logger: logger,
	}
}

func (q *quotaCache) CreateOrUpdate(ctx context.Context, quotas ...domain.Quota) error {
	const (
		number = 2
	)
	vals := make([]interface{}, number*len(quotas))
	for _, quota := range quotas {
		vals = append(vals, quota)
	}
	return q.client.MSet(ctx, vals...).Err()
}

func (q *quotaCache) Find(ctx context.Context, bizID int64, channel domain.Channel) (domain.Quota, error) {
	quota, err := q.client.Get(ctx, q.key(domain.Quota{
		BizID:   bizID,
		Channel: channel,
	})).Int()
	if err != nil {
		return domain.Quota{}, err
	}
	return domain.Quota{
		BizID:   bizID,
		Channel: channel,
		Quota:   int32(quota),
	}, nil
}

func (q *quotaCache) Incr(ctx context.Context, bizID int64, channel domain.Channel, quota int32) error {
	return q.client.Eval(ctx, quotaScript, []string{q.key(domain.Quota{
		BizID:   bizID,
		Channel: channel,
	})}, quota).Err()
}

func (q *quotaCache) Decr(ctx context.Context, bizID int64, channel domain.Channel, quota int32) error {
	res, err := q.client.DecrBy(ctx, q.key(domain.Quota{
		BizID:   bizID,
		Channel: channel,
	}), int64(quota)).Result()
	if err != nil {
		return err
	}
	if res <= 0 {
		q.logger.Error("库存不足", logger.Int64("bizID", bizID), logger.String("channel", channel.String()))
		return ErrQuotaLessThenZero
	}
	return nil
}

func (q *quotaCache) MutiIncr(ctx context.Context, items []cache.IncrItem) error {
	if len(items) == 0 {
		return nil
	}
	keys, quotas := q.getKeysAndQuotas(items)
	err := q.client.Eval(ctx, batchIncrQuotaScript, keys, quotas).Err()
	if err != nil {
		return err
	}
	return nil
}

func (q *quotaCache) MutiDecr(ctx context.Context, items []cache.IncrItem) error {
	keys, quotas := q.getKeysAndQuotas(items)
	res, err := q.client.Eval(ctx, batchDecrQuotaScript, keys, quotas).Result()
	if err != nil {
		return err
	}
	resStr, ok := res.(string)
	if !ok {
		return errors.New("返回值不正确")
	}
	if resStr == "" {
		return fmt.Errorf("%s不足 %w", resStr, ErrQuotaLessThenZero)
	}
	return nil
}

func (q *quotaCache) key(quota domain.Quota) string {
	return fmt.Sprintf("quota:%d:%s", quota.BizID, quota.Channel)
}

func (q *quotaCache) getKeysAndQuotas(items []cache.IncrItem) (keys []string, quotas []interface{}) {
	keys = make([]string, 0, len(items))
	quotas = make([]interface{}, 0, len(items))
	for idx := range items {
		item := items[idx]
		keys = append(keys, q.key(domain.Quota{
			BizID:   item.BizID,
			Channel: item.Channel,
		}))
		quotas = append(quotas, item.Val)
	}
	return keys, quotas
}
