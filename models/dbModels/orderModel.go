package dbModels

import (
	"database/sql"
	"time"

	"github.com/shopspring/decimal"
)

type TransactionType int

const (
	TransactionType_NONE          TransactionType = iota
	TransactionType_OpenPosition                  // 開倉
	TransactionType_ClosePosition                 // 關倉
)

type OrderStatus int

const (
	OrderStatus_None       OrderStatus = iota
	OrderStatus_Pending                // 待處理
	OrderStatus_Failed                 // 失敗
	OrderStatus_Finished               // 完成
	OrderStatus_Cancelled              // 取消
	OrderStatus_Rollbacked             // 回滾
)

type ProductType int

const (
	ProductType_None ProductType = iota
	ProductType_Stock
	ProductType_Crypto
	ProductType_Forex
	ProductType_Futures
)

type TradeType int

const (
	TradeType_None TradeType = iota
	TradeType_Buy
	TradeType_Sell
)

type OrderModel struct {
	ID                  uint64              `gorm:"column:id; primary_key"`
	MemberID            uint64              `gorm:"column:member_id"`
	OrderStatus         OrderStatus         `gorm:"column:order_status"`
	TransactionType     TransactionType     `gorm:"column:transaction_type"`
	ProductType         ProductType         `gorm:"column:product_type"`
	ExchangeCode        string              `gorm:"column:exchange_code"`
	ProductCode         string              `gorm:"column:product_code"`
	TradeType           TradeType           `gorm:"column:trade_type"`
	Amount              decimal.Decimal     `gorm:"column:amount"`
	UnitPrice           decimal.NullDecimal `gorm:"column:unit_price"`
	TransactionRecordID sql.NullInt64       `gorm:"column:transaction_record_id"`
	PositionID          sql.NullInt64       `gorm:"column:position_id"`
	CreatedAt           time.Time           `gorm:"column:created_at"`
	FinishedAt          sql.NullTime        `gorm:"column:finished_at"`
	RollbackerID        sql.NullInt64       `gorm:"column:rollbacker_id"`
	RollbackedAt        sql.NullTime        `gorm:"column:rollbacked_at"`
	Remark              sql.NullString      `gorm:"column:remark"`
	FailCode            sql.NullInt64       `gorm:"column:fail_code"`
}
