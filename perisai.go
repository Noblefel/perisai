// A very basic and naive in-memory rate limiter middleware.
// Compatible with standard library as it uses http.Handler interface (i think?)
//
// THIS IS FOR LEARNING PURPOSE, SO DONT USE IT 😁
package perisai

import (
	"context"
	"net"
	"net/http"
	"sync"
	"time"
)

type Options struct {
	// how many request allowed (per time interval)
	MaxRequest int
	// waiting time to reset the rate limiter
	Interval time.Duration
	// optional custom handler if request hits the rate limit
	Handler http.HandlerFunc
	// function to get the value from http.Request, which will then
	// be stored incrementally in the rate limiter no more than the MaxRequest.
	// Modify this if you want to use other value like IP from headers etc,
	// OR if value is saved elsewhere e.g a session package but still need the request object
	ValueFunc func(r *http.Request) any
	// use this to cancel the clean up process
	KillSwitch context.Context
}

// Default returns a rate limiter middleware and start the cleanup process in the background.
// This is set to 10 max request per 8s interval.
// MUST put this after a middleware (auth), where user ids are stored in the request context.
func Default() func(next http.Handler) http.Handler {
	return New(Options{
		MaxRequest: 10,
		Interval:   time.Second * 8,
		Handler:    defaultHandler,
		ValueFunc:  FuncUserId,
	})
}

// New return a rate limiter middleware and start the cleanup process in the background.
// Use context.Context if you want to cancel the cleanup process.
func New(op Options) func(next http.Handler) http.Handler {
	if op.MaxRequest == 0 {
		panic("max request not set")
	}
	if op.Interval == 0 {
		panic("interval not set")
	}
	if op.ValueFunc == nil {
		panic("value func not set")
	}
	if op.Handler == nil {
		op.Handler = defaultHandler
	}
	if op.KillSwitch == nil {
		op.KillSwitch = context.Background()
	}

	store := new(sync.Map)
	go cleanup(op.KillSwitch, store, op.Interval)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			contextVal := op.ValueFunc(r)
			if contextVal == nil {
				next.ServeHTTP(w, r)
				return
			}

			v, ok := store.Load(contextVal)
			if !ok {
				v = 0
			}

			counts := v.(int) + 1
			if counts > op.MaxRequest {
				op.Handler(w, r)
				return
			}

			store.Swap(contextVal, counts)
			next.ServeHTTP(w, r)
		})
	}
}

// ValueFunc: get "user_id" key value from request context
func FuncUserId(r *http.Request) any {
	return r.Context().Value("user_id")
}

// ValueFunc: get ip address the most basic way
func FuncIP(r *http.Request) any {
	if ip := r.Header.Get("x-real-ip"); ip != "" {
		return ip
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return nil
	}
	return ip
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
