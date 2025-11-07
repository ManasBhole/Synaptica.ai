package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/synaptica-ai/platform/pkg/common/logger"
	"github.com/synaptica-ai/platform/pkg/gateway/auth"
)

type contextKey string

const UserContextKey contextKey = "user"

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		// Ensure a request ID exists
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = uuid.New().String()
		}

		// Propagate request ID downstream
		r.Header.Set("X-Request-ID", reqID)
		next.ServeHTTP(w, r)

		logger.Log.WithFields(map[string]interface{}{
			"method":      r.Method,
			"path":        r.URL.Path,
			"remote_addr": r.RemoteAddr,
			"request_id":  reqID,
			"duration":    time.Since(start).Milliseconds(),
		}).Info("HTTP request")
	})
}

func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				logger.Log.WithField("error", err).Error("Panic recovered")
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func Authenticate(oidcAuth *auth.OIDCAuthenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get("Authorization")
			if token == "" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Extract Bearer token
			if len(token) > 7 && token[:7] == "Bearer " {
				token = token[7:]
			}

			claims, err := oidcAuth.ValidateToken(r.Context(), token)
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Add user to context
			ctx := context.WithValue(r.Context(), UserContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Row-Level Security middleware
func RLS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract user from context
		user := r.Context().Value(UserContextKey)
		if user == nil {
			// For public endpoints, allow
			next.ServeHTTP(w, r)
			return
		}

		// Add RLS context - in production, this would set tenant/org filters
		ctx := context.WithValue(r.Context(), "rls_filters", map[string]interface{}{
			"user_id": user.(map[string]interface{})["sub"],
		})

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Simple token-bucket rate limiter middleware (per-process)
func RateLimit(rps int, burst int) func(http.Handler) http.Handler {
	type bucket struct {
		tokens int
		last   time.Time
		mu     sync.Mutex
	}
	b := &bucket{tokens: burst, last: time.Now()}
	refill := func() {
		now := time.Now()
		elapsed := now.Sub(b.last).Seconds()
		add := int(elapsed * float64(rps))
		if add > 0 {
			b.tokens += add
			if b.tokens > burst {
				b.tokens = burst
			}
			b.last = now
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			b.mu.Lock()
			refill()
			if b.tokens <= 0 {
				b.mu.Unlock()
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			b.tokens--
			b.mu.Unlock()
			next.ServeHTTP(w, r)
		})
	}
}

// CORS middleware (allow basic dev flows)
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func BodyLimit(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}
