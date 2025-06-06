package grpc

import (
	"context"
	"errors"
	"fmt"
	notificationv1 "go-notification/api/proto/gen/api/proto/notification/v1"
	"go-notification/internal/api/grpc/interceptor/jwt"
	"go-notification/internal/domain"
	"go-notification/internal/errs"
	notificationSvc "go-notification/internal/service/notification"
	templatesvc "go-notification/internal/service/template/manage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	batchSizeLimit = 100
)

// NotificationServer 通知平台gRPC服务器处理gRPC请求
type NotificationServer struct {
	notificationv1.UnimplementedNotificationServiceServer
	notificationv1.UnimplementedNotificationQueryServiceServer

	notificationSvc notificationSvc.Service
	sendSvc         notificationSvc.SendService
	txnSvc          notificationSvc.TxNotificationService
	templateSvc     templatesvc.ChannelTemplateService
}

func NewNotificationServer(notificationSvc notificationSvc.Service, sendScc notificationSvc.SendService, txnSvc notificationSvc.TxNotificationService, templateSvc templatesvc.ChannelTemplateService) *NotificationServer {
	return &NotificationServer{notificationSvc: notificationSvc, sendSvc: sendScc, txnSvc: txnSvc, templateSvc: templateSvc}
}

// SendNotification 处理同步发送请求
func (n NotificationServer) SendNotification(ctx context.Context, request *notificationv1.SendNotificationRequest) (*notificationv1.SendNotificationResponse, error) {
	response := &notificationv1.SendNotificationResponse{}

	// 从 metadata 中解析 Authorization JWT Token
	bizID, err := jwt.GetBizIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	// 构建领域对象
	notification, err := n.buildNotification(ctx, request.Notification, bizID)
	if err != nil {
		response.ErrorCode = notificationv1.ErrorCode_INVALID_PARAMETER
		response.ErrorMessage = err.Error()
		response.Status = notificationv1.SendStatus_FAILED
		return response, nil
	}

	// 调用发送服务
	result, err := n.sendSvc.SendNotification(ctx, notification)
	if err != nil {
		if n.isSystemError(err) {
			return nil, status.Errorf(codes.Internal, "系统错误: %v", err)
		} else {
			response.ErrorCode = n.convertToGRPCErrorCode(err)
			response.ErrorMessage = err.Error()
			response.Status = notificationv1.SendStatus_FAILED
			return response, nil
		}
	}

	response.NotificationId = uint64(result.NotificationID)
	response.Status = n.covertToGRPCSendStatus(result.Status)
	return response, nil
}

func (n NotificationServer) SendNotificationAsync(ctx context.Context, request *notificationv1.SendNotificationAsyncRequest) (*notificationv1.SendNotificationAsyncResponse, error) {
	response := &notificationv1.SendNotificationAsyncResponse{}

	// 从metadata中解析Authorization JWT Token
	bizID, err := jwt.GetBizIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	// 构建领域对象
	notification, err := n.buildNotification(ctx, request.Notification, bizID)
	if err != nil {
		response.ErrorCode = notificationv1.ErrorCode_INVALID_PARAMETER
		response.ErrorMessage = err.Error()
		return response, nil
	}

	// 执行发送
	result, err := n.sendSvc.SendNotificationAsync(ctx, notification)
	if err != nil {
		// 区分系统错误和业务错误
		if n.isSystemError(err) {
			// 系统错误通过gRPC错误返回
			return nil, status.Errorf(codes.Internal, "%v", err)
		} else {
			// 业务错误通过ErrorCode返回
			response.ErrorCode = n.convertToGRPCErrorCode(err)
			response.ErrorMessage = err.Error()
			return response, nil
		}
	}

	// 将结果转为响应
	response.NotificationId = uint64(result.NotificationID)
	return response, nil
}

// SendNotificationBatch 处理批量同步发送通知请求
func (n NotificationServer) SendNotificationBatch(ctx context.Context, request *notificationv1.SendNotificationBatchRequest) (*notificationv1.SendNotificationBatchResponse, error) {
	// 从metadata中解析Authorization JWT Token
	bizID, err := jwt.GetBizIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	// 处理空请求或者通知列表
	if request == nil || len(request.GetNotifications()) == 0 {
		return &notificationv1.SendNotificationBatchResponse{
			TotalCount:   0,
			SuccessCount: 0,
			Results:      []*notificationv1.SendNotificationResponse{},
		}, nil
	}

	if len(request.GetNotifications()) > batchSizeLimit {
		return nil, status.Errorf(codes.InvalidArgument, "%v: %d > %d", errs.ErrBatchSizeOverLimit, len(request.GetNotifications()), batchSizeLimit)
	}

	var hasError bool
	results := make([]*notificationv1.SendNotificationResponse, len(request.GetNotifications()))
	// 构建领域对象
	notifications := make([]domain.Notification, 0, len(request.GetNotifications()))
	for i := range request.GetNotifications() {
		notification, err := n.buildNotification(ctx, request.Notifications[i], bizID)
		if err != nil {
			results[i] = &notificationv1.SendNotificationResponse{
				ErrorCode:    notificationv1.ErrorCode_INVALID_PARAMETER,
				ErrorMessage: err.Error(),
				Status:       notificationv1.SendStatus_FAILED,
			}
			hasError = true
			continue
		}
		notifications = append(notifications, notification)
	}
	if hasError {
		return &notificationv1.SendNotificationBatchResponse{
			TotalCount:   int32(len(results)),
			SuccessCount: int32(0),
			Results:      results,
		}, nil
	}

	// 执行发送
	responses, err := n.sendSvc.BatchSendNotifications(ctx, notifications)
	if err != nil {
		if n.isSystemError(err) {
			return nil, status.Errorf(codes.Internal, "%v", err)
		} else {
			for i := range results {
				results[i] = &notificationv1.SendNotificationResponse{
					ErrorCode:    n.convertToGRPCErrorCode(err),
					ErrorMessage: err.Error(),
					Status:       notificationv1.SendStatus_FAILED,
				}
			}
			return &notificationv1.SendNotificationBatchResponse{
				TotalCount:   int32(len(results)),
				SuccessCount: int32(0),
				Results:      results,
			}, nil
		}
	}

	// 将结果转为响应
	successCount := int32(0)
	for i := range responses.Results {
		results[i] = n.buildGRPCSendResponse(responses.Results[i], nil)
		if notifications[0].SendStrategyConfig.Type == domain.SendStrategyImmediate && domain.SendStatusSending == responses.Results[i].Status {
			successCount++
		}
		if notifications[0].SendStrategyConfig.Type != domain.SendStrategyImmediate && responses.Results[i].Status == domain.SendStatusSending {
			successCount++
		}
	}
	return &notificationv1.SendNotificationBatchResponse{
		TotalCount:   int32(len(results)),
		SuccessCount: successCount,
		Results:      results,
	}, nil
}

// SendNotificationBatchAsync 处理批量异步发送
func (n NotificationServer) SendNotificationBatchAsync(ctx context.Context, request *notificationv1.SendNotificationBatchAsyncRequest) (*notificationv1.SendNotificationBatchAsyncResponse, error) {
	// 从metadata中解析Authorization JWT Token
	bizID, err := jwt.GetBizIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	// 处理空请求或者通知列表
	if request == nil || len(request.GetNotifications()) == 0 {
		return &notificationv1.SendNotificationBatchAsyncResponse{
			NotificationIds: []uint64{},
		}, nil
	}

	if len(request.GetNotifications()) > batchSizeLimit {
		return nil, status.Errorf(codes.InvalidArgument, "%v: %d > %d", errs.ErrBatchSizeOverLimit, len(request.GetNotifications()), batchSizeLimit)
	}

	// 构建领域对象
	notifications := make([]domain.Notification, 0, len(request.GetNotifications()))
	for i := range request.GetNotifications() {
		notification, err := n.buildNotification(ctx, request.Notifications[i], bizID)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "%v: %#v", err, request.Notifications[i])
		}
		notifications = append(notifications, notification)
	}

	// 执行发送
	result, err := n.sendSvc.BatchSendNotificationsAsync(ctx, notifications)
	if err != nil {
		if n.isSystemError(err) {
			return nil, status.Errorf(codes.Internal, "%v", err)
		} else {
			return nil, status.Errorf(codes.InvalidArgument, "批量异步发送失败：%v", err)
		}
	}

	// 将结果转换为响应
	return &notificationv1.SendNotificationBatchAsyncResponse{
		NotificationIds: result.NotificationIDs,
	}, nil
}

// PrepareTx 处理事务通知准备请求
func (n NotificationServer) PrepareTx(ctx context.Context, request *notificationv1.PrepareTxRequest) (*notificationv1.PrepareTxResponse, error) {
	// 从metadata中解析Authorization JWT Token
	bizID, err := jwt.GetBizIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	// 构建领域对象
	txn, err := n.buildNotification(ctx, request.Notification, bizID)
	
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

func (n NotificationServer) buildNotification(ctx context.Context, noti *notificationv1.Notification, bizID int64) (domain.Notification, error) {
	notification, err := domain.NewNotificationFromAPI(noti)
	if err != nil {
		return domain.Notification{}, err
	}

	tmpl, err := n.templateSvc.GetTemplateByID(ctx, notification.Template.ID)
	if err != nil {
		return domain.Notification{}, fmt.Errorf("%w: 模板ID: %s", errs.ErrInvalidParameter, noti.TemplateId)
	}

	if !tmpl.HasPublished() {
		return domain.Notification{}, fmt.Errorf("%w: 模板ID: %s 未发布", errs.ErrInvalidParameter, noti.TemplateId)
	}

	notification.BizID = bizID
	notification.Template.VersionID = tmpl.ActiveVersionID
	return notification, nil
}

func (n NotificationServer) isSystemError(err error) bool {
	return errors.Is(err, errs.ErrDatabaseError) ||
		errors.Is(err, errs.ErrExternalServiceError) ||
		errors.Is(err, errs.ErrNotificationDuplicate) ||
		errors.Is(err, errs.ErrNotificationVersionMismatch)
}

func (n NotificationServer) convertToGRPCErrorCode(err error) notificationv1.ErrorCode {
	// 注意：这个函数只处理业务错误，系统错误由isSystemError判断后直接通过gRPC status返回
	switch {
	case errors.Is(err, errs.ErrInvalidParameter):
		return notificationv1.ErrorCode_INVALID_PARAMETER

	case errors.Is(err, errs.ErrTemplateNotFound):
		return notificationv1.ErrorCode_TEMPLATE_NOT_FOUND

	case errors.Is(err, errs.ErrChannelDisabled):
		return notificationv1.ErrorCode_CHANNEL_DISABLED

	case errors.Is(err, errs.ErrRateLimited):
		return notificationv1.ErrorCode_RATE_LIMITED

	case errors.Is(err, errs.ErrBizIDNotFound):
		return notificationv1.ErrorCode_BIZ_ID_NOT_FOUND

	case errors.Is(err, errs.ErrSendNotificationFailed):
		return notificationv1.ErrorCode_SEND_NOTIFICATION_FAILED

	case errors.Is(err, errs.ErrCreateNotificationFailed):
		return notificationv1.ErrorCode_CREATE_NOTIFICATION_FAILED

	case errors.Is(err, errs.ErrNotificationNotFound):
		return notificationv1.ErrorCode_NOTIFICATION_NOT_FOUND

	case errors.Is(err, errs.ErrNoAvailableProvider):
		return notificationv1.ErrorCode_NO_AVAILABLE_PROVIDER

	case errors.Is(err, errs.ErrNoAvailableChannel):
		return notificationv1.ErrorCode_NO_AVAILABLE_CHANNEL

	case errors.Is(err, errs.ErrConfigNotFound):
		return notificationv1.ErrorCode_CONFIG_NOT_FOUND

	case errors.Is(err, errs.ErrNoQuotaConfig):
		return notificationv1.ErrorCode_NO_QUOTA_CONFIG

	case errors.Is(err, errs.ErrNoQuota):
		return notificationv1.ErrorCode_NO_QUOTA

	case errors.Is(err, errs.ErrQuotaNotFound):
		return notificationv1.ErrorCode_QUOTA_NOT_FOUND

	case errors.Is(err, errs.ErrProviderNotFound):
		return notificationv1.ErrorCode_PROVIDER_NOT_FOUND

	case errors.Is(err, errs.ErrUnknownChannel):
		return notificationv1.ErrorCode_UNKNOWN_CHANNEL

	default:
		return notificationv1.ErrorCode_ERROR_CODE_UNSPECIFIED
	}
}

// covertToGRPCSendStatus 将领域层的发送状态转换为gRPC层的发送状态
func (n NotificationServer) covertToGRPCSendStatus(status domain.SendStatus) notificationv1.SendStatus {
	switch status {
	case domain.SendStatusPrepare:
		return notificationv1.SendStatus_PREPARE
	case domain.SendStatusCanceled:
		return notificationv1.SendStatus_CANCELED
	case domain.SendStatusSending:
		return notificationv1.SendStatus_PENDING
	case domain.SendStatusSucceeded:
		return notificationv1.SendStatus_SUCCEEDED
	case domain.SendStatusFailed:
		return notificationv1.SendStatus_FAILED
	default:
		return notificationv1.SendStatus_SEND_STATUS_UNSPECIFIED
	}
}

// buildGRPCSendResponse 将领域响应转换为gRPC响应
func (n NotificationServer) buildGRPCSendResponse(res domain.SendResponse, err error) *notificationv1.SendNotificationResponse {
	response := &notificationv1.SendNotificationResponse{
		NotificationId: uint64(res.NotificationID),
		Status:         n.covertToGRPCSendStatus(res.Status),
	}
	// 如果有错误，提取错误代码和消息
	if err != nil {
		response.ErrorCode = n.convertToGRPCErrorCode(err)
		response.ErrorMessage = err.Error()

		// 如果状态不是失败，但是有错误，更新状态为失败
		if response.Status != notificationv1.SendStatus_FAILED {
			response.Status = notificationv1.SendStatus_FAILED
		}
	}
	return response
}
