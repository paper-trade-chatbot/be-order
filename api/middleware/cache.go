package middleware

import (
	"bytes"
	"fmt"
	"hash/fnv"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"

	"github.com/paper-trade-chatbot/be-order/cache"
)

//go:generate msgp -tests=false

// CachedResponse is the struct containing information of the handler response
// that will be stored to Redis cache in MessagePack format.
type CachedResponse struct {
	Status  int                 `msg:"status"`
	Headers map[string][]string `msg:"headers"`
	Body    []byte              `msg:"body"`
}

// CacheKeyFlags indicates which parts of the request are used for key generation.
type CacheKeyFlags uint

const (
	// CacheKeyURL flag signals the use of URL as part of key generation.
	CacheKeyURL = 0x0001

	// CacheKeyUserID flag signals the use of user ID as part of key generation.
	CacheKeyUserID = 0x0002
)

// Cache is a middleware to fetch / store response body from / to Redis.
func Cache(timeout time.Duration, flags CacheKeyFlags) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Only cache for GET requests.
		if ctx.Request.Method != http.MethodGet {
			ctx.Next()
			return
		}

		// Fetch cached response from Redis.
		response := fetchFromCache(ctx, flags)
		if response != nil {
			// Responsed using cached data.
			writeCachedResponse(ctx, response)
			ctx.Abort()
			return
		}

		// Continue processing chain of request handlers and store intercepted
		// response if there is one returned.
		response = interceptResponse(ctx)
		if response != nil {
			storeToCache(ctx, response, timeout, flags)
		}
	}
}

// fetchFromCache tries to fetch the cached data from Redis.
func fetchFromCache(ctx *gin.Context, flags CacheKeyFlags) *CachedResponse {
	// Obtain logger instance.
	logger := GetLogger(ctx)

	redisCache, err := cache.GetRedis()
	if err != nil {
		return nil
	}

	// Generate cache key.
	key := generateCacheKey(ctx, flags)

	// Try to fetch cached data from Redis.
	str, err := redisCache.Get(ctx, key).Result()
	data := []byte(str)
	if err != nil && err != redis.Nil {
		logger.Error(ctx, "Failed to fetch %s: %v", key, err)
		return nil
	}

	// Stop if no data could be found in cache.
	if len(data) <= 0 {
		return nil
	}

	// Unmarshal data from cached MessagePack data to cached response.
	response := CachedResponse{}
	if _, err := response.UnmarshalMsg(data); err != nil {
		logger.Error(ctx, "Failed to unmarshal %s: %v", key, err)
		return nil
	}

	return &response
}

// writeCachedResponse writes the response from obtained cached data.
func writeCachedResponse(ctx *gin.Context, response *CachedResponse) {
	// Obtain logger instance.
	logger := GetLogger(ctx)

	// Write cached status code header.
	ctx.Writer.WriteHeader(response.Status)

	// Write cached HTTP response headers.
	for header, values := range response.Headers {
		for _, value := range values {
			ctx.Writer.Header().Add(header, value)
		}
	}

	// Write cached response body.
	_, err := ctx.Writer.Write(response.Body)
	if err != nil {
		logger.Error(ctx, "Failed to write cached body: %v", err)
	}
}

// responseInterceptor is an interceptor used to obtain handler responses.
type responseInterceptor struct {
	gin.ResponseWriter
	buffer bytes.Buffer
}

// Copy intercepted response data into cache buffer.
func (interceptor *responseInterceptor) Write(buffer []byte) (int, error) {
	// Write buffer data to both original writer and our own cache buffer.
	count, err := interceptor.ResponseWriter.Write(buffer)
	interceptor.buffer.Write(buffer)
	return count, err
}

// interceptResponse continues processing the request handler chain, while
// intercepting the handler response for cache preparation.
func interceptResponse(ctx *gin.Context) *CachedResponse {
	// Create temporary writer to intercept handler response.
	original := ctx.Writer
	interceptor := responseInterceptor{original, bytes.Buffer{}}
	ctx.Writer = &interceptor

	// Continue processing request handler chain.
	ctx.Next()

	// Restore original writer.
	ctx.Writer = original

	// Compose cached response from intercepted response.
	response := CachedResponse{
		Status:  interceptor.Status(),
		Headers: map[string][]string(interceptor.Header()),
		Body:    interceptor.buffer.Bytes(),
	}

	return &response
}

// storeToCache stores the response data to Redis.
func storeToCache(ctx *gin.Context, response *CachedResponse,
	timeout time.Duration, flags CacheKeyFlags) {
	// Obtain logger instance.
	logger := GetLogger(ctx)
	// Obtain Redis connection.
	redisCache, err := cache.GetRedis()
	if err != nil {
		logger.Error(ctx, "Failed to get Redis connection: %v", err)
		return
	}

	// Generate cache key.
	key := generateCacheKey(ctx, flags)

	// Marshal cached response into MessagePack format for storage.
	data, err := response.MarshalMsg(nil)
	if err != nil {
		logger.Error(ctx, "Failed to marshal cached response for %s: %v", key, err)
		return
	}
	// Try to store cached response to Redis.
	_, err = redisCache.Set(ctx, key, data, timeout).Result()
	if err != nil {
		logger.Error(ctx, "Failed to store %s: %v", key, err)
		return
	}
}

// Generate the key used to fetch / store to / from Redis.
func generateCacheKey(ctx *gin.Context, flags CacheKeyFlags) string {
	// Generate hash object.
	hash := fnv.New64a()

	// Compute hash value from request URI.
	if flags&CacheKeyURL != 0 {
		hash.Write([]byte(ctx.Request.RequestURI))
	}
	if flags&CacheKeyUserID != 0 {
		userID, _ := ctx.Get("userId")
		hash.Write([]byte(userID.(string)))
	}

	// Convert hash to hex string.
	hashHexStr := fmt.Sprintf("%016x", hash.Sum64())

	return hashHexStr
}
