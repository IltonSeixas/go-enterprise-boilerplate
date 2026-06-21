package resilience

import (
	"sync"
	"time"
)

type CircuitState int

const (
	StateClosed CircuitState = iota
	StateOpen
	StateHalfOpen
)

// CircuitBreaker implements a Closed -> Open -> HalfOpen -> Closed state
// machine for guarding calls to a single transient-failure-prone dependency
// (e.g. a Redis connection).
//
// Open rejects calls immediately until resetTimeout elapses, at which point
// a single probe call is allowed through (HalfOpen). That probe's outcome
// decides whether the breaker closes again or re-opens.
type CircuitBreaker struct {
	mu                  sync.Mutex
	failureThreshold    int
	resetTimeout        time.Duration
	consecutiveFailures int
	state               CircuitState
	openedAt            time.Time
	probeInFlight       bool
}

func NewCircuitBreaker(failureThreshold int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		failureThreshold: failureThreshold,
		resetTimeout:     resetTimeout,
		state:            StateClosed,
	}
}

func (b *CircuitBreaker) State() CircuitState {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.stateLocked()
}

func (b *CircuitBreaker) stateLocked() CircuitState {
	if b.state == StateOpen && time.Since(b.openedAt) >= b.resetTimeout {
		return StateHalfOpen
	}
	return b.state
}

// AllowRequest reports whether a call is currently allowed to proceed. In
// HalfOpen, only a single probe is let through at a time.
func (b *CircuitBreaker) AllowRequest() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.stateLocked() {
	case StateClosed:
		return true
	case StateOpen:
		return false
	default: // StateHalfOpen
		if b.probeInFlight {
			return false
		}
		b.probeInFlight = true
		return true
	}
}

func (b *CircuitBreaker) RecordSuccess() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.consecutiveFailures = 0
	b.state = StateClosed
	b.probeInFlight = false
}

func (b *CircuitBreaker) RecordFailure() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.consecutiveFailures++
	if b.consecutiveFailures >= b.failureThreshold {
		b.state = StateOpen
		b.openedAt = time.Now()
		b.probeInFlight = false
	}
}
