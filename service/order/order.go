package order

import (
	"context"
	"database/sql"
	"time"

	common "github.com/paper-trade-chatbot/be-common"
	"github.com/paper-trade-chatbot/be-common/pagination"
	"github.com/paper-trade-chatbot/be-order/cache"
	"github.com/paper-trade-chatbot/be-order/dao/orderDao"
	"github.com/paper-trade-chatbot/be-order/dao/orderProcessDao"
	"github.com/paper-trade-chatbot/be-order/database"
	"github.com/paper-trade-chatbot/be-order/logging"
	"github.com/paper-trade-chatbot/be-order/models/dbModels"
	"github.com/paper-trade-chatbot/be-order/models/redisModels"
	"github.com/paper-trade-chatbot/be-order/pubsub"
	"github.com/paper-trade-chatbot/be-order/service"
	"github.com/paper-trade-chatbot/be-proto/order"
	"github.com/paper-trade-chatbot/be-proto/position"
	"github.com/paper-trade-chatbot/be-proto/product"
	rabbitmqClosePosition "github.com/paper-trade-chatbot/be-pubsub/order/closePosition/rabbitmq"
	rabbitmqOpenPosition "github.com/paper-trade-chatbot/be-pubsub/order/openPosition/rabbitmq"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc/status"
)

type OrderIntf interface {
	StartOpenPositionOrder(ctx context.Context, in *order.StartOpenPositionOrderReq) (*order.StartOpenPositionOrderRes, error)
	FinishOpenPositionOrder(ctx context.Context, in *order.FinishOpenPositionOrderReq) (*order.FinishOpenPositionOrderRes, error)
	StartClosePositionOrder(ctx context.Context, in *order.StartClosePositionOrderReq) (*order.StartClosePositionOrderRes, error)
	FinishClosePositionOrder(ctx context.Context, in *order.FinishClosePositionOrderReq) (*order.FinishClosePositionOrderRes, error)
	FailOrder(ctx context.Context, in *order.FailOrderReq) (*order.FailOrderRes, error)
	RollbackOrder(ctx context.Context, in *order.RollbackOrderReq) (*order.RollbackOrderRes, error)
	GetOrders(ctx context.Context, in *order.GetOrdersReq) (*order.GetOrdersRes, error)
	UpdateOrderProcess(ctx context.Context, in *order.UpdateOrderProcessReq) (*order.UpdateOrderProcessRes, error)
	GetOrderProcess(ctx context.Context, in *order.GetOrderProcessReq) (*order.GetOrderProcessRes, error)
}

type OrderImpl struct {
	OrderClient order.OrderServiceClient
}

func New() OrderIntf {
	return &OrderImpl{}
}

func (impl *OrderImpl) StartOpenPositionOrder(ctx context.Context, in *order.StartOpenPositionOrderReq) (*order.StartOpenPositionOrderRes, error) {
	logging.Info(ctx, "[StartOpenPositionOrder] in: %#v", in)
	db := database.GetDB()
	rds, _ := cache.GetRedis()

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
		logging.Error(ctx, "[StartOpenPositionOrder] GetProduct failed: %v", common.ErrNoSuchProduct)
		return nil, common.ErrNoSuchProduct
	}
	amount, err := decimal.NewFromString(in.Amount)
	if err != nil {
		logging.Error(ctx, "[StartOpenPositionOrder] amount failed: %v", err)
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
		logging.Error(ctx, "[StartOpenPositionOrder] New failed: %v", err)
		return nil, err
	}

	message := &rabbitmqOpenPosition.OpenPositionModel{
		ID:           model.ID,
		MemberID:     in.MemberID,
		ExchangeCode: in.ExchangeCode,
		ProductCode:  in.ProductCode,
		TradeType:    rabbitmqOpenPosition.TradeType(in.TradeType),
		Amount:       amount,
	}

	if _, err = pubsub.GetPublisher[*rabbitmqOpenPosition.OpenPositionModel](ctx).Produce(ctx, message); err != nil {
		logging.Error(ctx, "[StartOpenPositionOrder] error: %v", err)
		failCode := uint64(status.Code(common.ErrUnablePushRabbitmq))
		s, _ := status.FromError(common.ErrUnablePushRabbitmq)
		remark := s.Message()
		if _, err := impl.FailOrder(ctx, &order.FailOrderReq{
			Id:       model.ID,
			FailCode: &failCode,
			Remark:   &remark,
		}); err != nil {
			logging.Error(ctx, "[StartOpenPositionOrder] failed to FailOrder [%d]: %v", model.ID, err)
		}
		return nil, err
	}

	if err = orderProcessDao.Update(ctx, rds, &redisModels.OrderProcessModel{
		OrderID:      model.ID,
		OrderProcess: redisModels.OrderProcess_Waiting,
	}, nil); err != nil {
		logging.Error(ctx, "[StartOpenPositionOrder] failed to Update OrderProcess [%d]: %v", model.ID, err)
	}

	return &order.StartOpenPositionOrderRes{
		Id: model.ID,
	}, nil
}

func (impl *OrderImpl) FinishOpenPositionOrder(ctx context.Context, in *order.FinishOpenPositionOrderReq) (*order.FinishOpenPositionOrderRes, error) {
	logging.Info(ctx, "[FinishOpenPositionOrder] in: %#v", in)
	db := database.GetDB()
	var orderErr error

	defer func() {
		if orderErr != nil {
			failCode := uint64(status.Code(orderErr))
			s, _ := status.FromError(orderErr)
			remark := s.Message()
			if _, err := impl.FailOrder(ctx, &order.FailOrderReq{
				Id:       in.Id,
				FailCode: &failCode,
				Remark:   &remark,
			}); err != nil {
				logging.Error(ctx, "[FinishOpenPositionOrder] failed to FailOrder [%d]: %v", in.Id, err)
			}
		}
	}()

	unitPriceDeciaml, err := decimal.NewFromString(in.UnitPrice)
	if err != nil {
		logging.Error(ctx, "[FinishOpenPositionOrder] NewFromString failed: %v", err)
		orderErr = err
		return nil, err
	}
	unitPrice := decimal.NewNullDecimal(unitPriceDeciaml)

	query := &orderDao.QueryModel{
		ID: []uint64{in.Id},
	}

	model, err := orderDao.Get(db, query)
	if err != nil {
		logging.Error(ctx, "[FinishOpenPositionOrder] Get failed: %v", err)
		orderErr = err
		return nil, err
	}

	if model == nil {
		logging.Error(ctx, "[FinishOpenPositionOrder] Get failed: %v", common.ErrNoSuchOrder)
		orderErr = common.ErrNoSuchOrder
		return nil, common.ErrNoSuchOrder
	}

	positionRes, err := service.Impl.PositionIntf.OpenPosition(ctx, &position.OpenPositionReq{
		MemberID:     model.MemberID,
		Type:         product.ProductType(model.ProductType),
		ExchangeCode: model.ExchangeCode,
		ProductCode:  model.ProductCode,
		TradeType:    position.TradeType(model.TradeType),
		Amount:       model.Amount.String(),
		UnitPrice:    in.UnitPrice,
	})
	if err != nil {
		logging.Error(ctx, "[FinishOpenPositionOrder] OpenPosition failed: %v", err)
		orderErr = common.ErrNoSuchOrder
		return nil, err
	}

	orderStatus := dbModels.OrderStatus_Finished

	pending := dbModels.OrderStatus_Pending
	lock := &orderDao.QueryModel{
		OrderStatus: &pending,
		UnitPrice:   &decimal.NullDecimal{},
	}
	if err = orderDao.Modify(db, model, lock, &orderDao.UpdateModel{
		OrderStatus: &orderStatus,
		UnitPrice:   &unitPrice,
		FinishedAt: &sql.NullTime{
			Valid: true,
			Time:  time.Unix(in.FinishedAt, 0),
		},
		PositionID: &sql.NullInt64{
			Valid: true,
			Int64: int64(positionRes.Id),
		},
	}); err != nil {
		logging.Error(ctx, "[FinishOpenPositionOrder] Modify failed: %v", err)
		orderErr = err
		return nil, err
	}

	return &order.FinishOpenPositionOrderRes{
		PositionID: positionRes.Id,
	}, nil
}

func (impl *OrderImpl) StartClosePositionOrder(ctx context.Context, in *order.StartClosePositionOrderReq) (*order.StartClosePositionOrderRes, error) {
	logging.Info(ctx, "[StartClosePositionOrder] in: %#v", in)
	db := database.GetDB()

	getPositionRes, err := service.Impl.PositionIntf.GetPositions(ctx, &position.GetPositionsReq{
		Id:         []uint64{in.Id},
		Pagination: pagination.NewPagination(10),
	})
	if err != nil {
		logging.Error(ctx, "[StartClosePositionOrder] GetPositions failed: %v", err)
		return nil, err
	}

	if len(getPositionRes.Positions) == 0 {
		logging.Error(ctx, "[StartClosePositionOrder] GetPositions failed: %v", common.ErrNoSuchPosition)
		return nil, common.ErrNoSuchPosition
	}

	amount, err := decimal.NewFromString(in.Amount)
	if err != nil {
		logging.Error(ctx, "[StartClosePositionOrder] amount failed: %v", err)
		return nil, err
	}

	unitPrice, err := decimal.NewFromString(getPositionRes.Positions[0].UnitPrice)
	if err != nil {
		logging.Error(ctx, "[StartClosePositionOrder] unitPrice failed: %v", err)
		return nil, err
	}

	positionRes, err := service.Impl.PositionIntf.PendingToClosePosition(ctx, &position.PendingToClosePositionReq{
		Id:          in.Id,
		CloseAmount: in.Amount,
	})

	if err != nil {
		logging.Error(ctx, "[StartClosePositionOrder] PendingToClosePosition failed: %v", err)
		return nil, err
	}

	if !positionRes.PreemptSuccess {
		logging.Error(ctx, "[StartClosePositionOrder] PendingToClosePosition failed: %v", common.ErrProcessStateNotPending)
		return nil, common.ErrProcessStateNotPending
	}

	model := &dbModels.OrderModel{
		MemberID:        getPositionRes.Positions[0].MemberID,
		OrderStatus:     dbModels.OrderStatus_Pending,
		TransactionType: dbModels.TransactionType_ClosePosition,
		ProductType:     dbModels.ProductType(getPositionRes.Positions[0].Type),
		ExchangeCode:    getPositionRes.Positions[0].ExchangeCode,
		ProductCode:     getPositionRes.Positions[0].ProductCode,
		TradeType:       dbModels.TradeType(getPositionRes.Positions[0].TradeType),
		Amount:          amount,
		PositionID: sql.NullInt64{
			Valid: true,
			Int64: int64(getPositionRes.Positions[0].Id),
		},
	}
	if _, err := orderDao.New(db, model); err != nil {
		logging.Error(ctx, "[StartClosePositionOrder] New failed: %v", err)
		if _, err := service.Impl.PositionIntf.StopPendingPosition(ctx, &position.StopPendingPositionReq{
			Id: in.Id,
		}); err != nil {
			logging.Error(ctx, "[StartClosePositionOrder] StopPendingPosition failed: %v", err)
		}
		return nil, err
	}

	message := &rabbitmqClosePosition.ClosePositionModel{
		ID:           model.ID,
		MemberID:     getPositionRes.Positions[0].MemberID,
		PositionID:   getPositionRes.Positions[0].Id,
		ExchangeCode: getPositionRes.Positions[0].ExchangeCode,
		ProductCode:  getPositionRes.Positions[0].ProductCode,
		TradeType:    rabbitmqClosePosition.TradeType(getPositionRes.Positions[0].TradeType),
		OpenPrice:    unitPrice,
		CloseAmount:  amount,
	}

	if _, err = pubsub.GetPublisher[*rabbitmqClosePosition.ClosePositionModel](ctx).Produce(ctx, message); err != nil {
		logging.Error(ctx, "[StartClosePositionOrder] error: %v", err)
		failCode := uint64(status.Code(common.ErrUnablePushRabbitmq))
		s, _ := status.FromError(common.ErrUnablePushRabbitmq)
		remark := s.Message()
		if _, err := impl.FailOrder(ctx, &order.FailOrderReq{
			Id:       model.ID,
			FailCode: &failCode,
			Remark:   &remark,
		}); err != nil {
			logging.Error(ctx, "[StartClosePositionOrder] failed to FailOrder [%d]: %v", model.ID, err)
		}
		if _, err := service.Impl.PositionIntf.StopPendingPosition(ctx, &position.StopPendingPositionReq{
			Id: in.Id,
		}); err != nil {
			logging.Error(ctx, "[StartClosePositionOrder] StopPendingPosition failed: %v", err)
		}
		return nil, err
	}

	return &order.StartClosePositionOrderRes{
		Id: model.ID,
	}, nil
}

func (impl *OrderImpl) FinishClosePositionOrder(ctx context.Context, in *order.FinishClosePositionOrderReq) (*order.FinishClosePositionOrderRes, error) {
	logging.Info(ctx, "[FinishClosePositionOrder] in: %#v", in)
	db := database.GetDB()
	var orderErr error

	defer func() {
		if orderErr != nil {
			failCode := uint64(status.Code(orderErr))
			s, _ := status.FromError(orderErr)
			remark := s.Message()
			if _, err := impl.FailOrder(ctx, &order.FailOrderReq{
				Id:       in.Id,
				FailCode: &failCode,
				Remark:   &remark,
			}); err != nil {
				logging.Error(ctx, "[FinishOpenPositionOrder] failed to FailOrder [%d]: %v", in.Id, err)
			}
			if _, err := service.Impl.PositionIntf.StopPendingPosition(ctx, &position.StopPendingPositionReq{
				Id: in.PositionID,
			}); err != nil {
				logging.Error(ctx, "[FinishClosePositionOrder] StopPendingPosition failed: %v", err)
			}
		}
	}()

	unitPriceDeciaml, err := decimal.NewFromString(in.UnitPrice)
	if err != nil {
		logging.Error(ctx, "[FinishClosePositionOrder] NewFromString failed: %v", err)
		orderErr = err
		return nil, err
	}
	unitPrice := decimal.NewNullDecimal(unitPriceDeciaml)

	query := &orderDao.QueryModel{
		ID: []uint64{in.Id},
	}

	model, err := orderDao.Get(db, query)
	if err != nil {
		logging.Error(ctx, "[FinishClosePositionOrder] Get failed: %v", err)
		orderErr = err
		return nil, err
	}

	if model == nil {
		logging.Error(ctx, "[FinishClosePositionOrder] Get failed: %v", common.ErrNoSuchOrder)
		orderErr = common.ErrNoSuchOrder
		return nil, common.ErrNoSuchOrder
	}

	_, err = service.Impl.PositionIntf.ClosePosition(ctx, &position.ClosePositionReq{
		Id:          in.PositionID,
		CloseAmount: in.CloseAmount,
	})
	if err != nil {
		logging.Error(ctx, "[FinishClosePositionOrder] ClosePosition failed: %v", err)
		orderErr = err
		return nil, err
	}

	orderStatus := dbModels.OrderStatus_Finished

	pending := dbModels.OrderStatus_Pending
	lock := &orderDao.QueryModel{
		OrderStatus: &pending,
		UnitPrice:   &decimal.NullDecimal{},
	}
	if err = orderDao.Modify(db, model, lock, &orderDao.UpdateModel{
		OrderStatus: &orderStatus,
		UnitPrice:   &unitPrice,
		FinishedAt: &sql.NullTime{
			Valid: true,
			Time:  time.Unix(in.FinishedAt, 0),
		},
		PositionID: &sql.NullInt64{
			Valid: true,
			Int64: int64(in.PositionID),
		},
	}); err != nil {
		logging.Error(ctx, "[FinishClosePositionOrder] Modify failed: %v", err)
		return nil, err
	}

	return &order.FinishClosePositionOrderRes{}, nil
}

func (impl *OrderImpl) FailOrder(ctx context.Context, in *order.FailOrderReq) (*order.FailOrderRes, error) {
	logging.Info(ctx, "[FailOrder] in: %#v", in)
	db := database.GetDB()

	query := &orderDao.QueryModel{
		ID: []uint64{in.Id},
	}

	model, err := orderDao.Get(db, query)
	if err != nil {
		logging.Error(ctx, "[FailOrder] Get failed: %v", err)
		return nil, err
	}

	if model == nil {
		logging.Error(ctx, "[FailOrder] Get failed: %v", common.ErrNoSuchOrder)
		return nil, common.ErrNoSuchOrder
	}

	if model.OrderStatus != dbModels.OrderStatus_Pending {
		return nil, common.ErrOrderNotPending
	}

	pending := dbModels.OrderStatus_Pending
	orderStatus := dbModels.OrderStatus_Failed
	update := &orderDao.UpdateModel{
		OrderStatus: &orderStatus,
	}
	if in.FailCode != nil {
		update.FailCode = &sql.NullInt64{
			Valid: true,
			Int64: int64(*in.FailCode),
		}
	}

	if in.Remark != nil {
		update.Remark = &sql.NullString{
			Valid:  true,
			String: *in.Remark,
		}
	}

	if err = orderDao.Modify(
		db,
		model,
		&orderDao.QueryModel{
			OrderStatus: &pending,
		},
		update,
	); err != nil {
		logging.Error(ctx, "[FailOrder] Modify failed: %v", err)
		return nil, err
	}

	return &order.FailOrderRes{}, nil
}

func (impl *OrderImpl) RollbackOrder(ctx context.Context, in *order.RollbackOrderReq) (*order.RollbackOrderRes, error) {
	return nil, common.ErrNotImplemented
}

func (impl *OrderImpl) GetOrders(ctx context.Context, in *order.GetOrdersReq) (*order.GetOrdersRes, error) {
	logging.Info(ctx, "[GetOrders] in: %#v", in)
	db := database.GetDB()

	query := &orderDao.QueryModel{
		ID:           in.Id,
		MemberID:     in.MemberID,
		ExchangeCode: in.ExchangeCode,
		ProductCode:  in.ProductCode,
	}

	if in.CreatedFrom != nil {
		createdFrom := time.Unix(*in.CreatedFrom, 0)
		query.CreatedFrom = &createdFrom
	}

	if in.CreatedTo != nil {
		createdTo := time.Unix(*in.CreatedTo, 0)
		query.CreatedTo = &createdTo
	}

	if in.TransactionType != nil {
		transactionType := dbModels.TransactionType(*in.TransactionType)
		query.TransactionType = &transactionType
	}

	if in.Type != nil {
		productType := dbModels.ProductType(*in.Type)
		query.ProductType = &productType
	}

	if in.TradeType != nil {
		tradeType := dbModels.TradeType(*in.TradeType)
		query.TradeType = &tradeType
	}

	models, paginationInfo, err := orderDao.GetsWithPagination(db, query, in.Pagination)
	if err != nil {
		logging.Error(ctx, "[FailOrder] GetsWithPagination failed: %v", err)
		return nil, err
	}

	orders := []*order.Order{}

	for _, m := range models {
		o := &order.Order{
			Id:              m.ID,
			MemberID:        m.MemberID,
			OrderStatus:     order.OrderStatus(m.OrderStatus),
			TransactionType: order.TransactionType(m.TransactionType),
			Type:            product.ProductType(m.ProductType),
			ExchangeCode:    m.ExchangeCode,
			ProductCode:     m.ProductCode,
			TradeType:       position.TradeType(m.TradeType),
			Amount:          m.Amount.String(),
			CreatedAt:       m.CreatedAt.Unix(),
		}
		if m.UnitPrice.Valid {
			unitPrice := m.UnitPrice.Decimal.String()
			o.UnitPrice = &unitPrice
		}
		if m.TransactionRecordID.Valid {
			transactionRecordID := uint64(m.TransactionRecordID.Int64)
			o.TransactionRecordID = &transactionRecordID
		}
		if m.FinishedAt.Valid {
			finishedAt := m.FinishedAt.Time.Unix()
			o.FinishedAt = &finishedAt
		}
		if m.RollbackerID.Valid {
			rollbackerID := uint64(m.RollbackerID.Int64)
			o.RollbackerID = &rollbackerID
		}
		if m.RollbackedAt.Valid {
			rollbackedAt := m.RollbackedAt.Time.Unix()
			o.RollbackedAt = &rollbackedAt
		}
		if m.Remark.Valid {
			o.Remark = &m.Remark.String
		}
		if m.FailCode.Valid {
			failCode := uint64(m.FailCode.Int64)
			o.FailCode = &failCode
		}

		orders = append(orders, o)
	}

	return &order.GetOrdersRes{
		Orders:         orders,
		PaginationInfo: paginationInfo,
	}, nil
}

func (impl *OrderImpl) UpdateOrderProcess(ctx context.Context, in *order.UpdateOrderProcessReq) (*order.UpdateOrderProcessRes, error) {
	logging.Info(ctx, "[UpdateOrderProcess] in: %#v", in)
	rds, _ := cache.GetRedis()

	model := &redisModels.OrderProcessModel{
		OrderID:      in.Id,
		OrderProcess: redisModels.OrderProcess(in.OrderProcess),
	}

	var expiration *time.Duration
	if in.Expire != nil {
		expirationInstance := time.Duration(*in.Expire)
		expiration = &expirationInstance
	}

	if err := orderProcessDao.Update(ctx, rds, model, expiration); err != nil {
		logging.Error(ctx, "[UpdateOrderProcess] Update failed: %v", err)
		return nil, err
	}

	return &order.UpdateOrderProcessRes{}, nil
}

func (impl *OrderImpl) GetOrderProcess(ctx context.Context, in *order.GetOrderProcessReq) (*order.GetOrderProcessRes, error) {
	logging.Info(ctx, "[GetOrderProcess] in: %#v", in)
	rds, _ := cache.GetRedis()
	db := database.GetDB()

	query := &orderProcessDao.QueryModel{
		OrderID: in.Id,
	}

	model, err := orderProcessDao.Get(ctx, rds, query)
	if err != nil {
		logging.Error(ctx, "[GetOrderProcess] Get failed: %v", err)
		return nil, err
	}
	if model == nil {
		model, err := orderDao.Get(db, &orderDao.QueryModel{
			ID: []uint64{in.Id},
		})
		if err != nil {
			logging.Error(ctx, "[GetOrderProcess] Get failed: %v", err)
			return nil, err
		}
		if model == nil {
			return nil, common.ErrNoSuchOrder
		}

		if model.OrderStatus == dbModels.OrderStatus_Finished {
			return &order.GetOrderProcessRes{
				OrderProcess: order.OrderProcess_OrderProcess_Finished,
			}, nil
		}

		orderProcess := order.OrderProcess_OrderProcess_Unknown
		if model.OrderStatus == dbModels.OrderStatus_Failed {
			orderProcess = order.OrderProcess_OrderProcess_Failed
		}

		orderProcessModel := &redisModels.OrderProcessModel{
			OrderID:      in.Id,
			OrderProcess: redisModels.OrderProcess(orderProcess),
		}

		if err := orderProcessDao.Update(ctx, rds, orderProcessModel, nil); err != nil {
			logging.Error(ctx, "[GetOrderProcess] Update failed: %v", err)
			return nil, err
		}
		return &order.GetOrderProcessRes{
			OrderProcess: orderProcess,
		}, nil
	}

	return &order.GetOrderProcessRes{
		OrderProcess: order.OrderProcess(model.OrderProcess),
	}, nil
}
