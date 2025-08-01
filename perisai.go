// A very basic and naive in-memory rate limiter middleware.
// Compatible with standard library as it uses http.Handler interface (i think?)
//
// THIS IS FOR LEARNING PURPOSE, SO DONT USE IT 😁
package perisai

import (
	"context"
	"net/http"
	"sync"
	"time"
)

type Options struct {
	// default response if request hits the rate limit
	Handler http.HandlerFunc
	// how many request allowed (per time interval)
	MaxRequest int
	// to get value from the context e.g the user id to be stored in the rate limiter
	ContextKey any
	// waiting time to reset the rate limiter
	Interval time.Duration
}

// New return a rate limiter middleware and start the cleanup process in the background.
// MUST put this after a middleware (like auth), where something like user ids are stored in the request context.
// Use context.Context if you want to cancel the cleanup process.
func New(killswitch context.Context, options Options) func(next http.Handler) http.Handler {
	if options.MaxRequest == 0 {
		panic("max request not set")
	}

	if options.ContextKey == nil {
		panic("context key is not set")
	}

	if options.Interval == 0 {
		panic("interval not set")
	}

	if options.Handler == nil {
		options.Handler = defaultHandler
	}

	store := new(sync.Map)
	go cleanup(killswitch, store, options.Interval)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			contextVal := r.Context().Value(options.ContextKey)

			v, ok := store.Load(contextVal)
			if !ok {
				v = 0
			}

			counts := v.(int) + 1

			if counts > options.MaxRequest {
				options.Handler(w, r)
				return
			}

			store.Swap(contextVal, counts)
			next.ServeHTTP(w, r)
		})
	}
}

func defaultHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusTooManyRequests)
	w.Write([]byte("too many request sorry"))
}

func cleanup(ctx context.Context, store *sync.Map, td time.Duration) {
	ticker := time.NewTicker(td)

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
