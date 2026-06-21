package resilience

import (
	"context"
	"time"
)

// RetryPolicy configures exponential backoff between retry attempts.
type RetryPolicy struct {
	MaxAttempts       int
	InitialBackoff    time.Duration
	BackoffMultiplier int
}

func NewRetryPolicy(maxAttempts int, initialBackoff time.Duration, backoffMultiplier int) RetryPolicy {
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	if backoffMultiplier < 1 {
		backoffMultiplier = 1
	}
	return RetryPolicy{
		MaxAttempts:       maxAttempts,
		InitialBackoff:    initialBackoff,
		BackoffMultiplier: backoffMultiplier,
	}
}

func (p RetryPolicy) backoffForAttempt(attempt int) time.Duration {
	backoff := p.InitialBackoff
	for i := 0; i < attempt; i++ {
		backoff *= time.Duration(p.BackoffMultiplier)
	}
	return backoff
}

// RetryWithBackoff retries operation while isRetryable(err) is true and the
// attempt budget allows, sleeping with exponential backoff between attempts.
func RetryWithBackoff[T any](
	ctx context.Context,
	policy RetryPolicy,
	isRetryable func(error) bool,
	operation func(context.Context) (T, error),
) (T, error) {
	var attempt int
	for {
		value, err := operation(ctx)
		if err == nil {
			return value, nil
		}
		if attempt+1 >= policy.MaxAttempts || !isRetryable(err) {
			return value, err
		}

		select {
		case <-ctx.Done():
			var zero T
			return zero, ctx.Err()
		case <-time.After(policy.backoffForAttempt(attempt)):
		}
		attempt++
	}
}
