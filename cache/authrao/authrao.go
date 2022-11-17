package authrao

import (
	"context"
	"time"

	"github.com/paper-trade-chatbot/be-order/cache"
)

const (
	Fb_Token_Key          = "fb_token:"
	Line_Token_Key        = "line_token:"
	Line_Access_Token_Key = "line_access_token:"
	Google_Token_Key      = "google_token:"
	Apple_Token_Key       = "apple_token:"
)

func SetFBToken(ctx context.Context, r *cache.RedisInstance, token, externalId string) {
	r.Set(ctx, Fb_Token_Key+token, externalId, time.Hour*72)
}

func SetLineToken(ctx context.Context, r *cache.RedisInstance, token, externalId string) {
	r.Set(ctx, Line_Token_Key+token, externalId, time.Hour*72)
}

func SetGoogleToken(ctx context.Context, r *cache.RedisInstance, token, externalId string) {
	r.Set(ctx, Google_Token_Key+token, externalId, time.Hour*72)
}

func SetLineAccessToken(ctx context.Context, r *cache.RedisInstance, requestToken, responseToken string) {
	r.Set(ctx, Line_Access_Token_Key+requestToken, responseToken, time.Hour*24)
}

func GetLineAccessToken(ctx context.Context, r *cache.RedisInstance, requestToken string) string {
	return r.Get(ctx, Line_Access_Token_Key+requestToken).Val()
}

func DeleteLineAccessToken(ctx context.Context, r *cache.RedisInstance, requestToken string) {
	r.Del(ctx, Line_Access_Token_Key+requestToken)
}

func SetAppleToken(ctx context.Context, r *cache.RedisInstance, token, externalId string) {
	r.Set(ctx, Apple_Token_Key+token, externalId, time.Hour*72)
}

func GetAppleToken(ctx context.Context, r *cache.RedisInstance, token string) string {
	return r.Get(ctx, Apple_Token_Key+token).Val()
}
