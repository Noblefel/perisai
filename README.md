A very basic and naive in-memory rate limiter middleware. Compatible with standard library (i think?) as it uses http.Handler interface

**THIS IS FOR LEARNING PURPOSE, SO DONT USE IT 😁**

```
go mod init github.com/Noblefel/perisai
```

Example:

```go
package main

import (
	"context"
	"net/http"
	"time"

	"github.com/Noblefel/perisai"
)

func main() {
	go perisai.RunCleanup(context.Background(), 5*time.Second)
	perisai.MaxRequest = 10

	mux := http.NewServeMux()
	mux.HandleFunc("/", ping)

	handler := auth(perisai.RateLimit(mux, "user_id"))
	http.ListenAndServe("localhost:8080", handler)
}

// example authentication middleware
func auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString := r.Header.Get("Authorization")

		userId, err := verifyToken(tokenString)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), "user_id", userId)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func ping(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ping"))
}

func verifyToken(string) (int, error) {
	return 1, nil
}

```
