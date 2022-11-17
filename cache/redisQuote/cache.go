package redisQuote

import (
	"context"
)

// Initialize initializes the cache module.
func Initialize(ctx context.Context) {
	// Initialize Redis.
	initializeRedis(ctx)
}

// Finalize finalizes the cache module.
func Finalize() {
	// Finalize Redis.
	finalizeRedis()
}
