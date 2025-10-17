package indexer

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/smart-contract-event-indexer/shared/utils"
)

// ErrorType represents the type of error
type ErrorType int

const (
	// ErrorTypeTransient represents a temporary error that can be retried
	ErrorTypeTransient ErrorType = iota
	// ErrorTypePermanent represents a permanent error that should not be retried
	ErrorTypePermanent
	// ErrorTypeRateLimit represents a rate limit error
	ErrorTypeRateLimit
	// ErrorTypeNetwork represents a network connectivity error
	ErrorTypeNetwork
)

// RetryPolicy defines the retry behavior
type RetryPolicy struct {
	MaxAttempts     int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	BackoffFactor   float64
	RetriableErrors []string
}

// DefaultRetryPolicy returns the default retry policy
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxAttempts:   3,
		InitialDelay:  1 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		RetriableErrors: []string{
			"connection refused",
			"connection reset",
			"timeout",
			"temporary failure",
			"too many requests",
			"rate limit",
			"EOF",
			"broken pipe",
		},
	}
}

// ErrorClassifier classifies and handles errors
type ErrorClassifier struct {
	policy *RetryPolicy
	logger utils.Logger
}

// NewErrorClassifier creates a new error classifier
func NewErrorClassifier(policy *RetryPolicy, logger utils.Logger) *ErrorClassifier {
	if policy == nil {
		policy = DefaultRetryPolicy()
	}
	return &ErrorClassifier{
		policy: policy,
		logger: logger,
	}
}

// ClassifyError determines the type of error
func (c *ErrorClassifier) ClassifyError(err error) ErrorType {
	if err == nil {
		return ErrorTypePermanent
	}
	
	errStr := strings.ToLower(err.Error())
	
	// Check for rate limit errors
	if strings.Contains(errStr, "rate limit") || 
	   strings.Contains(errStr, "too many requests") ||
	   strings.Contains(errStr, "429") {
		return ErrorTypeRateLimit
	}
	
	// Check for network errors
	if strings.Contains(errStr, "connection") ||
	   strings.Contains(errStr, "network") ||
	   strings.Contains(errStr, "dial") ||
	   strings.Contains(errStr, "timeout") {
		return ErrorTypeNetwork
	}
	
	// Check for transient errors
	for _, retriable := range c.policy.RetriableErrors {
		if strings.Contains(errStr, strings.ToLower(retriable)) {
			return ErrorTypeTransient
		}
	}
	
	// Default to permanent if we can't classify it
	return ErrorTypePermanent
}

// IsRetriable determines if an error should be retried
func (c *ErrorClassifier) IsRetriable(err error) bool {
	errType := c.ClassifyError(err)
	return errType == ErrorTypeTransient || 
	       errType == ErrorTypeRateLimit || 
	       errType == ErrorTypeNetwork
}

// GetRetryDelay calculates the delay before the next retry
func (c *ErrorClassifier) GetRetryDelay(attempt int, errType ErrorType) time.Duration {
	baseDelay := c.policy.InitialDelay
	
	// Apply exponential backoff
	delay := time.Duration(float64(baseDelay) * (1 << uint(attempt-1)))
	
	// Apply backoff factor
	delay = time.Duration(float64(delay) * c.policy.BackoffFactor)
	
	// Cap at max delay
	if delay > c.policy.MaxDelay {
		delay = c.policy.MaxDelay
	}
	
	// Apply special handling for rate limits
	if errType == ErrorTypeRateLimit {
		delay = delay * 2 // Wait longer for rate limits
	}
	
	return delay
}

// ExecuteWithRetry executes a function with retry logic
func (c *ErrorClassifier) ExecuteWithRetry(
	ctx context.Context,
	operation string,
	fn func() error,
) error {
	var lastErr error
	
	for attempt := 1; attempt <= c.policy.MaxAttempts; attempt++ {
		// Execute the function
		err := fn()
		
		if err == nil {
			// Success
			if attempt > 1 {
				c.logger.WithFields(map[string]interface{}{
					"operation": operation,
					"attempt":   attempt,
				}).Info("Operation succeeded after retry")
			}
			return nil
		}
		
		lastErr = err
		
		// Classify the error
		errType := c.ClassifyError(err)
		
		c.logger.WithFields(map[string]interface{}{
			"operation":  operation,
			"attempt":    attempt,
			"error_type": c.getErrorTypeName(errType),
			"error":      err.Error(),
		}).Warn("Operation failed")
		
		// Check if we should retry
		if !c.IsRetriable(err) {
			c.logger.WithField("operation", operation).Error("Error is not retriable, giving up")
			return fmt.Errorf("permanent error: %w", err)
		}
		
		// Check if we've exhausted attempts
		if attempt >= c.policy.MaxAttempts {
			c.logger.WithFields(map[string]interface{}{
				"operation": operation,
				"attempts":  attempt,
			}).Error("Max retry attempts reached")
			return fmt.Errorf("max retries exceeded: %w", lastErr)
		}
		
		// Calculate retry delay
		delay := c.GetRetryDelay(attempt, errType)
		
		c.logger.WithFields(map[string]interface{}{
			"operation":    operation,
			"next_attempt": attempt + 1,
			"delay":        delay.String(),
		}).Info("Retrying operation")
		
		// Wait before retrying
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled during retry: %w", ctx.Err())
		case <-time.After(delay):
			// Continue to next attempt
		}
	}
	
	return fmt.Errorf("operation failed after %d attempts: %w", c.policy.MaxAttempts, lastErr)
}

// getErrorTypeName returns a human-readable error type name
func (c *ErrorClassifier) getErrorTypeName(errType ErrorType) string {
	switch errType {
	case ErrorTypeTransient:
		return "transient"
	case ErrorTypePermanent:
		return "permanent"
	case ErrorTypeRateLimit:
		return "rate_limit"
	case ErrorTypeNetwork:
		return "network"
	default:
		return "unknown"
	}
}

// CircuitBreaker implements a circuit breaker pattern
type CircuitBreaker struct {
	maxFailures    int
	resetTimeout   time.Duration
	failures       int
	lastFailure    time.Time
	state          CircuitState
	logger         *utils.Logger
	mu             sync.RWMutex
}

// CircuitState represents the state of a circuit breaker
type CircuitState int

const (
	// CircuitClosed means the circuit is closed (normal operation)
	CircuitClosed CircuitState = iota
	// CircuitOpen means the circuit is open (rejecting requests)
	CircuitOpen
	// CircuitHalfOpen means the circuit is testing if it can close
	CircuitHalfOpen
)

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(maxFailures int, resetTimeout time.Duration, logger utils.Logger) *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures:  maxFailures,
		resetTimeout: resetTimeout,
		state:        CircuitClosed,
		logger:       logger,
	}
}

// Execute executes a function through the circuit breaker
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	// Check circuit state
	if !cb.canExecute() {
		return fmt.Errorf("circuit breaker is open")
	}
	
	// Execute the function
	err := fn()
	
	// Update circuit state based on result
	cb.recordResult(err)
	
	return err
}

// canExecute checks if the circuit allows execution
func (cb *CircuitBreaker) canExecute() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	
	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		// Check if we should transition to half-open
		if time.Since(cb.lastFailure) > cb.resetTimeout {
			cb.mu.RUnlock()
			cb.mu.Lock()
			cb.state = CircuitHalfOpen
			cb.logger.Info("Circuit breaker transitioning to half-open")
			cb.mu.Unlock()
			cb.mu.RLock()
			return true
		}
		return false
	case CircuitHalfOpen:
		return true
	default:
		return false
	}
}

// recordResult records the result of an execution
func (cb *CircuitBreaker) recordResult(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	if err != nil {
		cb.failures++
		cb.lastFailure = time.Now()
		
		if cb.failures >= cb.maxFailures {
			cb.state = CircuitOpen
			cb.logger.WithField("failures", cb.failures).Warn("Circuit breaker opened")
		}
	} else {
		// Success
		if cb.state == CircuitHalfOpen {
			cb.state = CircuitClosed
			cb.failures = 0
			cb.logger.Info("Circuit breaker closed after successful test")
		} else {
			cb.failures = 0
		}
	}
}

// GetState returns the current circuit state
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Reset resets the circuit breaker
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	cb.state = CircuitClosed
	cb.failures = 0
	cb.logger.Info("Circuit breaker reset")
}

// GetStats returns circuit breaker statistics
func (cb *CircuitBreaker) GetStats() map[string]interface{} {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	
	return map[string]interface{}{
		"state":         cb.getStateName(),
		"failures":      cb.failures,
		"max_failures":  cb.maxFailures,
		"last_failure":  cb.lastFailure,
		"reset_timeout": cb.resetTimeout.String(),
	}
}

// getStateName returns a human-readable state name
func (cb *CircuitBreaker) getStateName() string {
	switch cb.state {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half_open"
	default:
		return "unknown"
	}
}

