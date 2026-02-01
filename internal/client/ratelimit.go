package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"time"

	"github.com/deevus/terraform-provider-truenas/internal/api"
	"golang.org/x/time/rate"
)

const (
	defaultRateLimit  = 300 // calls per minute (5 per second)
	defaultMaxRetries = 3
)

// RateLimitedClient wraps a Client with rate limiting and retry logic.
type RateLimitedClient struct {
	client     Client
	limiter    *rate.Limiter
	maxRetries int
	classifier RetryClassifier
}

// Compile-time check that RateLimitedClient implements Client.
var _ Client = (*RateLimitedClient)(nil)

// NewRateLimitedClient creates a client wrapper with rate limiting and retry.
// If callsPerMinute is 0 or negative, defaults to 18. If maxRetries is negative, defaults to 3.
// Setting maxRetries to 0 disables retries. If classifier is nil, defaults to SSHRetryClassifier.
func NewRateLimitedClient(client Client, callsPerMinute int, maxRetries int, classifier RetryClassifier) *RateLimitedClient {
	if callsPerMinute <= 0 {
		callsPerMinute = defaultRateLimit
	}
	if maxRetries < 0 {
		maxRetries = defaultMaxRetries
	}
	if classifier == nil {
		classifier = &SSHRetryClassifier{}
	}

	// Convert calls/minute to rate.Limiter parameters
	interval := time.Minute / time.Duration(callsPerMinute)
	limiter := rate.NewLimiter(rate.Every(interval), 1)

	return &RateLimitedClient{
		client:     client,
		limiter:    limiter,
		maxRetries: maxRetries,
		classifier: classifier,
	}
}

// Connect delegates to the underlying client.
func (r *RateLimitedClient) Connect(ctx context.Context) error {
	return r.client.Connect(ctx)
}

// Version delegates to the underlying client.
func (r *RateLimitedClient) Version() api.Version {
	return r.client.Version()
}

// Call executes a midclt command and returns the parsed JSON response.
func (r *RateLimitedClient) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	// Wait for rate limiter (respects context cancellation)
	if err := r.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= r.maxRetries; attempt++ {
		result, err := r.client.Call(ctx, method, params)
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Don't retry non-retriable errors
		if !r.classifier.IsRetriable(err) {
			return nil, err
		}

		// Don't sleep after final attempt
		if attempt == r.maxRetries {
			break
		}

		// Backoff before retry
		delay := CalculateBackoff(attempt)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return nil, fmt.Errorf("after %d retries: %w", r.maxRetries, lastErr)
}

// CallAndWait executes a method that returns a job, with rate limiting and retry.
func (r *RateLimitedClient) CallAndWait(ctx context.Context, method string, params any) (json.RawMessage, error) {
	// Wait for rate limiter
	if err := r.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= r.maxRetries; attempt++ {
		result, err := r.client.CallAndWait(ctx, method, params)
		if err == nil {
			return result, nil
		}

		lastErr = err

		if !r.classifier.IsRetriable(err) {
			return nil, err
		}

		if attempt == r.maxRetries {
			break
		}

		delay := CalculateBackoff(attempt)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
		}
	}

	return nil, fmt.Errorf("after %d retries: %w", r.maxRetries, lastErr)
}

// WriteFile delegates to the underlying client with rate limiting.
func (r *RateLimitedClient) WriteFile(ctx context.Context, path string, params WriteFileParams) error {
	if err := r.limiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limiter: %w", err)
	}
	return r.client.WriteFile(ctx, path, params)
}

// ReadFile delegates to the underlying client with rate limiting.
func (r *RateLimitedClient) ReadFile(ctx context.Context, path string) ([]byte, error) {
	if err := r.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}
	return r.client.ReadFile(ctx, path)
}

// DeleteFile delegates to the underlying client with rate limiting.
func (r *RateLimitedClient) DeleteFile(ctx context.Context, path string) error {
	if err := r.limiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limiter: %w", err)
	}
	return r.client.DeleteFile(ctx, path)
}

// RemoveDir delegates to the underlying client with rate limiting.
func (r *RateLimitedClient) RemoveDir(ctx context.Context, path string) error {
	if err := r.limiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limiter: %w", err)
	}
	return r.client.RemoveDir(ctx, path)
}

// RemoveAll delegates to the underlying client with rate limiting.
func (r *RateLimitedClient) RemoveAll(ctx context.Context, path string) error {
	if err := r.limiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limiter: %w", err)
	}
	return r.client.RemoveAll(ctx, path)
}

// FileExists delegates to the underlying client with rate limiting.
func (r *RateLimitedClient) FileExists(ctx context.Context, path string) (bool, error) {
	if err := r.limiter.Wait(ctx); err != nil {
		return false, fmt.Errorf("rate limiter: %w", err)
	}
	return r.client.FileExists(ctx, path)
}

// Chown delegates to the underlying client with rate limiting.
func (r *RateLimitedClient) Chown(ctx context.Context, path string, uid, gid int) error {
	if err := r.limiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limiter: %w", err)
	}
	return r.client.Chown(ctx, path, uid, gid)
}

// ChmodRecursive delegates to the underlying client with rate limiting.
func (r *RateLimitedClient) ChmodRecursive(ctx context.Context, path string, mode fs.FileMode) error {
	if err := r.limiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limiter: %w", err)
	}
	return r.client.ChmodRecursive(ctx, path, mode)
}

// MkdirAll delegates to the underlying client with rate limiting.
func (r *RateLimitedClient) MkdirAll(ctx context.Context, path string, mode fs.FileMode) error {
	if err := r.limiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limiter: %w", err)
	}
	return r.client.MkdirAll(ctx, path, mode)
}

// Close closes the underlying client.
func (r *RateLimitedClient) Close() error {
	return r.client.Close()
}
