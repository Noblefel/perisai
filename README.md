A very basic and naive in-memory rate limiter middleware. Compatible with standard library (i think?) as it uses http.Handler interface

**THIS IS FOR LEARNING PURPOSE, SO DONT USE IT 😁**

```
go get github.com/Noblefel/perisai
```

**Example #1 - limit by user id:**

```go
package main

import (
	"context"
	"net/http"

	"github.com/Noblefel/perisai"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ping"))
	})

	rateLimit := perisai.Default()
	http.ListenAndServe("localhost:8080", auth(rateLimit(mux)))
}

// example a typical authentication middleware
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

func verifyToken(string) (int, error) { return 1, nil }
```

**Example #2 - limit by ip (basic):**

```go
rateLimit := perisai.New(perisai.Options{
	MaxRequest: 10,
	Interval:   8 * time.Second,
	ValueFunc:  perisai.FuncIP,
})
```

**Example #3 - custom value func**:

scenario: limit post request once every 10s. since method "post" will be too common to be incremented, we'll concat it with user id so it wont affect other users

```go
postrequestFunc := func (r *http.Request) any {
	if r.Method != "POST" {
		return nil // ignore other methods
	}

	id := r.Context().Value("user_id")
	return fmt.Sprintf("%d:post", id)
}

rateLimit := perisai.New(perisai.Options{
	MaxRequest: 1,
	Interval:   time.Second * 10,
	ValueFunc:  postrequestFunc,
})
```
