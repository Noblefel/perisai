package perisai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

var contextKey = "user_id"

func TestRunCleanup(t *testing.T) {
	defer store.Delete(1)
	ctx, cancel := context.WithCancel(context.Background())

	for i := 0; i < 20; i++ {
		store.Store(i, i)
	}

	go RunCleanup(ctx, 50*time.Millisecond)

	if _, ok := store.Load(19); !ok {
		t.Error("missing value") //should wait for tick
	}

	time.Sleep(70 * time.Millisecond)

	if _, ok := store.Load(19); ok {
		t.Error("value still exist")
	}

	cancel()
	store.Store(1, 1)
	time.Sleep(70 * time.Millisecond)

	if _, ok := store.Load(1); !ok {
		t.Error("value gets deleted after context has been cancelled")
	}
}

func TestRateLimit(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("get", "/", nil)
	ctx := context.WithValue(req.Context(), contextKey, 1)
	req = req.WithContext(ctx)
	RateLimit(handler, contextKey).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("should not be rate limited yet, got: %d", rec.Code)
	}

	if count, _ := store.Load(1); count == nil || count != 1 {
		t.Errorf("count should be set to 1, got %v", count)
	}

	MaxRequest = 5
	store.Swap(1, 5)
	rec = httptest.NewRecorder()
	req = httptest.NewRequest("get", "/", nil)
	ctx = context.WithValue(req.Context(), contextKey, 1)
	req = req.WithContext(ctx)
	RateLimit(handler, contextKey).ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("should return 429, got: %d", rec.Code)
	}

	t.Run("should not interfere with different request", func(t *testing.T) {
		//we know user id of 1 is already being rate limited
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("get", "/", nil)
		ctx := context.WithValue(req.Context(), contextKey, 2)
		req = req.WithContext(ctx)
		RateLimit(handler, contextKey).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("want 200, got: %d", rec.Code)
		}
	})
}
