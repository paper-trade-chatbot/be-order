package orderDao

import (
	"database/sql"
	"errors"
	"time"

	"github.com/paper-trade-chatbot/be-common/pagination"
	"github.com/paper-trade-chatbot/be-order/models/dbModels"
	"github.com/paper-trade-chatbot/be-proto/general"
	"github.com/shopspring/decimal"

	"gorm.io/gorm"
)

const table = "order"

type OrderColumn int

const (
	OrderColumn_None OrderColumn = iota
	OrderColumn_ProductCode
	OrderColumn_CreatedAt
)

type OrderDirection int

const (
	OrderDirection_None = 0
	OrderDirection_ASC  = 1
	OrderDirection_DESC = -1
)

type Order struct {
	Column    OrderColumn
	Direction OrderDirection
}

// QueryModel set query condition, used by queryChain()
type QueryModel struct {
	ID              []uint64
	MemberID        []uint64
	RollbackerID    []uint64
	OrderStatus     *dbModels.OrderStatus
	TransactionType *dbModels.TransactionType
	ProductType     *dbModels.ProductType
	ExchangeCode    *string
	ProductCode     *string
	TradeType       *dbModels.TradeType
	Amount          *decimal.Decimal
	UnitPrice       *decimal.NullDecimal
	CreatedFrom     *time.Time
	CreatedTo       *time.Time
	OrderBy         []*Order
}

type UpdateModel struct {
	OrderStatus  *dbModels.OrderStatus
	UnitPrice    *decimal.NullDecimal
	RollbackerID *sql.NullInt64
	RollbackedAt *sql.NullTime
	Remark       *sql.NullString
}

// New a row
func New(db *gorm.DB, model *dbModels.OrderModel) (int, error) {

	err := db.Table(table).
		Create(model).Error

	if err != nil {
		return 0, err
	}
	return 1, nil
}

// New rows
func News(db *gorm.DB, m []*dbModels.OrderModel) (int, error) {

	err := db.Transaction(func(tx *gorm.DB) error {

		err := tx.Table(table).
			CreateInBatches(m, 3000).Error

		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return 0, err
	}

	return len(m), nil
}

// Get return a record as raw-data-form
func Get(tx *gorm.DB, query *QueryModel) (*dbModels.OrderModel, error) {

	result := &dbModels.OrderModel{}
	err := tx.Table(table).
		Scopes(queryChain(query)).
		Scan(result).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Gets return records as raw-data-form
func Gets(tx *gorm.DB, query *QueryModel) ([]dbModels.OrderModel, error) {
	result := make([]dbModels.OrderModel, 0)
	err := tx.Table(table).
		Scopes(queryChain(query)).
		Scan(&result).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return []dbModels.OrderModel{}, nil
	}

	if err != nil {
		return nil, err
	}

	return result, nil
}

func GetsWithPagination(tx *gorm.DB, query *QueryModel, paginate *general.Pagination) ([]dbModels.OrderModel, *general.PaginationInfo, error) {

	var rows []dbModels.OrderModel
	var count int64 = 0
	err := tx.Table(table).
		Scopes(queryChain(query)).
		Count(&count).
		Scopes(paginateChain(paginate)).
		Scan(&rows).Error

	offset, _ := pagination.GetOffsetAndLimit(paginate)
	paginationInfo := pagination.SetPaginationDto(paginate.Page, paginate.PageSize, int32(count), int32(offset))

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return []dbModels.OrderModel{}, paginationInfo, nil
	}

	if err != nil {
		return []dbModels.OrderModel{}, nil, err
	}

	return rows, paginationInfo, nil
}

// Gets return records as raw-data-form
func Modify(tx *gorm.DB, model *dbModels.OrderModel, update *UpdateModel) error {
	attrs := map[string]interface{}{}
	if update.OrderStatus != nil {
		attrs["order_status"] = *update.OrderStatus
	}
	if update.UnitPrice != nil {
		attrs["unit_price"] = *update.UnitPrice
	}
	if update.RollbackerID != nil {
		attrs["rollbacker_id"] = *update.RollbackerID
	}
	if update.RollbackedAt != nil {
		attrs["rollbacked_at"] = *update.RollbackedAt
	}
	if update.Remark != nil {
		attrs["remark"] = *update.Remark
	}

	if update.OrderStatus == nil &&
		update.UnitPrice == nil &&
		update.RollbackerID == nil &&
		update.RollbackedAt == nil {
		err := tx.Table(table).
			Model(dbModels.OrderModel{}).
			Where(table+".id = ?", model.ID).
			Updates(attrs).Error
		return err
	}

	if update.OrderStatus != nil &&
		update.UnitPrice != nil &&
		update.RollbackerID != nil &&
		update.RollbackedAt != nil {
		err := tx.Table(table).
			Model(dbModels.OrderModel{}).
			Where(table+".id = ? AND "+table+".order_status = ? AND "+table+".unit_price = ? AND "+table+".rollbacker_id = ? AND "+table+".rollbacked_at = ?",
				model.ID,
				model.OrderStatus,
				model.UnitPrice,
				model.RollbackerID,
				model.RollbackedAt).
			Updates(attrs).Error
		return err
	}

	if update.OrderStatus != nil &&
		update.UnitPrice != nil {
		err := tx.Table(table).
			Model(dbModels.OrderModel{}).
			Where(table+".id = ? AND "+table+".order_status = ? AND "+table+".unit_price = ?",
				model.ID,
				model.OrderStatus,
				model.UnitPrice).
			Updates(attrs).Error
		return err
	}

	if update.RollbackerID != nil &&
		update.RollbackedAt != nil {
		err := tx.Table(table).
			Model(dbModels.OrderModel{}).
			Where(table+".id = ? AND "+table+".rollbacker_id = ? AND "+table+".rollbacked_at = ?",
				model.ID,
				model.RollbackerID,
				model.RollbackedAt).
			Updates(attrs).Error
		return err
	}

	err := tx.Table(table).
		Model(dbModels.OrderModel{}).
		Where(table+".id = ?",
			model.ID,
			model.RollbackerID,
			model.RollbackedAt).
		Updates(attrs).Error

	return err
}

func queryChain(query *QueryModel) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.
			Scopes(idInScope(query.ID)).
			Scopes(memberIDInScope(query.MemberID)).
			Scopes(rollbackerIDInScope(query.RollbackerID)).
			Scopes(orderStatusEqualScope(query.OrderStatus)).
			Scopes(transactionTypeEqualScope(query.TransactionType)).
			Scopes(productTypeEqualScope(query.ProductType)).
			Scopes(exchangeCodeEqualScope(query.ExchangeCode)).
			Scopes(productCodeEqualScope(query.ProductCode)).
			Scopes(tradeTypeEqualScope(query.TradeType)).
			Scopes(createdBetweenScope(query.CreatedFrom, query.CreatedTo)).
			Scopes(orderByScope(query.OrderBy))
	}
}

func paginateChain(paginate *general.Pagination) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		offset, limit := pagination.GetOffsetAndLimit(paginate)
		return db.
			Scopes(offsetScope(offset)).
			Scopes(limitScope(limit))

	}
}

func idInScope(id []uint64) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if len(id) > 0 {
			return db.Where(table+".id IN ?", id)
		}
		return db
	}
}

func memberIDInScope(memberID []uint64) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if len(memberID) > 0 {
			return db.Where(table+".member_id IN ?", memberID)
		}
		return db
	}
}

func rollbackerIDInScope(rollbackerID []uint64) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if len(rollbackerID) > 0 {
			return db.Where(table+".rollbacker_id IN ?", rollbackerID)
		}
		return db
	}
}

func orderStatusEqualScope(orderStatus *dbModels.OrderStatus) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if orderStatus != nil {
			return db.Where(table+".order_status = ?", *orderStatus)
		}
		return db
	}
}

func transactionTypeEqualScope(transactionType *dbModels.TransactionType) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if transactionType != nil {
			return db.Where(table+".transaction_type = ?", *transactionType)
		}
		return db
	}
}

func productTypeEqualScope(productType *dbModels.ProductType) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if productType != nil {
			return db.Where(table+".product_type = ?", *productType)
		}
		return db
	}
}

func exchangeCodeEqualScope(exchangeCode *string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if exchangeCode != nil {
			return db.Where(table+".exchange_code = ?", *exchangeCode)
		}
		return db
	}
}

func productCodeEqualScope(productCode *string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if productCode != nil {
			return db.Where(table+".product_code = ?", *productCode)
		}
		return db
	}
}

func tradeTypeEqualScope(tradeType *dbModels.TradeType) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if tradeType != nil {
			return db.Where(table+".trade_type = ?", *tradeType)
		}
		return db
	}
}

func createdBetweenScope(createdFrom, createdTo *time.Time) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if createdFrom != nil && createdTo != nil {
			return db.Where(table+".created_at BETWEEN ? AND ?", createdFrom, createdTo)
		}
		return db
	}
}

func orderByScope(order []*Order) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if len(order) > 0 {
			for _, o := range order {
				orderClause := ""
				switch o.Column {
				case OrderColumn_ProductCode:
					orderClause += "product_code"
				case OrderColumn_CreatedAt:
					orderClause += "created_at"
				default:
					continue
				}

				switch o.Direction {
				case OrderDirection_ASC:
					orderClause += " ASC"
				case OrderDirection_DESC:
					orderClause += " DESC"
				}

				db = db.Order(orderClause)
			}
			return db
		}
		return db
	}
}

func limitScope(limit int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if limit > 0 {
			return db.Limit(limit)
		}
		return db
	}
}

func offsetScope(offset int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if offset > 0 {
			return db.Limit(offset)
		}
		return db
	}
}
