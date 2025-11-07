package httpclient

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"
)

// New creates an HTTP client tuned for outbound service-to-service communication.
func New(timeout time.Duration) *http.Client {
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
}

// Retry executes fn with simple exponential backoff retry semantics.
func Retry(ctx context.Context, attempts int, baseDelay time.Duration, fn func() error) error {
	if attempts <= 1 {
		return fn()
	}

	var err error
	delay := baseDelay
	for i := 0; i < attempts; i++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		err = fn()
		if err == nil {
			return nil
		}

		// Do not sleep after last attempt
		if i == attempts-1 {
			break
		}

		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return ctx.Err()
		}

		// exponential backoff with cap
		delay *= 2
		if delay > 2*time.Second {
			delay = 2 * time.Second
		}
	}

	return err
}

// IsRetriable determines if the error is worth retrying.
func IsRetriable(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout() || netErr.Temporary()
	}
	return errors.Is(err, context.DeadlineExceeded)
}
