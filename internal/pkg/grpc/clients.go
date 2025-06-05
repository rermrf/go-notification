package grpc

import (
	"fmt"
	"google.golang.org/grpc"
	"sync"
)

type Clients[T any] struct {
	clientMap sync.Map
	creator   func(conn *grpc.ClientConn) T
}

func NewClients[T any](creator func(conn *grpc.ClientConn) T) *Clients[T] {
	return &Clients[T]{creator: creator}
}

func (c *Clients[T]) Get(serviceName string) T {
	client, ok := c.clientMap.Load(serviceName)
	if !ok {
		// 初始化 client
		grpcConn, _ := grpc.NewClient(fmt.Sprintf("etcd:///%s", serviceName))
		client = c.creator(grpcConn)
		c.clientMap.Store(serviceName, client)
	}
	return client
}
