package wallet

import (
	"context"
	"database/sql"
	"errors"
	"time"

	common "github.com/paper-trade-chatbot/be-common"
	"github.com/paper-trade-chatbot/be-proto/wallet"
	"github.com/paper-trade-chatbot/be-order/dao/transactionRecordDao"
	"github.com/paper-trade-chatbot/be-order/dao/walletDao"
	"github.com/paper-trade-chatbot/be-order/database"
	"github.com/paper-trade-chatbot/be-order/logging"
	"github.com/paper-trade-chatbot/be-order/models/dbModels"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type WalletIntf interface {
	CreateWallet(ctx context.Context, in *wallet.CreateWalletReq) (*wallet.CreateWalletRes, error)
	GetWallets(ctx context.Context, in *wallet.GetWalletsReq) (*wallet.GetWalletsRes, error)
	DeleteWallet(ctx context.Context, in *wallet.DeleteWalletReq) (*wallet.DeleteWalletRes, error)
	Transaction(ctx context.Context, in *wallet.TransactionReq) (*wallet.TransactionRes, error)
	RollbackTransaction(ctx context.Context, in *wallet.RollbackTransactionReq) (*wallet.RollbackTransactionRes, error)
	GetTransactionRecord(ctx context.Context, in *wallet.GetTransactionRecordReq) (*wallet.GetTransactionRecordRes, error)
	GetTransactionRecords(ctx context.Context, in *wallet.GetTransactionRecordsReq) (*wallet.GetTransactionRecordsRes, error)
}

type WalletImpl struct {
	WalletClient wallet.WalletServiceClient
}

func New() WalletIntf {
	return &WalletImpl{}
}

func (impl *WalletImpl) CreateWallet(ctx context.Context, in *wallet.CreateWalletReq) (*wallet.CreateWalletRes, error) {
	db := database.GetDB()
	model := &dbModels.WalletModel{
		MemberID: in.MemberID,
		Currency: in.Currency,
		Amount:   decimal.Zero,
	}
	if _, err := walletDao.New(db, model); err != nil {
		logging.Error(ctx, "[CreateWallet] failed to new wallet: %v", err)
		return nil, err
	}
	return &wallet.CreateWalletRes{
		WalletID: model.ID,
	}, nil
}

func (impl *WalletImpl) GetWallets(ctx context.Context, in *wallet.GetWalletsReq) (*wallet.GetWalletsRes, error) {
	db := database.GetDB()
	query := &walletDao.QueryModel{}
	switch w := in.Wallet.(type) {
	case *wallet.GetWalletsReq_Id:
		query.ID = []uint64{w.Id}
	case *wallet.GetWalletsReq_MemberID:
		query.MemberID = []uint64{w.MemberID}
		if in.Currency != nil {
			query.Currency = in.Currency
		}
	}

	models, err := walletDao.Gets(db, query)
	if err != nil {
		return nil, err
	}

	wallets := []*wallet.Wallet{}
	for _, m := range models {
		w := &wallet.Wallet{
			Id:        m.ID,
			MemberID:  m.MemberID,
			Amount:    m.Amount.String(),
			Currency:  m.Currency,
			CreatedAt: m.CreatedAt.Unix(),
			UpdatedAt: m.CreatedAt.Unix(),
		}

		wallets = append(wallets, w)
	}

	return &wallet.GetWalletsRes{
		Wallets: wallets,
	}, nil
}

func (impl *WalletImpl) DeleteWallet(ctx context.Context, in *wallet.DeleteWalletReq) (*wallet.DeleteWalletRes, error) {
	db := database.GetDB()
	query := &walletDao.QueryModel{
		ID: []uint64{in.Id},
	}

	if err := walletDao.Delete(db, query); err != nil {
		logging.Error(ctx, "[DeleteWallet] failed to delete wallet: %v", err)
		return nil, err
	}

	return &wallet.DeleteWalletRes{}, nil
}

func (impl *WalletImpl) Transaction(ctx context.Context, in *wallet.TransactionReq) (*wallet.TransactionRes, error) {

	db := database.GetDB()
	amount, err := decimal.NewFromString(in.Amount)
	if err != nil {
		logging.Error(ctx, "[Transaction] failed to cast amount to decimal: %v", err)
		return nil, err
	}

	walletModel, err := walletDao.Get(db, &walletDao.QueryModel{
		ID: []uint64{in.WalletID},
	})
	if err != nil {
		logging.Error(ctx, "[Transaction] failed to get wallet %d: %v", in.WalletID, err)
		return nil, err
	}

	transactionRecord := &dbModels.TransactionRecordModel{
		MemberID:    walletModel.MemberID,
		WalletID:    in.WalletID,
		Action:      dbModels.TransactionAction(in.Action),
		Amount:      amount,
		Currency:    in.Currency,
		CommitterID: in.CommitterID,
		Status:      dbModels.TransactionStatus_Pending,
	}

	if in.Remark != nil {
		transactionRecord.Remark = sql.NullString{
			Valid:  true,
			String: *in.Remark,
		}
	}

	if _, err := transactionRecordDao.New(db, transactionRecord); err != nil {
		logging.Error(ctx, "[Transaction] failed to new transaction record: %v", err)
		return nil, err
	}

	success := false
	retryCount := 0

	beforeAmount := decimal.NewNullDecimal(decimal.Zero)
	afterAmount := decimal.NewNullDecimal(decimal.Zero)

	db = db.Begin()
	for !success {
		if retryCount > 10 {
			logging.Error(ctx, "[Transaction] failed to transaction %d: %v", in.WalletID, common.ErrUpdateWalletInterrupted)

			db = db.Rollback()
			status := dbModels.TransactionStatus_Failed
			if err := transactionRecordDao.Modify(db, transactionRecord, &transactionRecordDao.UpdateModel{
				Status: &status,
			}); err != nil {
				logging.Error(ctx, "[Transaction] failed to modify transaction record: %v", err)
			}
			return nil, common.ErrUpdateWalletInterrupted
		}
		retryCount++

		walletModel, err := walletDao.Get(db, &walletDao.QueryModel{
			ID: []uint64{in.WalletID},
		})
		if err != nil {
			logging.Error(ctx, "[Transaction] failed to get wallet %d: %v", in.WalletID, err)

			db = db.Rollback()
			status := dbModels.TransactionStatus_Failed
			if err := transactionRecordDao.Modify(db, transactionRecord, &transactionRecordDao.UpdateModel{
				Status: &status,
			}); err != nil {
				logging.Error(ctx, "[Transaction] failed to modify transaction record: %v", err)
			}
			return nil, err
		}
		if walletModel == nil {
			logging.Error(ctx, "[Transaction] no such wallet %d: %v", in.WalletID, common.ErrNoSuchWallet)

			db = db.Rollback()
			status := dbModels.TransactionStatus_Failed
			if err := transactionRecordDao.Modify(db, transactionRecord, &transactionRecordDao.UpdateModel{
				Status: &status,
			}); err != nil {
				logging.Error(ctx, "[Transaction] failed to modify transaction record: %v", err)
			}
			return nil, common.ErrNoSuchWallet
		}

		beforeAmount.Decimal = walletModel.Amount
		afterAmount.Decimal = walletModel.Amount.Copy().Add(amount)
		update := &walletDao.UpdateModel{
			Amount: &afterAmount.Decimal,
		}

		if afterAmount.Decimal.LessThan(decimal.Zero) {
			return nil, common.ErrInsufficientBalance
		}

		err = walletDao.Modify(db, walletModel, update)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logging.Debug(ctx, "[Transaction] wallet been modified when updating %d: %v", in.WalletID, err)
			continue
		}
		if err != nil {
			logging.Error(ctx, "[Transaction] failed to update wallet %d: %v", in.WalletID, err)

			db = db.Rollback()
			status := dbModels.TransactionStatus_Failed
			if err := transactionRecordDao.Modify(db, transactionRecord, &transactionRecordDao.UpdateModel{
				Status: &status,
			}); err != nil {
				logging.Error(ctx, "[Transaction] failed to modify transaction record: %v", err)
			}
			return nil, err
		}

		success = true
	}

	status := dbModels.TransactionStatus_Success
	if err := transactionRecordDao.Modify(db, transactionRecord, &transactionRecordDao.UpdateModel{
		BeforeAmount: &beforeAmount,
		AfterAmount:  &afterAmount,
		Status:       &status,
	}); err != nil {
		db = db.Rollback()
		logging.Error(ctx, "[Transaction] failed to modify transaction record: %v", err)
		return nil, err
	}
	db = db.Commit()
	db = database.GetDB()

	transactionRecord, err = transactionRecordDao.Get(db, &transactionRecordDao.QueryModel{
		ID: &transactionRecord.ID,
	})
	if err != nil {
		logging.Error(ctx, "[Transaction] failed to get modified transaction record: %v", err)
		return nil, err
	}

	return &wallet.TransactionRes{
		Id:           transactionRecord.ID,
		BeforeAmount: beforeAmount.Decimal.String(),
		AfterAmount:  afterAmount.Decimal.String(),
		Currency:     in.Currency,
		Status:       wallet.Status(dbModels.TransactionStatus_Success),
		CreatedAt:    transactionRecord.CreatedAt.Unix(),
		UpdatedAt:    transactionRecord.UpdatedAt.Unix(),
	}, nil
}

func (impl *WalletImpl) RollbackTransaction(ctx context.Context, in *wallet.RollbackTransactionReq) (*wallet.RollbackTransactionRes, error) {

	db := database.GetDB()

	record, err := transactionRecordDao.Get(db, &transactionRecordDao.QueryModel{
		ID: &in.Id,
	})
	if err != nil {
		logging.Error(ctx, "[RollbackTransaction] failed to get transaction record: %v", err)
		return nil, err
	}
	if record == nil {
		logging.Error(ctx, "[RollbackTransaction] no such transaction record: %v", common.ErrNoSuchTransactionRecord)
		return nil, common.ErrNoSuchTransactionRecord
	}
	if record.Status != dbModels.TransactionStatus_Success {
		logging.Error(ctx, "[RollbackTransaction] this transaction is not successful: %v", common.ErrTransactionNotSuccess)
		return nil, common.ErrTransactionNotSuccess
	}

	success := false
	retryCount := 0

	beforeAmount := decimal.NewNullDecimal(decimal.Zero)
	afterAmount := decimal.NewNullDecimal(decimal.Zero)

	db = db.Begin()
	for !success {
		if retryCount > 10 {
			logging.Error(ctx, "[RollbackTransaction] failed to transaction %d: %v", record.WalletID, common.ErrUpdateWalletInterrupted)
			return nil, common.ErrUpdateWalletInterrupted
		}
		retryCount++

		walletModel, err := walletDao.Get(db, &walletDao.QueryModel{
			ID: []uint64{record.WalletID},
		})
		if err != nil {
			logging.Error(ctx, "[RollbackTransaction] failed to get wallet %d: %v", record.WalletID, err)
			db = db.Rollback()
			return nil, err
		}
		if walletModel == nil {
			logging.Error(ctx, "[RollbackTransaction] no such wallet %d: %v", record.WalletID, common.ErrNoSuchWallet)
			db = db.Rollback()
			return nil, common.ErrNoSuchWallet
		}

		beforeAmount.Decimal = walletModel.Amount
		afterAmount.Decimal = walletModel.Amount.Add(record.Amount.Neg())
		update := &walletDao.UpdateModel{
			Amount: &afterAmount.Decimal,
		}

		err = walletDao.Modify(db, walletModel, update)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logging.Debug(ctx, "[RollbackTransaction] wallet been modified when updating %d: %v", record.WalletID, err)
			continue
		}
		if err != nil {
			logging.Error(ctx, "[RollbackTransaction] failed to update wallet %d: %v", record.WalletID, err)
			db = db.Rollback()
			return nil, err
		}

		success = true
	}

	status := dbModels.TransactionStatus_Rollback
	rollbackerID := sql.NullInt64{
		Int64: int64(in.RollbackerID),
		Valid: true,
	}
	remark := &sql.NullString{
		Valid: true,
	}
	if in.Remark != nil {
		remark.String = *in.Remark
	} else {
		remark = nil
	}
	if err := transactionRecordDao.Modify(db, record, &transactionRecordDao.UpdateModel{
		RollbackBeforeAmount: &beforeAmount,
		RollbackAfterAmount:  &afterAmount,
		Status:               &status,
		RollbackerID:         &rollbackerID,
		Remark:               remark,
	}); err != nil {
		logging.Error(ctx, "[Transaction] failed to modify transaction record: %v", err)
		db = db.Rollback()
		return nil, err
	}
	db = db.Commit()

	return &wallet.RollbackTransactionRes{}, nil
}

func (impl *WalletImpl) GetTransactionRecord(ctx context.Context, in *wallet.GetTransactionRecordReq) (*wallet.GetTransactionRecordRes, error) {

	db := database.GetDB()
	query := &transactionRecordDao.QueryModel{
		ID: &in.Id,
	}

	model, err := transactionRecordDao.Get(db, query)
	if err != nil {
		return nil, err
	}

	if model == nil {
		return &wallet.GetTransactionRecordRes{}, nil
	}

	transactionRecord := &wallet.TransactionRecord{
		Id:          model.ID,
		MemberID:    model.MemberID,
		WalletID:    model.WalletID,
		Action:      wallet.Action(model.Action),
		Amount:      model.Amount.String(),
		Currency:    model.Currency,
		CommitterID: model.CommitterID,
		Status:      wallet.Status(model.Status),
		CreatedAt:   model.CreatedAt.Unix(),
		UpdatedAt:   model.UpdatedAt.Unix(),
	}

	if model.Remark.Valid {
		transactionRecord.Remark = &model.Remark.String
	}

	if model.BeforeAmount.Valid {
		beforeAmount := model.BeforeAmount.Decimal.String()
		transactionRecord.BeforeAmount = &beforeAmount
	}
	if model.AfterAmount.Valid {
		afterAmount := model.AfterAmount.Decimal.String()
		transactionRecord.AfterAmount = &afterAmount
	}
	if model.RollbackBeforeAmount.Valid {
		rollbackBeforeAmount := model.RollbackBeforeAmount.Decimal.String()
		transactionRecord.RollbackBeforeAmount = &rollbackBeforeAmount
	}
	if model.RollbackAfterAmount.Valid {
		rollbackAfterAmount := model.RollbackAfterAmount.Decimal.String()
		transactionRecord.RollbackAfterAmount = &rollbackAfterAmount
	}
	if model.RollbackerID.Valid {
		rollbackerID := uint64(model.RollbackerID.Int64)
		transactionRecord.RollbackerID = &rollbackerID
	}

	return &wallet.GetTransactionRecordRes{
		Record: transactionRecord,
	}, nil
}

func (impl *WalletImpl) GetTransactionRecords(ctx context.Context, in *wallet.GetTransactionRecordsReq) (*wallet.GetTransactionRecordsRes, error) {

	db := database.GetDB()

	query := &transactionRecordDao.QueryModel{
		MemberID:     in.MemberID,
		CommitterID:  in.CommitterID,
		RollbackerID: in.RollbackerID,
		Currency:     in.Currency,
	}

	for _, a := range in.Action {
		query.Action = append(query.Action, dbModels.TransactionAction(a))
	}
	for _, s := range in.Status {
		query.Status = append(query.Status, dbModels.TransactionStatus(s))
	}
	if in.CreatedFrom != nil {
		createdFrom := time.Unix(*in.CreatedFrom, 10)
		query.CreatedFrom = &createdFrom
	}
	if in.CreatedTo != nil {
		createdTo := time.Unix(*in.CreatedTo, 10)
		query.CreatedTo = &createdTo
	}
	for _, o := range in.Order {
		query.OrderBy = append(query.OrderBy, &transactionRecordDao.Order{
			Column:    transactionRecordDao.OrderColumn(o.OrderBy),
			Direction: transactionRecordDao.OrderDirection(o.OrderDirection),
		})
	}

	models, paginationInfo, err := transactionRecordDao.GetsWithPagination(db, query, in.Pagination)
	if err != nil {
		return nil, err
	}

	if len(models) == 0 {
		return &wallet.GetTransactionRecordsRes{
			PaginationInfo: paginationInfo,
		}, nil
	}

	res := &wallet.GetTransactionRecordsRes{
		PaginationInfo: paginationInfo,
	}
	for _, m := range models {
		transactionRecord := &wallet.TransactionRecord{
			Id:          m.ID,
			MemberID:    m.MemberID,
			WalletID:    m.WalletID,
			Action:      wallet.Action(m.Action),
			Amount:      m.Amount.String(),
			Currency:    m.Currency,
			CommitterID: m.CommitterID,
			Status:      wallet.Status(m.Status),
			CreatedAt:   m.CreatedAt.Unix(),
			UpdatedAt:   m.UpdatedAt.Unix(),
		}
		if m.Remark.Valid {
			transactionRecord.Remark = &m.Remark.String
		}
		if m.BeforeAmount.Valid {
			beforeAmount := m.BeforeAmount.Decimal.String()
			transactionRecord.BeforeAmount = &beforeAmount
		}
		if m.AfterAmount.Valid {
			afterAmount := m.AfterAmount.Decimal.String()
			transactionRecord.AfterAmount = &afterAmount
		}
		if m.RollbackBeforeAmount.Valid {
			rollbackBeforeAmount := m.RollbackBeforeAmount.Decimal.String()
			transactionRecord.RollbackBeforeAmount = &rollbackBeforeAmount
		}
		if m.RollbackAfterAmount.Valid {
			rollbackAfterAmount := m.RollbackAfterAmount.Decimal.String()
			transactionRecord.RollbackAfterAmount = &rollbackAfterAmount
		}
		if m.RollbackerID.Valid {
			rollbackerID := uint64(m.RollbackerID.Int64)
			transactionRecord.RollbackerID = &rollbackerID
		}

		res.Records = append(res.Records, transactionRecord)
	}
	return res, nil
}
