package middlewares

import (
	"encoding/json"
	"honux-core/internal/interfaces"
	"log"
	"net/http"
	"time"
)

type CachedResponse struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

func NewCacheMiddleware(provider interfaces.CacheProvider, ttl time.Duration) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				next.ServeHTTP(w, r)
				return
			}

			cacheKey := r.URL.String()

			if data, err := provider.Get(cacheKey); err == nil {
				var cached CachedResponse
				if err := json.Unmarshal(*data, &cached); err == nil {
					for key, values := range cached.Header {
						for _, v := range values {
							w.Header().Add(key, v)
						}
					}
					w.WriteHeader(cached.StatusCode)
					if len(cached.Body) > 0 {
						w.Write(cached.Body)
					}
					return
				}
				log.Println("deserialization error")
			}

			capture := newCaptureWriter(w)
			next.ServeHTTP(capture, r)

			if capture.statusCode >= 200 && capture.statusCode <= 300 {
				cached := CachedResponse{
					StatusCode: capture.statusCode,
					Header:     capture.header,
					Body:       capture.body.Bytes(),
				}

				if cacheData, err := json.Marshal(cached); err == nil {
					if err := provider.Set(cacheKey, cacheData, ttl); err != nil {
						log.Println("error saving cache", err)
					}
				}
			}
			capture.flushTo()
		})
	}
}
