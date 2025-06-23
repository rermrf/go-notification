package grpc

import (
	"fmt"
	"google.golang.org/grpc"
	"sync"
)

type Clients[T any] struct {
	clientMap sync.Map // 存储 serviceName -> T
	creator   func(conn *grpc.ClientConn) T
	mu        sync.Mutex // 保护每个serviceName的创建过程
}

func NewClients[T any](creator func(conn *grpc.ClientConn) T) *Clients[T] {
	return &Clients[T]{creator: creator}
}

func (c *Clients[T]) Get(serviceName string) T {
	// 先尝试无锁读取
	if client, ok := c.clientMap.Load(serviceName); ok {
		return client.(T) // 类型断言
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 双检查：获取锁后再次检查
	if client, ok := c.clientMap.Load(serviceName); ok {
		return client.(T)
	}

	// 创建新连接和客户端
	grpcConn, _ := grpc.Dial(fmt.Sprintf("etcd:///%s", serviceName))
	client := c.creator(grpcConn)
	c.clientMap.Store(serviceName, client)
	return client
}
