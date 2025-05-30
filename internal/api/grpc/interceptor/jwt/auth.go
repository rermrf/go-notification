package jwt

import (
	"context"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"strings"
	"time"
)

const BizIDName = "biz_id"

type InterceptorBuilder struct {
	key string
}

func (b *InterceptorBuilder) Decode(tokenString string) (jwt.MapClaims, error) {
	// 去除可能的 Bearer 前缀（兼容不同客户端实现）
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")

	// 解析 Token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("不支持的签名算法: %v", token.Header["alg"])
		}
		return []byte(b.key), nil
	})
	if err != nil {
		return nil, fmt.Errorf("令牌解析失败: %w", err)
	}
	// 严重 Token 有效性
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, fmt.Errorf("无效令牌")
}

func (b *InterceptorBuilder) Encode(customClaims jwt.MapClaims) (string, error) {
	// 合并自定义声明和默认声明
	claims := jwt.MapClaims{
		"iat": time.Now().Unix(),
		"iss": "notification-platform",
	}

	// 合并用户自定义声明（覆盖默认声明）
	for k, v := range customClaims {
		claims[k] = v
	}

	// 自动处理处理过期时间
	if _, ok := claims["exp"]; !ok {
		claims["exp"] = time.Now().Add(time.Hour * 24).Unix() // 默认24小时过期
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(b.key))
}

func (b *InterceptorBuilder) JwtAuthInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// 1. 提取 metadata
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		// 2. 获取Authorization头
		authHeader := md.Get("authorization")
		if len(authHeader) == 0 {
			return nil, status.Error(codes.Unauthenticated, "authorization token is required")
		}

		// 3. 处理Bearer Token 格式
		tokenString := authHeader[0]
		// 4. 使用现有JwtAuth解码验证
		val, err := b.Decode(tokenString)
		if err != nil {
			// 细化错误类型处理
			if errors.Is(err, jwt.ErrTokenExpired) {
				return nil, status.Error(codes.Unauthenticated, "token is expired")
			}
			if errors.Is(err, jwt.ErrTokenSignatureInvalid) {
				return nil, status.Error(codes.Unauthenticated, "token signature is invalid")
			}
			return nil, status.Error(codes.Unauthenticated, "invalid token"+err.Error())
		}
		v, ok := val[BizIDName]
		if !ok {
			bizId := v.(int64)
			ctx = context.WithValue(ctx, BizIDName, bizId)
		}

		v, ok = val["Priority"]
		if ok {
			ctx = context.WithValue(ctx, "Priority", v)
		}

		return handler(ctx, req)
	}
}

func NewJwtAuth(key string) *InterceptorBuilder {
	return &InterceptorBuilder{
		key: key,
	}
}
