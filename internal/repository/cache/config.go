package cache

import (
	"context"
	"errors"
	"fmt"
	"go-notification/internal/domain"
	"time"
)

var ErrKeyNotFound = errors.New("key not found")

const (
	ConfigPrefix        = "config"
	DefaultExpriredTime = 10 * time.Minute
)

type ConfigCache interface {
	Get(ctx context.Context, bizID int64) (domain.BusinessConfig, error)
	Set(ctx context.Context, cfg domain.BusinessConfig) error
	Delete(ctx context.Context, bizID int64) error
	GetConfigs(ctx context.Context, bizIDs []int64) (map[int64]domain.BusinessConfig, error)
	SetConfigs(ctx context.Context, cfgs []domain.BusinessConfig) error
}

func ConfigKey(bizID int64) string {
	return fmt.Sprintf("%s:%d", ConfigPrefix, bizID)
}
