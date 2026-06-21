package resilience

import (
	"context"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/apperror"
)

// Config bundles circuit breaker and retry settings sourced from
// application configuration.
type Config struct {
	FailureThreshold    int
	ResetTimeoutMS      int
	RetryMaxAttempts    int
	RetryInitialBackoff int
	RetryMultiplier     int
}

// CallWithResilience guards operation with a circuit breaker and retries
// retryable failures with exponential backoff. The circuit only reacts to
// errors satisfying isRetryable; other errors pass through untouched.
func CallWithResilience[T any](
	ctx context.Context,
	breaker *CircuitBreaker,
	policy RetryPolicy,
	isRetryable func(error) bool,
	operation func(context.Context) (T, error),
) (T, error) {
	if !breaker.AllowRequest() {
		var zero T
		return zero, apperror.ErrServiceUnavailable
	}

	value, err := RetryWithBackoff(ctx, policy, isRetryable, operation)

	switch {
	case err == nil:
		breaker.RecordSuccess()
	case isRetryable(err):
		breaker.RecordFailure()
	}

	return value, err
}
