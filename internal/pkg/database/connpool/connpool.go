package connpool

import (
	"context"
	"database/sql"
	"fmt"
	"go-notification/internal/errs"
	"go-notification/internal/event/failover"
	"go-notification/internal/pkg/database/monitor"
	"go-notification/internal/pkg/logger"
	"gorm.io/gorm"
)

type DBWithFailOver struct {
	db        gorm.ConnPool
	log       logger.Logger
	dbMonitor monitor.DBMonitor
	producker failover.ConnPoolEventProducer
}

func NewDBWithFailOver(db *sql.DB, log logger.Logger, dbMonitor monitor.DBMonitor, producker failover.ConnPoolEventProducer) *DBWithFailOver {
	return &DBWithFailOver{db: db, log: log, dbMonitor: dbMonitor, producker: producker}
}

func (d DBWithFailOver) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	if !d.dbMonitor.Health() {
		return nil, errs.ErrDatabaseError
	}
	return d.db.PrepareContext(ctx, query)
}

func (d DBWithFailOver) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	// 如果不健康，则转到异步处理
	if !d.dbMonitor.Health() {
		err := d.producker.Produce(ctx, failover.ConnPoolEvent{
			SQL:  query,
			Args: args,
		})
		if err != nil {
			return nil, fmt.Errorf("数据库有问题转异步失败")
		}
		// 通过 ErrToAsync 代表这边已经转异步了
		return nil, errs.ErrToAsync
	}
	return d.db.ExecContext(ctx, query, args...)
}

func (d DBWithFailOver) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if !d.dbMonitor.Health() {
		return nil, errs.ErrDatabaseError
	}
	return d.db.QueryContext(ctx, query, args...)
}

func (d DBWithFailOver) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	//if !d.dbMonitor.Health() {
	//	// 可以考虑直接拒绝
	//	return &sql.Row{
	//		// 私有字段，要考虑使用 unsafe 来赋值
	//		err: errs.ErrDatabaseError,
	//	}
	//}
	return d.db.QueryRowContext(ctx, query, args...)
}
