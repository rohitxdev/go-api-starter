package util

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

const (
	StatusClosed uint = iota
	StatusHalfOpen
	StatusOpen
)

var (
	ErrorCircuitBreakerOpen = errors.New("circuit-breaker is open")
)

type CircuitBreaker struct {
	mu               sync.Mutex
	status           uint
	lastFailureAt    time.Time
	resetTimeout     time.Duration
	successThreshold uint
	successes        uint
	failureThreshold uint
	failures         uint
	onStatusChange   func(from uint, to uint)
}

type CircuitBreakerOpts struct {
	ResetTimeout     time.Duration
	FailureThreshold uint
	SuccessThreshold uint
	OnStatusChange   func(from uint, to uint)
}

func NewCircuitBreaker(opts *CircuitBreakerOpts) *CircuitBreaker {
	if opts == nil {
		opts = &CircuitBreakerOpts{
			ResetTimeout:     time.Second * 5,
			FailureThreshold: 5,
			SuccessThreshold: 2,
		}
	}

	return &CircuitBreaker{
		resetTimeout:     opts.ResetTimeout,
		failureThreshold: opts.FailureThreshold,
		successThreshold: opts.SuccessThreshold,
		onStatusChange:   opts.OnStatusChange,
	}
}

func (cb *CircuitBreaker) Do(fn func() (any, error)) (any, error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	initialStatus := cb.status
	var val any
	var err error

checkStatus:
	switch cb.status {
	case StatusClosed:
		if val, err = fn(); err != nil {
			cb.lastFailureAt = time.Now()
			cb.failures += 1
			if cb.failures >= cb.failureThreshold {
				cb.status = StatusOpen
				cb.successes = 0
			}
		}
	case StatusHalfOpen:
		if val, err = fn(); err != nil {
			cb.status = StatusOpen
			cb.lastFailureAt = time.Now()
			cb.successes = 0
			cb.failures = 0
		} else {
			cb.successes += 1
			if cb.successes >= cb.successThreshold {
				cb.status = StatusClosed
				cb.failures = 0
			}
		}
	case StatusOpen:
		if time.Since(cb.lastFailureAt) > cb.resetTimeout {
			cb.status = StatusHalfOpen
			goto checkStatus
		} else {
			err = ErrorCircuitBreakerOpen
		}
	default:
		err = fmt.Errorf("invalid circuit-breaker status: %v", cb.status)
	}

	if cb.onStatusChange != nil && initialStatus != cb.status {
		go cb.onStatusChange(initialStatus, cb.status)
	}
	return val, err
}

// Retry calls the provided function fn up to maxRetries times with exponential backoff.
func Retry[T any](fn func(attempt uint) (T, error), maxRetries uint, initialDelay time.Duration) (T, error) {
	var val T
	var err error

	for i := range maxRetries {
		val, err = fn(i)
		if err == nil {
			return val, nil
		}
		if i < maxRetries-1 {
			time.Sleep(initialDelay << i)
		}
	}
	return val, err
}
