package repository

import (
	"context"
	"go-notification/internal/domain"
	log "go-notification/internal/pkg/logger"
	"go-notification/internal/repository/cache"
	"go-notification/internal/repository/dao"
	"time"
)

type BusinessConfigRepository interface {
	LoadCache(ctx context.Context) error
	GetByIDs(ctx context.Context, ids []int64) (map[int64]domain.BusinessConfig, error)
	GetByID(ctx context.Context, id int64) (domain.BusinessConfig, error)
	DeleteByID(ctx context.Context, id int64) error
	SaveConfig(ctx context.Context, config domain.BusinessConfig) error
	Find(ctx context.Context, offset, limit int) ([]domain.BusinessConfig, error)
}

type businessConfigRepository struct {
	dao        dao.BusinessConfigDAO
	localCache cache.ConfigCache
	redisCache cache.ConfigCache
	logger     log.Logger
}

func NewBusinessConfigRepository(dao dao.BusinessConfigDAO, localCache cache.ConfigCache, redisCache cache.ConfigCache, logger log.Logger) BusinessConfigRepository {
	res := &businessConfigRepository{
		dao:        dao,
		localCache: localCache,
		redisCache: redisCache,
		logger:     logger,
	}
	// 再复杂系统里。启动非常慢，可以考虑开 goroutine
	go func() {
		const preloadTimeout = time.Minute
		ctx, cancel := context.WithTimeout(context.Background(), preloadTimeout)
		defer cancel()
		err := res.LoadCache(ctx)
		if err != nil {
			res.logger.Error("预热缓存失败", log.Error(err))
		}
	}()
	return res
}

// LoadCache 加载缓存，用 DB 中的数据，填充本地缓存
func (b *businessConfigRepository) LoadCache(ctx context.Context) error {
	offset := 0
	const (
		limit       = 10
		loopTimeout = time.Second * 3
	)
	for {
		ctx, cancel := context.WithTimeout(ctx, loopTimeout)
		cnt, err := b.loadCacheBatch(ctx, offset, limit)
		cancel()
		if err != nil {
			// 继续下一轮
			// 精细处理：比如说三个循环都是 error，你就判定数据库不可挽回，就中断
			b.logger.Error("分批加载缓存失败", log.Error(err))
			continue
		}
		if cnt < limit {
			return nil
		}
		offset += limit
	}
}

// GetByIDs 根据多个ID批量获取业务配置
// 用在异步请求调度的时候批量处理，批量执行，批量发送
func (b *businessConfigRepository) GetByIDs(ctx context.Context, ids []int64) (map[int64]domain.BusinessConfig, error) {
	// 两种思路，一种是整体从本地缓存、redis缓存、数据库中取
	// 另外一种是从本地缓存取，没取到的从 redis 中取，再没有，从数据库中取。
	//1. 先从本地缓存批量获取
	result, err := b.localCache.GetConfigs(ctx, ids)
	if err != nil {
		b.logger.Error("从本地缓存批量获取失败", log.Error(err))
		// 初始化 map，要注意指定容量，规避扩容引发的性能问题
		result = make(map[int64]domain.BusinessConfig, len(ids))
	}
	// 尝试从 redis 中获取 result 中没有的
	// 取 result 当中没有的

	// 叠加可用性设计，只查询本地缓存
	//if ctx.Value("downgrade") == true {
	//	return result, nil
	//}

	missedIDS := b.diffIDs(ids, result)
	// 没有差异，但是有没有可能本地缓存和redis中都没有的呢
	if len(missedIDS) == 0 {
		return result, nil
	}
	// 2. 从 redis 里面获取
	// 相比之下可能需要查询更少的数据，Redis 传输的数据量也更少，性能会更好
	redisConfigs, err := b.redisCache.GetConfigs(ctx, missedIDS)
	if err != nil {
		b.logger.Error("从 Redis 中批量获取失败", log.Error(err))
	} else {
		// 尝试会写到 本地缓存
		// 需要会写的，以及合并 redisConfig 和 result
		// 这个是精确控制
		configToLocalCache := make([]domain.BusinessConfig, 0, len(redisConfigs))
		for id, conf := range redisConfigs {
			result[id] = conf
			configToLocalCache = append(configToLocalCache, conf)
		}
		// 全部会写，问题不大
		err = b.localCache.SetConfigs(ctx, configToLocalCache)
		if err != nil {
			b.logger.Error("批量会写本地缓存失败", log.Error(err))
		}
	}

	// 可叠加可用性设计，查询 redis 但不查询数据库
	// if ctx.Value("downgrade") == true {
	// if ctx.Value("rate_limit") == true {
	// if ctx.Value("high_load") == true {
	//	return result, err
	// }

	// 从数据库中获取缓存未找到的配置
	missedIDS = b.diffIDs(ids, result)
	// 精确控制，查询更少的 id，回表次数更少
	congifMap, err := b.dao.GetByIDs(ctx, missedIDS)
	if err != nil {
		return nil, err
	}
	// 处理 configMap，回写本地缓存
	configs := make([]domain.BusinessConfig, 0, len(congifMap))
	for id := range congifMap {
		configs = append(configs, b.toDomain(congifMap[id]))
	}

	if len(configs) > 0 {
		err = b.localCache.SetConfigs(ctx, configs)
		if err != nil {
			b.logger.Error("批量回写本地缓存失败", log.Error(err))
		}

		err = b.redisCache.SetConfigs(ctx, configs)
		if err != nil {
			b.logger.Error("批量回写 Redis 缓存失败", log.Error(err))
		}
	}
	return result, err
}

// GetByID 根据ID获取业务配置
func (b *businessConfigRepository) GetByID(ctx context.Context, id int64) (domain.BusinessConfig, error) {
	cfg, localErr := b.localCache.Get(ctx, id)
	if localErr == nil {
		return cfg, nil
	}
	cfg, redisErr := b.redisCache.Get(ctx, id)
	if redisErr == nil {
		// 刷新本地缓存
		lerr := b.localCache.Set(ctx, cfg)
		if lerr != nil {
			b.logger.Error("刷新本地缓存失败", log.Error(lerr), log.Int64("bizID", id))
		}
		return cfg, nil
	}

	// 从数据库获取配置
	c, err := b.dao.GetByID(ctx, id)
	if err != nil {
		return domain.BusinessConfig{}, err
	}
	domainConfig := b.toDomain(c)
	// 刷新本地缓存和 redis 缓存
	lerr := b.localCache.Set(ctx, domainConfig)
	if lerr != nil {
		b.logger.Error("刷新本地缓存失败", log.Error(lerr), log.Int64("bizID", id))
	}
	rerr := b.redisCache.Set(ctx, domainConfig)
	if rerr != nil {
		b.logger.Error("刷新redis缓存失败", log.Error(rerr), log.Int64("bizID", id))
	}
	return domainConfig, nil
}

// DeleteByID 删除业务配置
func (b *businessConfigRepository) DeleteByID(ctx context.Context, id int64) error {
	err := b.dao.DeleteByID(ctx, id)
	if err != nil {
		return err
	}
	err = b.redisCache.Delete(ctx, id)
	if err != nil {
		b.logger.Error("删除redis缓存失败", log.Error(err), log.Int64("bizID", id))
	}
	err = b.localCache.Delete(ctx, id)
	if err != nil {
		b.logger.Error("删除本地缓存失败", log.Error(err), log.Int64("bizID", id))
	}
	return nil
}

func (b *businessConfigRepository) SaveConfig(ctx context.Context, config domain.BusinessConfig) error {
	cfg, err := b.dao.SaveConfig(ctx)
}

func (b *businessConfigRepository) Find(ctx context.Context, offset, limit int) ([]domain.BusinessConfig, error) {
	res, err := b.dao.Find(ctx, offset, limit)
	resList := make([]domain.BusinessConfig, 0, len(res))
	for _, val := range res {
		resList = append(resList, b.toDomain(val))
	}
	return resList, err
}

func (b *businessConfigRepository) loadCacheBatch(ctx context.Context, offset int, limit int) (int, error) {
	res, err := b.Find(ctx, offset, limit)
	if err != nil {
		return 0, err
	}
	err = b.localCache.SetConfigs(ctx, res)
	return len(res), err
}

func (b *businessConfigRepository) toDomain(config dao.BusinessConfig) domain.BusinessConfig {
	domainCfg := domain.BusinessConfig{
		ID:        config.ID,
		OwnerId:   config.OwnerID,
		OwnerType: config.OwnerType,
		RateLimit: config.RateLimit,
		Ctime:     config.Ctime,
		Utime:     config.Utime,
	}
	if config.ChannelConfig.Valid {
		domainCfg.ChannelConfig = &config.ChannelConfig.Val
	}
	if config.TxnConfig.Valid {
		domainCfg.TxnConfig = &config.TxnConfig.Val
	}
	if config.Quota.Valid {
		domainCfg.Quota = &config.Quota.Val
	}
	if config.CallbackConfig.Valid {
		domainCfg.CallbackConfig = &config.CallbackConfig.Val
	}
	return domainCfg
}

func (b *businessConfigRepository) diffIDs(ids []int64, m map[int64]domain.BusinessConfig) []int64 {
	res := make([]int64, 0, len(ids))
	for _, id := range ids {
		if _, ok := m[id]; !ok {
			res = append(res, id)
		}
	}
	return res
}
