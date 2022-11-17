package redisCluster

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/paper-trade-chatbot/be-order/logging"
)

type RedisCluster struct {
	Cc *redis.ClusterClient
	sync.RWMutex
	lastConnet time.Time
}

var redisInstance *RedisCluster
var redisRootCtx context.Context

var redisAddresses []string
var redisUser, redisPassword string
var redisPoolSize int
var redisPoolTimeout, redisIdleTimeout, redisReadTimeout, redisWriteTimeout time.Duration

func init() {
	redisInstance = &RedisCluster{RWMutex: sync.RWMutex{}}

	redisAddresses = []string{"redis-cluster:6379"}
	redisUser = "default"
	redisPassword = "123456"
	redisPoolSize = 10000
}

func InitRedisClusterAdapter(ctx context.Context) {
	redisInstance.Lock()
	defer redisInstance.Unlock()

	// check last connection time
	if time.Since(redisInstance.lastConnet) <= 5*time.Second {
		return
	}

	fmt.Println(redisAddresses, redisUser, redisPassword)

	client := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:        redisAddresses,
		Username:     redisUser,
		Password:     redisPassword,
		PoolSize:     redisPoolSize,
		PoolTimeout:  10 * time.Second,
		IdleTimeout:  10 * time.Second,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		//RouteByLatency: true,
		//RouteRandomly:  true,
		//MaxRetries: 2,
	})

	redisInstance.Cc = client

	pctx, pccl := context.WithTimeout(ctx, 100*time.Second)
	defer pccl()

	// Perform a PING to see if the connection is usable.
	if err := redisInstance.Cc.Ping(pctx).Err(); err != nil {
		logging.Error(ctx, "redis Ping: %v", err)
		return
	}
	// update last connection time
	redisInstance.lastConnet = time.Now()

	// Keep a reference to root context.
	redisRootCtx = ctx
}

// Finalize Redis connection client.
func finalizeRedis() {
	redisInstance.RLock()
	redisClient := redisInstance.Cc
	redisInstance.RUnlock()

	// Check to see if the Redis connection client has been initialized first.
	if redisClient == nil {
		logging.Error(ctx, "Redis connection client not initialized")
		return
	}

	// Close the Redis connection client.
	if err := redisClient.Close(); err != nil {
		logging.Error(ctx, "Failed to close Redis connection pool: %v", err)
	}
}

// GetRedis returns a Redis connection.
func GetRedis() (*RedisCluster, error) {
	redisInstance.RLock()
	redisClient := redisInstance.Cc
	redisInstance.RUnlock()

	// Grab and return a connection from the Redis connection pool.
	if redisClient == nil {
		InitRedisClusterAdapter(redisRootCtx)
	}

	// re-assign redisInstance if dns error
	if err := redisClient.Ping(redisRootCtx).Err(); err != nil {
		InitRedisClusterAdapter(redisRootCtx)
	}

	return redisInstance, nil
}
