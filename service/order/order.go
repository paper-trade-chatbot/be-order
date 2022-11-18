package order

import (
	"context"

	common "github.com/paper-trade-chatbot/be-common"
	"github.com/paper-trade-chatbot/be-order/dao/orderDao"
	"github.com/paper-trade-chatbot/be-order/database"
	"github.com/paper-trade-chatbot/be-order/models/dbModels"
	"github.com/paper-trade-chatbot/be-order/service"
	"github.com/paper-trade-chatbot/be-proto/order"
	"github.com/paper-trade-chatbot/be-proto/product"
	"github.com/shopspring/decimal"
)

type OrderIntf interface {
	StartOpenPositionOrder(ctx context.Context, in *order.StartOpenPositionOrderReq) (*order.StartOpenPositionOrderRes, error)
	FinishOpenPositionOrder(ctx context.Context, in *order.FinishOpenPositionOrderReq) (*order.FinishOpenPositionOrderRes, error)
	StartClosePositionOrder(ctx context.Context, in *order.StartClosePositionOrderReq) (*order.StartClosePositionOrderRes, error)
	FinishClosePositionOrder(ctx context.Context, in *order.FinishClosePositionOrderReq) (*order.FinishClosePositionOrderRes, error)
	FailOrder(ctx context.Context, in *order.FailOrderReq) (*order.FailOrderRes, error)
	RollbackOrder(ctx context.Context, in *order.RollbackOrderReq) (*order.RollbackOrderRes, error)
	GetOrders(ctx context.Context, in *order.GetOrdersReq) (*order.GetOrdersRes, error)
}

type OrderImpl struct {
	OrderClient order.OrderServiceClient
}

func New() OrderIntf {
	return &OrderImpl{}
}

func (impl *OrderImpl) StartOpenPositionOrder(ctx context.Context, in *order.StartOpenPositionOrderReq) (*order.StartOpenPositionOrderRes, error) {
	db := database.GetDB()

	productRes, err := service.Impl.ProductIntf.GetProduct(ctx, &product.GetProductReq{
		Product: &product.GetProductReq_Code{
			Code: &product.ExchangeCodeProductCode{
				ExchangeCode: in.ExchangeCode,
				ProductCode:  in.ProductCode,
			},
		},
	})
	if err != nil {
		return nil, err
	}
	if productRes.Product == nil {
		return nil, common.ErrNoSuchProduct
	}
	amount, err := decimal.NewFromString(in.Amount)
	if err != nil {
		return nil, err
	}

	model := &dbModels.OrderModel{
		MemberID:        in.MemberID,
		OrderStatus:     dbModels.OrderStatus_Pending,
		TransactionType: dbModels.TransactionType_OpenPosition,
		ProductType:     dbModels.ProductType(productRes.Product.Type),
		ExchangeCode:    in.ExchangeCode,
		ProductCode:     in.ProductCode,
		TradeType:       dbModels.TradeType(in.TradeType),
		Amount:          amount,
	}
	if _, err := orderDao.New(db, model); err != nil {
		return nil, err
	}

	return &order.StartOpenPositionOrderRes{
		Id: model.ID,
	}, nil
}

func (impl *OrderImpl) FinishOpenPositionOrder(ctx context.Context, in *order.FinishOpenPositionOrderReq) (*order.FinishOpenPositionOrderRes, error) {
	return nil, common.ErrNotImplemented
}

func (impl *OrderImpl) StartClosePositionOrder(ctx context.Context, in *order.StartClosePositionOrderReq) (*order.StartClosePositionOrderRes, error) {
	return nil, common.ErrNotImplemented
}

func (impl *OrderImpl) FinishClosePositionOrder(ctx context.Context, in *order.FinishClosePositionOrderReq) (*order.FinishClosePositionOrderRes, error) {
	return nil, common.ErrNotImplemented
}

func (impl *OrderImpl) FailOrder(ctx context.Context, in *order.FailOrderReq) (*order.FailOrderRes, error) {
	return nil, common.ErrNotImplemented
}

func (impl *OrderImpl) RollbackOrder(ctx context.Context, in *order.RollbackOrderReq) (*order.RollbackOrderRes, error) {
	return nil, common.ErrNotImplemented
}

func (impl *OrderImpl) GetOrders(ctx context.Context, in *order.GetOrdersReq) (*order.GetOrdersRes, error) {
	return nil, common.ErrNotImplemented
}
