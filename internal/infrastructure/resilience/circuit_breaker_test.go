package resilience_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/infrastructure/resilience"
)

func TestCircuitBreaker_StartsClosedAndAllowsRequests(t *testing.T) {
	breaker := resilience.NewCircuitBreaker(3, time.Second)

	assert.Equal(t, resilience.StateClosed, breaker.State())
	assert.True(t, breaker.AllowRequest())
}

func TestCircuitBreaker_OpensAfterReachingFailureThreshold(t *testing.T) {
	breaker := resilience.NewCircuitBreaker(3, time.Minute)

	breaker.RecordFailure()
	breaker.RecordFailure()
	assert.Equal(t, resilience.StateClosed, breaker.State())

	breaker.RecordFailure()
	assert.Equal(t, resilience.StateOpen, breaker.State())
	assert.False(t, breaker.AllowRequest())
}

func TestCircuitBreaker_SuccessResetsTheFailureCount(t *testing.T) {
	breaker := resilience.NewCircuitBreaker(3, time.Minute)

	breaker.RecordFailure()
	breaker.RecordFailure()
	breaker.RecordSuccess()
	breaker.RecordFailure()
	breaker.RecordFailure()

	assert.Equal(t, resilience.StateClosed, breaker.State())
}

func TestCircuitBreaker_TransitionsToHalfOpenOnceResetTimeoutElapses(t *testing.T) {
	breaker := resilience.NewCircuitBreaker(1, 0)

	breaker.RecordFailure()

	assert.Equal(t, resilience.StateHalfOpen, breaker.State())
	assert.True(t, breaker.AllowRequest())
}

func TestCircuitBreaker_FailedProbeInHalfOpenReopensTheCircuit(t *testing.T) {
	const resetTimeout = 5 * time.Millisecond
	breaker := resilience.NewCircuitBreaker(1, resetTimeout)

	breaker.RecordFailure()
	assert.Equal(t, resilience.StateOpen, breaker.State())

	time.Sleep(2 * resetTimeout)
	assert.Equal(t, resilience.StateHalfOpen, breaker.State())

	breaker.RecordFailure()
	assert.Equal(t, resilience.StateOpen, breaker.State())
}

func TestCircuitBreaker_SuccessfulProbeInHalfOpenClosesTheCircuit(t *testing.T) {
	breaker := resilience.NewCircuitBreaker(1, 0)

	breaker.RecordFailure()
	assert.Equal(t, resilience.StateHalfOpen, breaker.State())

	breaker.RecordSuccess()
	assert.Equal(t, resilience.StateClosed, breaker.State())
}

func TestCircuitBreaker_OnlyOneProbeAllowedAtATimeInHalfOpen(t *testing.T) {
	breaker := resilience.NewCircuitBreaker(1, 0)

	breaker.RecordFailure()
	assert.Equal(t, resilience.StateHalfOpen, breaker.State())

	assert.True(t, breaker.AllowRequest())
	assert.False(t, breaker.AllowRequest())
}
