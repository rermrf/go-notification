package ginx

import (
	"errors"
	"github.com/gin-gonic/gin"
	"log/slog"
	"net/http"
)

// W 不需要请求参数的wrapper函数
func W(fn func(ctx *gin.Context) (Result, error)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		res, err := fn(ctx)
		if errors.Is(err, ErrNoResponse) {
			slog.Debug("不需要响应", slog.Any("err", err))
			return
		}
		if errors.Is(err, ErrUnauthorized) {
			slog.Debug("未授权", slog.Any("err", err))
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		if err != nil {
			slog.Error("执行业务逻辑失败", slog.Any("err", err))
			ctx.PureJSON(http.StatusInternalServerError, res)
			return
		}
		ctx.PureJSON(http.StatusOK, res)
	}
}

// B 需求请求参数的包裹函数
func B[Req any](fn func(ctx *gin.Context, req Req) (Result, error)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req Req
		if err := ctx.Bind(&req); err != nil {
			slog.Debug("绑定参数失败", slog.Any("err", err))
			return
		}
		res, err := fn(ctx, req)
		if errors.Is(err, ErrNoResponse) {
			slog.Debug("不需要响应", slog.Any("err", err))
			return
		}
		if errors.Is(err, ErrUnauthorized) {
			slog.Debug("未授权", slog.Any("err", err))
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		if err != nil {
			slog.Error("执行业务逻辑失败", slog.Any("err", err))
			ctx.PureJSON(http.StatusInternalServerError, res)
			return
		}
		ctx.PureJSON(http.StatusOK, res)
	}
}
