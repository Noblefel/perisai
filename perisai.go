// A very basic and naive in-memory rate limiter middleware.
// Compatible with standard library as it uses http.Handler interface
//
// THIS IS FOR LEARNING PURPOSE, SO DONT USE IT 😁
package perisai

import (
	"context"
	"net/http"
	"sync"
	"time"
)

// default resopnse if being rate limited
var DefaultHandler http.HandlerFunc = handle

// how many request allowed (per time interval)
var MaxRequest = 10

var store = new(sync.Map)

func handle(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusTooManyRequests)
	w.Write([]byte("too many request sorry"))
}

// RunCleanup reset the rate limiter after given time
func RunCleanup(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)

	for {
		select {
		case <-ticker.C:
			store.Range(func(key, value any) bool {
				store.Delete(key)
				return true
			})
		case <-ctx.Done():
			return
		}
	}
}

// Rate limit based on value from context.
// Will proceed to the next handler if <= MaxRequest,otherwise will use DefaultHandler as response.
// For example use this after a typical authentication middleware,
// where something like user ids are stored in the request context
func RateLimit(next http.Handler, contextKey any) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contextVal := r.Context().Value(contextKey)

		v, ok := store.Load(contextVal)
		if !ok {
			v = 0
		}

		count := v.(int) + 1

		if count > MaxRequest {
			DefaultHandler(w, r)
			return
		}

		store.Swap(contextVal, count)
		next.ServeHTTP(w, r)
	})
}
