package ioc

import (
	"github.com/spf13/viper"
	igrpc "go-notification/internal/api/grpc"
	"go-notification/internal/pkg/grpcx"
	"go-notification/internal/pkg/logger"
	"google.golang.org/grpc"
)

func InitGRPCServer(notifiServer *igrpc.NotificationServer, logger logger.Logger) *grpcx.Server {
	type Config struct {
		Port      int      `yaml:"port"`
		EtcdAddrs []string `yaml:"etcdAddrs"`
	}
	var cfg Config
	err := viper.UnmarshalKey("grpc.server", &cfg)
	if err != nil {
		panic(err)
	}
	server := grpc.NewServer(grpc.ChainUnaryInterceptor())
	notifiServer.Register(server)

	return &grpcx.Server{
		Server:    server,
		Port:      cfg.Port,
		EtcdAddrs: cfg.EtcdAddrs,
		Name:      "notification",
		L:         logger,
	}
}
