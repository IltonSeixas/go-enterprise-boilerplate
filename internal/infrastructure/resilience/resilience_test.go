package resilience_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/apperror"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/infrastructure/resilience"
)

func newPolicy() resilience.RetryPolicy {
	return resilience.NewRetryPolicy(3, time.Millisecond, 1)
}

func TestCallWithResilience_SucceedsAndKeepsCircuitClosed(t *testing.T) {
	breaker := resilience.NewCircuitBreaker(3, time.Minute)

	result, err := resilience.CallWithResilience(context.Background(), breaker, newPolicy(), alwaysRetryable,
		func(context.Context) (int, error) { return 1, nil })

	require.NoError(t, err)
	assert.Equal(t, 1, result)
	assert.Equal(t, resilience.StateClosed, breaker.State())
}

func TestCallWithResilience_OpensCircuitAfterRepeatedFailures(t *testing.T) {
	breaker := resilience.NewCircuitBreaker(1, time.Minute)

	_, err := resilience.CallWithResilience(context.Background(), breaker, newPolicy(), alwaysRetryable,
		func(context.Context) (int, error) { return 0, errBoom })

	require.ErrorIs(t, err, errBoom)
	assert.Equal(t, resilience.StateOpen, breaker.State())
}

func TestCallWithResilience_FailsFastWithoutCallingOperationWhenCircuitIsOpen(t *testing.T) {
	breaker := resilience.NewCircuitBreaker(1, time.Minute)
	breaker.RecordFailure()
	require.Equal(t, resilience.StateOpen, breaker.State())

	var calls atomic.Int32
	_, err := resilience.CallWithResilience(context.Background(), breaker, newPolicy(), alwaysRetryable,
		func(context.Context) (int, error) {
			calls.Add(1)
			return 1, nil
		})

	require.ErrorIs(t, err, apperror.ErrServiceUnavailable)
	assert.Equal(t, int32(0), calls.Load())
}

func TestCallWithResilience_DoesNotTripBreakerOnNonRetryableErrors(t *testing.T) {
	breaker := resilience.NewCircuitBreaker(1, time.Minute)

	_, err := resilience.CallWithResilience(context.Background(), breaker, newPolicy(), neverRetryable,
		func(context.Context) (int, error) { return 0, errBoom })

	require.ErrorIs(t, err, errBoom)
	assert.Equal(t, resilience.StateClosed, breaker.State())
}
