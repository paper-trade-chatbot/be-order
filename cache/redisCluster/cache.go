package redisCluster

import (
	"context"
)

// Initialize initializes the cache module.
func Initialize(ctx context.Context) {
	// Initialize Redis.
	InitRedisClusterAdapter(ctx)
}

// Finalize finalizes the cache module.
func Finalize() {
	// Finalize Redis.
	finalizeRedis()
}
