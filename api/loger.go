package api

import (
	"context"
	"time"

	"github.com/lwmacct/250300-go-mod-mlog/pkg/mlog"
	"gorm.io/gorm/logger"
)

// 定义一个自定义 Logger 结构体
type Logger struct {
	logger.Interface
	LogLevel logger.LogLevel
}

// 实现 LogMode 方法
func (l Logger) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := l
	newLogger.LogLevel = level
	return newLogger
}

// 实现 Info 方法
func (l Logger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Info {
		mlog.Info(mlog.H{"msg": msg, "data": data})
	}
}

// 实现 Warn 方法
func (l Logger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Warn {
		mlog.Warn(mlog.H{"msg": msg, "data": data})
	}
}

// 实现 Error 方法
func (l Logger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Error {
		mlog.Error(mlog.H{"msg": msg, "data": data})
	}
}

// 实现 Trace 方法
func (l Logger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel <= logger.Silent {
		return
	}

	sql, rowsAffected := fc()
	elapsed := time.Since(begin)

	mlog.Trace(mlog.H{
		"msg":          "SQL",
		"data":         sql,
		"rowsAffected": rowsAffected,
		"timeTaken":    elapsed,
		"error":        err,
	})

}
