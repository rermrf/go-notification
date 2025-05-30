package grpc

import (
	"context"
	notificationv1 "go-notification/api/proto/gen/api/proto/notification/v1"
	notificationSvc "go-notification/internal/service/notification"
)

const (
	batchSizeLimit = 100
)

// NotificationServer 通知平台gRPC服务器处理gRPC请求
type NotificationServer struct {
	notificationv1.UnimplementedNotificationServiceServer
	notificationv1.UnimplementedNotificationQueryServiceServer

	notificationSvc notificationSvc.Service
	sendScc         notificationSvc.SendService
	txnSvc notificationSvc.
}

func (n NotificationServer) SendNotification(ctx context.Context, request *notificationv1.SendNotificationRequest) (*notificationv1.SendNotificationResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (n NotificationServer) SendNotificationAsync(ctx context.Context, request *notificationv1.SendNotificationAsyncRequest) (*notificationv1.SendNotificationAsyncResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (n NotificationServer) SendNotificationBatch(ctx context.Context, request *notificationv1.SendNotificationBatchRequest) (*notificationv1.SendNotificationBatchResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (n NotificationServer) SendNotificationBatchAsync(ctx context.Context, request *notificationv1.SendNotificationBatchAsyncRequest) (*notificationv1.SendNotificationBatchAsyncResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (n NotificationServer) PrepareTx(ctx context.Context, request *notificationv1.PrepareTxRequest) (*notificationv1.PrepareTxResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (n NotificationServer) CommitTx(ctx context.Context, request *notificationv1.CommitTxRequest) (*notificationv1.CommitTxResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (n NotificationServer) CancelTx(ctx context.Context, request *notificationv1.CancelTxRequest) (*notificationv1.CancelTxResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (n NotificationServer) QueryNotification(ctx context.Context, request *notificationv1.QueryNotificationRequest) (*notificationv1.QueryNotificationResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (n NotificationServer) BatchQueryNotifications(ctx context.Context, request *notificationv1.BatchQueryNotificationsRequest) (*notificationv1.BatchQueryNotificationsResponse, error) {
	//TODO implement me
	panic("implement me")
}
