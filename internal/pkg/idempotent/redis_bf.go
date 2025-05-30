package idempotent

import (
	"context"
	"errors"
	"github.com/redis/go-redis/v9"
)

type BloomIdempotencyService struct {
	client     redis.Cmdable
	filterName string
	capacity   uint64  // 预期容量
	errorRate  float64 // 错误率
}

func (s *BloomIdempotencyService) Exists(ctx context.Context, key string) (bool, error) {
	res, err := s.client.BFAdd(ctx, s.filterName, key).Result()
	return !res, err
}

func (s *BloomIdempotencyService) MExists(ctx context.Context, keys ...string) ([]bool, error) {
	if len(keys) == 0 {
		return nil, errors.New("empty keys")
	}
	// 批量查询
	var key1s []string
	for _, key := range keys {
		key1s = append(key1s, key)
	}
	res := s.client.BFMAdd(ctx, s.filterName, key1s)
	vals, err := res.Result()
	if err != nil {
		return nil, err
	}
	var results []bool
	for _, val := range vals {
		results = append(results, val)
	}
	return results, nil
}
