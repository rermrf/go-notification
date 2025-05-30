package limit

import (
	"context"
	configv1 "go-notification/api/proto/gen/api/proto/config/v1"
)

type ConfigServer struct {
}

func (c ConfigServer) GetByIDs(ctx context.Context, request *configv1.GetByIDsRequest) (*configv1.GetByIDsResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (c ConfigServer) GetByID(ctx context.Context, request *configv1.GetByIDRequest) (*configv1.GetByIDResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (c ConfigServer) Delete(ctx context.Context, request *configv1.DeleteRequest) (*configv1.DeleteResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (c ConfigServer) SaveConfig(ctx context.Context, request *configv1.SaveConfigRequest) (*configv1.SaveConfigResponse, error) {
	//TODO implement me
	panic("implement me")
}
