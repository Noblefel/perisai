package perisai

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

var empty http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {}

func TestRateLimit(t *testing.T) {
	rateLimit := New(Options{
		MaxRequest: 5,
		Interval:   50 * time.Millisecond,
		ValueFunc:  FuncUserId,
	})

	var success int

	for range 15 {
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

	t.Run("with empty context value", func(t *testing.T) {
		var success int

		rateLimit := New(Options{
			MaxRequest: 5,
			Interval:   50 * time.Millisecond,
			ValueFunc:  FuncUserId,
		})

		for range 10 {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("get", "/", nil)
			rateLimit(empty).ServeHTTP(rec, req)

			if rec.Code == 200 {
				success++
			}
		}

		if success != 10 {
			t.Errorf("all 10 request should be successful, got %d", success)
		}
	})

	t.Run("with custom value", func(t *testing.T) {
		var success int

		// scenario: limit post request.
		// since method "post" will be too common to be incremented,
		// we'll concat it with user id so it wont affect other users

		rateLimit := New(Options{
			MaxRequest: 5,
			Interval:   50 * time.Millisecond,
			ValueFunc: func(r *http.Request) any {
				if r.Method != "POST" {
					return nil // skip rate limit
				}

				id := r.Context().Value("user_id")
				return fmt.Sprintf("%d:post", id)
			},
		})

		for range 10 {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("POST", "/", nil)
			reqCtx := context.WithValue(req.Context(), "user_id", 1)
			rateLimit(empty).ServeHTTP(rec, req.WithContext(reqCtx))

			if rec.Code == 200 {
				success++
			}
		}

		if success != 5 {
			t.Errorf("successful POST request should be 5, got %d", success)
		}

		success = 0
		for range 10 {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/", nil)
			reqCtx := context.WithValue(req.Context(), "user_id", 1)
			rateLimit(empty).ServeHTTP(rec, req.WithContext(reqCtx))

			if rec.Code == 200 {
				success++
			}
		}

		if success != 10 {
			t.Errorf("all 10 GET request should be successful, got %d", success)
		}

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("get", "/", nil)
		reqCtx := context.WithValue(req.Context(), "user_id", 2)
		rateLimit(empty).ServeHTTP(rec, req.WithContext(reqCtx))

		if rec.Code == 429 {
			t.Error("other user shouldn't be affected")
		}
	})

	t.Run("with multiple users", func(t *testing.T) {
		var success int

		rateLimit := New(Options{
			MaxRequest: 5,
			Interval:   50 * time.Millisecond,
			ValueFunc:  FuncUserId,
		})

		for i := range 10 {
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

		rateLimit := New(Options{
			MaxRequest: 1,
			Interval:   50 * time.Millisecond,
			ValueFunc:  FuncUserId,
			KillSwitch: ctx,
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
