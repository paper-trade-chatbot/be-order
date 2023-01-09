package position

import (
	"context"

	"github.com/paper-trade-chatbot/be-common/config"
	"github.com/paper-trade-chatbot/be-proto/position"
)

type PositionIntf interface {
	OpenPosition(ctx context.Context, in *position.OpenPositionReq) (*position.OpenPositionRes, error)
	PendingToClosePosition(ctx context.Context, in *position.PendingToClosePositionReq) (*position.PendingToClosePositionRes, error)
	StopPendingPosition(ctx context.Context, in *position.StopPendingPositionReq) (*position.StopPendingPositionRes, error)
	ClosePosition(ctx context.Context, in *position.ClosePositionReq) (*position.ClosePositionRes, error)
	GetPositions(ctx context.Context, in *position.GetPositionsReq) (*position.GetPositionsRes, error)
	ModifyPosition(ctx context.Context, in *position.ModifyPositionReq) (*position.ModifyPositionRes, error)
}

type PositionImpl struct {
	PositionClient position.PositionServiceClient
}

var (
	PositionServiceHost    = config.GetString("POSITION_GRPC_HOST")
	PositionServerGRpcPort = config.GetString("POSITION_GRPC_PORT")
)

func New(positionClient position.PositionServiceClient) PositionIntf {
	return &PositionImpl{
		PositionClient: positionClient,
	}
}

func (impl *PositionImpl) OpenPosition(ctx context.Context, in *position.OpenPositionReq) (*position.OpenPositionRes, error) {
	return impl.PositionClient.OpenPosition(ctx, in)
}

func (impl *PositionImpl) ClosePosition(ctx context.Context, in *position.ClosePositionReq) (*position.ClosePositionRes, error) {
	return impl.PositionClient.ClosePosition(ctx, in)
}

func (impl *PositionImpl) PendingToClosePosition(ctx context.Context, in *position.PendingToClosePositionReq) (*position.PendingToClosePositionRes, error) {
	return impl.PositionClient.PendingToClosePosition(ctx, in)
}

func (impl *PositionImpl) StopPendingPosition(ctx context.Context, in *position.StopPendingPositionReq) (*position.StopPendingPositionRes, error) {
	return impl.PositionClient.StopPendingPosition(ctx, in)
}

func (impl *PositionImpl) GetPositions(ctx context.Context, in *position.GetPositionsReq) (*position.GetPositionsRes, error) {
	return impl.PositionClient.GetPositions(ctx, in)
}

func (impl *PositionImpl) ModifyPosition(ctx context.Context, in *position.ModifyPositionReq) (*position.ModifyPositionRes, error) {
	return impl.PositionClient.ModifyPosition(ctx, in)
}
