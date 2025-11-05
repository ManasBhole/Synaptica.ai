package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/synaptica-ai/platform/pkg/common/logger"
	"github.com/synaptica-ai/platform/pkg/gateway/auth"
)

type contextKey string

const UserContextKey contextKey = "user"

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		next.ServeHTTP(w, r)

		logger.Log.WithFields(map[string]interface{}{
			"method":     r.Method,
			"path":       r.URL.Path,
			"remote_addr": r.RemoteAddr,
			"duration":   time.Since(start).Milliseconds(),
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

