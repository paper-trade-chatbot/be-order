package cache

import (
	"github.com/go-redsync/redsync/v4"
	redsyncRedis "github.com/go-redsync/redsync/v4/redis"
	goredis "github.com/go-redsync/redsync/v4/redis/goredis/v8"
)

type RedisSyncPool struct {
	redsyncRedis.Pool
}

type RedisSyncInstance struct {
	*redsync.Redsync
}

var redisSyncPool *RedisSyncPool
var redsyncInstance *RedisSyncInstance

func (r *RedisInstance) GetLock(lock string) error {
	if redsyncInstance == nil {
		r.initRedsyncInstance()
	}

	mutex := redsyncInstance.NewMutex(lock, redsync.WithTries(1))

	return mutex.Lock()
}

func (r *RedisInstance) initSyncPool() {
	redisSyncPool = &RedisSyncPool{goredis.NewPool(r)}
}

func (r *RedisInstance) initRedsyncInstance() {
	if redisSyncPool == nil {
		r.initSyncPool()
	}
	redsyncInstance = &RedisSyncInstance{redsync.New(redisSyncPool)}
}
