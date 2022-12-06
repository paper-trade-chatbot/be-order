package cronjob

import (
	"context"
	"reflect"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/go-redis/redis/v9"
	"github.com/gofrs/uuid"
	"github.com/paper-trade-chatbot/be-order/cache"
	"github.com/paper-trade-chatbot/be-order/logging"
)

func Cron() {

	scheduler := gocron.NewScheduler(time.UTC)

	// Start all the pending jobs
	scheduler.StartAsync()

}

func work(cronjob func(context.Context) error, generateKey func() string, maxDuration time.Duration) {

	cronjobID, _ := uuid.NewV4()
	ctx := context.WithValue(context.Background(), logging.ContextKeyRequestId, cronjobID.String())

	funcName := strings.Split(runtime.FuncForPC(reflect.ValueOf(cronjob).Pointer()).Name(), "/")
	logging.Info(ctx, "[cronjob] start %s", funcName[len(funcName)-1])
	key := "cronjob:" + generateKey()

	r, _ := cache.GetRedis()
	if flag, _ := r.SetNX(ctx, key, cronjobID.String(), maxDuration).Result(); !flag {
		logging.Info(ctx, "[Cronjob] key already exist: %s", key)
		return
	}

	ch := make(chan int, 1)

	ctxTimeout, cancel := context.WithTimeout(ctx, maxDuration)
	defer cancel()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				// Record the stack trace to logging service, or if we cannot
				// find a logging from this request, use the static logging.
				logging.Error(ctx, "\x1b[31m%v\n[Stack Trace]\n%s\x1b[m", r, debug.Stack())
			}
			ch <- 1
		}()
		err := cronjob(ctxTimeout)
		if err != nil {
			logging.Error(ctxTimeout, "[Cronjob] %s error: %v", key, err)
		}
	}()

	select {
	case <-ctxTimeout.Done():
		logging.Error(ctxTimeout, "[Cronjob] %s timeout error: %v", key, ctxTimeout.Err())
	case <-ch:

	}

	value, err := r.Get(ctx, key).Result()
	if err != nil && err.Error() != redis.Nil.Error() && value == cronjobID.String() {
		if err := r.Del(ctx, key).Err(); err != nil && err.Error() != redis.Nil.Error() {
			logging.Error(ctxTimeout, "[Cronjob] %s failed to delete key: %v", key, err)
		}
	}
}
