package perisai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

var empty http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {}

func TestRateLimit(t *testing.T) {
	rateLimit := New(context.Background(), Options{
		MaxRequest: 5,
		ContextKey: "user_id",
		Interval:   50 * time.Millisecond,
	})

	var success int

	for i := 0; i < 15; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("get", "/", nil)
		reqCtx := context.WithValue(req.Context(), "user_id", 1)
		rateLimit(empty).ServeHTTP(rec, req.WithContext(reqCtx))

		if rec.Code == 200 {
			success++
		}
	}

	if success != 5 {
		t.Errorf("successful request should be 5, got %d", success)
	}

	time.Sleep(75 * time.Millisecond) // give time for the tick
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("get", "/", nil)
	reqCtx := context.WithValue(req.Context(), "user_id", 1)
	rateLimit(empty).ServeHTTP(rec, req.WithContext(reqCtx))

	if rec.Code == 429 {
		t.Error("rate limiter should be cleaned up")
	}

	t.Run("with multiple users", func(t *testing.T) {
		var success int

		rateLimit := New(context.Background(), Options{
			MaxRequest: 5,
			ContextKey: "user_id",
			Interval:   50 * time.Millisecond,
		})

		for i := 0; i < 10; i++ {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("get", "/", nil)
			reqCtx := context.WithValue(req.Context(), "user_id", i)
			rateLimit(empty).ServeHTTP(rec, req.WithContext(reqCtx))

			if rec.Code == 200 {
				success++
			}
		}

		if success != 10 {
			t.Errorf("all 10 request should be successful, got %d", success)
		}
	})

	t.Run("if cleanup is cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		rateLimit := New(ctx, Options{
			MaxRequest: 1,
			ContextKey: "user_id",
			Interval:   50 * time.Millisecond,
		})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("get", "/", nil)
		reqCtx := context.WithValue(req.Context(), "user_id", 1)
		rateLimit(empty).ServeHTTP(rec, req.WithContext(reqCtx))

		cancel()
		time.Sleep(75 * time.Millisecond) // give time for the tick

		rec = httptest.NewRecorder()
		req = httptest.NewRequest("get", "/", nil)
		reqCtx = context.WithValue(req.Context(), "user_id", 1)
		rateLimit(empty).ServeHTTP(rec, req.WithContext(reqCtx))

		if rec.Code != 429 {
			t.Errorf("want 429 status, got %d", rec.Code)
		}
	})
}
