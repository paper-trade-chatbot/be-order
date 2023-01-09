package orderProcessDao

import (
	"context"
	"strconv"
	"time"

	"github.com/go-redis/redis/v9"
	"github.com/paper-trade-chatbot/be-common/cache"
	"github.com/paper-trade-chatbot/be-common/marshaller"
	"github.com/paper-trade-chatbot/be-order/models/redisModels"
)

const (
	first_field = "order_process"
)

type QueryModel struct {
	OrderID uint64 `redis:"order_id"`
}

func Get(ctx context.Context, rds *cache.RedisInstance, query *QueryModel) (*redisModels.OrderProcessModel, error) {

	key := first_field + ":" + marshaller.Marshal(ctx, query, "redis", ":")

	value, err := rds.Get(ctx, key).Result()
	if err != nil {
		if err.Error() == redis.Nil.Error() {
			return nil, nil
		}
		return nil, err
	}

	orderProcess, err := strconv.Atoi(value)
	if err != nil {
		return nil, err
	}

	return &redisModels.OrderProcessModel{
		OrderID:      query.OrderID,
		OrderProcess: redisModels.OrderProcess(orderProcess),
	}, nil
}

func Update(ctx context.Context, rds *cache.RedisInstance, model *redisModels.OrderProcessModel, expiration *time.Duration) error {

	key := first_field + ":" + marshaller.Marshal(ctx, model, "redis", ":")
	var ex time.Duration
	if expiration == nil {
		ex = time.Duration(0)
	} else {
		ex = *expiration
	}
	if err := rds.Set(ctx, key, strconv.Itoa(int(model.OrderProcess)), ex).Err(); err != nil {
		return err
	}
	return nil
}

func Delete(ctx context.Context, rds *cache.RedisInstance, model *redisModels.OrderProcessModel) error {

	key := first_field + ":" + marshaller.Marshal(ctx, model, "redis", ":")
	if err := rds.Del(ctx, key).Err(); err != nil {
		return err
	}
	return nil
}
