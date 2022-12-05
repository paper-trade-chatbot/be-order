package redisModels

type OrderProcess int

const (
	OrderProcess_None OrderProcess = iota
	OrderProcess_Waiting
	OrderProcess_Matching
	OrderProcess_Matched
	OrderProcess_Finished
	OrderProcess_Failed
	OrderProcess_Unknown
)

type OrderProcessModel struct {
	OrderID      uint64 `redis:"order_id"`
	OrderProcess OrderProcess
}
