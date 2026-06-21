package resilience_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/infrastructure/resilience"
)

var errBoom = errors.New("boom")

func alwaysRetryable(error) bool { return true }
func neverRetryable(error) bool  { return false }

func TestRetryWithBackoff_ReturnsImmediatelyOnSuccess(t *testing.T) {
	policy := resilience.NewRetryPolicy(3, time.Millisecond, 1)
	var calls atomic.Int32

	result, err := resilience.RetryWithBackoff(context.Background(), policy, alwaysRetryable,
		func(context.Context) (int, error) {
			calls.Add(1)
			return 42, nil
		})

	require.NoError(t, err)
	assert.Equal(t, 42, result)
	assert.Equal(t, int32(1), calls.Load())
}

func TestRetryWithBackoff_StopsRetryingOnceAnAttemptSucceeds(t *testing.T) {
	policy := resilience.NewRetryPolicy(5, time.Millisecond, 1)
	var calls atomic.Int32

	result, err := resilience.RetryWithBackoff(context.Background(), policy, alwaysRetryable,
		func(context.Context) (int, error) {
			n := calls.Add(1)
			if n < 3 {
				return 0, errBoom
			}
			return 7, nil
		})

	require.NoError(t, err)
	assert.Equal(t, 7, result)
	assert.Equal(t, int32(3), calls.Load())
}

func TestRetryWithBackoff_RetriesRetryableErrorsUntilMaxAttempts(t *testing.T) {
	policy := resilience.NewRetryPolicy(3, time.Millisecond, 1)
	var calls atomic.Int32

	_, err := resilience.RetryWithBackoff(context.Background(), policy, alwaysRetryable,
		func(context.Context) (int, error) {
			calls.Add(1)
			return 0, errBoom
		})

	require.ErrorIs(t, err, errBoom)
	assert.Equal(t, int32(3), calls.Load())
}

func TestRetryWithBackoff_DoesNotRetryNonRetryableErrors(t *testing.T) {
	policy := resilience.NewRetryPolicy(5, time.Millisecond, 1)
	var calls atomic.Int32

	_, err := resilience.RetryWithBackoff(context.Background(), policy, neverRetryable,
		func(context.Context) (int, error) {
			calls.Add(1)
			return 0, errBoom
		})

	require.ErrorIs(t, err, errBoom)
	assert.Equal(t, int32(1), calls.Load())
}
