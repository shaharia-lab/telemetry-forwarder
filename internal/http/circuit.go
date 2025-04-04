package http

import (
	"log"
	"sync"
	"time"
)

const (
	circuitClosed = iota
	circuitOpen
	circuitHalfOpen
)

type CircuitBreaker struct {
	state       int
	failCount   int
	lastFailure time.Time
	mutex       sync.RWMutex
	timeout     time.Duration
	maxFailures int
	name        string
}

func NewCircuitBreaker(name string, maxFailures int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:       circuitClosed,
		maxFailures: maxFailures,
		timeout:     timeout,
		name:        name,
	}
}

func (cb *CircuitBreaker) IsAllowed() bool {
	cb.mutex.RLock()
	if cb.state == circuitClosed {
		cb.mutex.RUnlock()
		return true
	}

	if cb.state == circuitOpen && time.Since(cb.lastFailure) > cb.timeout {
		cb.mutex.RUnlock()
		cb.mutex.Lock()
		defer cb.mutex.Unlock()
		cb.state = circuitHalfOpen
		log.Printf("%s circuit half-open: testing API availability", cb.name)
		return true
	}

	allowed := cb.state == circuitHalfOpen
	cb.mutex.RUnlock()
	return allowed
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	if cb.state == circuitHalfOpen {
		cb.state = circuitClosed
		cb.failCount = 0
		log.Printf("%s circuit closed: API is back to normal", cb.name)
	}
}

func (cb *CircuitBreaker) RecordFailure() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.lastFailure = time.Now()

	switch cb.state {
	case circuitClosed:
		cb.failCount++
		if cb.failCount >= cb.maxFailures {
			cb.state = circuitOpen
			log.Printf("%s circuit open: stopping requests temporarily", cb.name)
		}
	case circuitHalfOpen:
		cb.state = circuitOpen
		log.Printf("%s circuit reopened: API still having issues", cb.name)
	}
}
