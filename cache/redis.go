package cache

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/go-redis/redismock/v8"
	"github.com/paper-trade-chatbot/be-order/config"
	"github.com/paper-trade-chatbot/be-order/logging"
)

// RedisInstance and the context used to retrieve connections.
type RedisInstance struct {
	*redis.Client
	sync.RWMutex
	lastConnet time.Time
}

var redisInstance *RedisInstance
var redisRootCtx context.Context

// Static Redis configuration variables.
var redisEndpoint string
var redisPassword string
var redisDB int
var redisPoolsize int
var idleTimeout time.Duration

func init() {
	redisInstance = &RedisInstance{RWMutex: sync.RWMutex{}}
}

// Initialize Redis connection pool.
func initializeRedis(ctx context.Context) {
	// load the cache configurations.

	redisEndpoint = config.GetString("REDIS_ENDPOINT")
	redisPassword = config.GetString("REDIS_PASSWORD")
	redisDB = config.GetInt("REDIS_DB")
	redisPoolsize = config.GetInt("REDIS_POOLSIZE")
	idleTimeout = config.GetMilliseconds("REDIS_IDLE_TIMEOUT")

	// Initialize a Redis client. Here we assume if the endpoint connects to
	// port 6379, the target Redis server is configured as a single instance,
	// i.e. local dev server. If the endpoint connects to 26379, then we are
	// connecting to a Redis cluster configured to use sentinels.
	redisInstance.Lock()
	defer redisInstance.Unlock()

	// check last connection time
	if time.Since(redisInstance.lastConnet) <= 5*time.Second {
		return
	}

	var redisClient *redis.Client

	if strings.Contains(redisEndpoint, ":26379") {
		redisClient = redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:    "redis-master",
			SentinelAddrs: []string{redisEndpoint},
			Password:      redisPassword,
			DB:            redisDB, // use default DB
			PoolSize:      redisPoolsize,
			IdleTimeout:   idleTimeout,
		})
	} else if strings.Contains(redisEndpoint, ":6379") {
		redisClient = redis.NewClient(&redis.Options{
			Addr:        redisEndpoint,
			Password:    redisPassword,
			DB:          redisDB, // use default DB
			PoolSize:    redisPoolsize,
			IdleTimeout: idleTimeout,
		})
	} else {
		panic(fmt.Errorf("cannot determine Redis mode"))
	}
	redisInstance.Client = redisClient

	// Perform a PING to see if the connection is usable.
	if err := redisInstance.Ping(ctx).Err(); err != nil {
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
	redisClient := redisInstance.Client
	redisInstance.RUnlock()

	// Check to see if the Redis connection client has been initialized first.
	if redisClient == nil {
		logging.Error(context.Background(), "Redis connection client not initialized")
		return
	}

	// Close the Redis connection client.
	if err := redisClient.Close(); err != nil {
		logging.Error(context.Background(), "Failed to close Redis connection pool: %v", err)
	}
}

// GetRedis returns a Redis connection.
func GetRedis() (*RedisInstance, error) {
	redisInstance.RLock()
	redisClient := redisInstance.Client
	redisInstance.RUnlock()

	// Grab and return a connection from the Redis connection pool.
	if redisClient == nil {
		initializeRedis(redisRootCtx)
	}

	// re-assign redisInstance if dns error
	// if err := redisClient.Ping(redisRootCtx).Err(); err != nil {
	// 	initializeRedis(redisRootCtx)
	// }
	return redisInstance, nil
}

// SetRedisMock is only used in mock
func SetRedisMock() (redismock.ClientMock, func()) {
	redisMockClient, redisMockCtrl := redismock.NewClientMock()
	redisInstance.Client = redisMockClient
	redisRootCtx = context.Background()

	closeRedis := func() {
		finalizeRedis()
	}

	return redisMockCtrl, closeRedis
}
