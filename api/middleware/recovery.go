package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"

	"github.com/paper-trade-chatbot/be-order/logging"
)

// Recovery is a middleware that recovers from panic then logs the stack trace.
func Recovery() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		defer func() {
			// Recover from panic.
			if recovered := recover(); recovered != nil {
				// Obtain logger instance.
				logger := GetLogger(ctx)

				// Assemble log string.
				message := fmt.Sprintf("\x1b[31m%v\n[Stack Trace]\n%s\x1b[m",
					recovered, debug.Stack())

				// Record the stack trace to logging service, or if we cannot
				// find a logger from this request, use the static logger.
				if logger != nil {
					logger.Error(ctx, message)
				} else {
					logging.Error(ctx, message)
				}

				// Return StatusBadRequest when panic
				ctx.JSON(http.StatusBadRequest, "recover from panic")

				// Discontinue the request handler chain processing.
				ctx.Abort()
			}
		}()

		// Continue processing request chain.
		ctx.Next()
	}
}
