package sharding

import (
	"context"
	"fmt"
	"go-notification/internal/pkg/hash"
	"go-notification/internal/pkg/id_generator"
	"strings"
)

type ShardingStrategy struct {
	dbPrefix      string
	tablePrefix   string
	tableSharding int64
	dbSharding    int64
}

type Dst struct {
	Table string
	DB    string

	TableSuffix int64
	DBSuffix    int64
}

func NewShardingStrategy(dbPrefix string, tablePrefix string, tableSharding int64, dbSharding int64) *ShardingStrategy {
	return &ShardingStrategy{dbPrefix: dbPrefix, tablePrefix: tablePrefix, tableSharding: tableSharding, dbSharding: dbSharding}
}

func (s ShardingStrategy) Shard(bizID int64, key string) Dst {
	hashValue := hash.Hash(bizID, key)
	dbHash := hashValue % s.dbSharding
	tableHash := (hashValue / s.dbSharding) % s.tableSharding
	return Dst{
		TableSuffix: tableHash,
		Table:       fmt.Sprintf("%s_%d", s.tablePrefix, tableHash),
		DBSuffix:    dbHash,
		DB:          fmt.Sprintf("%s_%d", s.dbPrefix, dbHash),
	}
}

func (s ShardingStrategy) ShardWithID(id int64) Dst {
	hashValue := id_generator.ExtractHashValue(id)
	dbHash := hashValue % s.dbSharding
	tableHash := (hashValue / s.dbSharding) % s.tableSharding
	return Dst{
		TableSuffix: tableHash,
		Table:       fmt.Sprintf("%s_%d", s.tablePrefix, tableHash),
		DBSuffix:    dbHash,
		DB:          fmt.Sprintf("%s_%d", s.dbPrefix, dbHash),
	}
}

func (s ShardingStrategy) Broadcast() []Dst {
	// 求笛卡尔积
	ans := make([]Dst, 0, s.tableSharding*s.dbSharding)
	for i := 0; i < int(s.dbSharding); i++ {
		for j := 0; j < int(s.dbSharding); j++ {
			ans = append(ans, Dst{
				TableSuffix: int64(j),
				Table:       fmt.Sprintf("%s_%d", s.tablePrefix, j),
				DBSuffix:    int64(j),
				DB:          fmt.Sprintf("%s_%d", s.dbPrefix, j),
			})
		}
	}
	return ans
}

func (s ShardingStrategy) TablePrefix() string {
	return s.tablePrefix
}

type dstKey struct{}

func DstFromCtx(ctx context.Context) (Dst, bool) {
	val := ctx.Value(dstKey{})
	res, ok := val.(Dst)
	return res, ok
}

func CtxWithDst(ctx context.Context, dst Dst) context.Context {
	return context.WithValue(ctx, dstKey{}, dst)
}

// ExtractSuffixAndFormatFromTable 从表名中提取后缀，按照下划线分隔并返回最后一个元素
func (s ShardingStrategy) ExtractSuffixAndFormatFromTable(tableName string) string {
	parts := strings.Split(tableName, "_")
	suffix := parts[len(parts)-1]
	return fmt.Sprintf("%s_%s", s.tablePrefix, suffix)
}
