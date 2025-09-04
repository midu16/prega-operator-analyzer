package pkg

import (
	"fmt"
	"time"
)

// ErrorType represents different types of errors that can occur
type ErrorType string

const (
	ErrorTypeNetwork     ErrorType = "NETWORK_ERROR"
	ErrorTypeGit         ErrorType = "GIT_ERROR"
	ErrorTypeParsing     ErrorType = "PARSING_ERROR"
	ErrorTypeFileSystem  ErrorType = "FILESYSTEM_ERROR"
	ErrorTypeValidation  ErrorType = "VALIDATION_ERROR"
	ErrorTypeTimeout     ErrorType = "TIMEOUT_ERROR"
	ErrorTypeUnknown     ErrorType = "UNKNOWN_ERROR"
)

// AnalyzerError represents a structured error with context
type AnalyzerError struct {
	Type      ErrorType
	Message   string
	Context   map[string]interface{}
	Timestamp time.Time
	Err       error
}

// Error implements the error interface
func (e *AnalyzerError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Type, e.Message)
}

// Unwrap returns the underlying error
func (e *AnalyzerError) Unwrap() error {
	return e.Err
}

// NewAnalyzerError creates a new AnalyzerError
func NewAnalyzerError(errorType ErrorType, message string, err error) *AnalyzerError {
	return &AnalyzerError{
		Type:      errorType,
		Message:   message,
		Context:   make(map[string]interface{}),
		Timestamp: time.Now(),
		Err:       err,
	}
}

// WithContext adds context information to the error
func (e *AnalyzerError) WithContext(key string, value interface{}) *AnalyzerError {
	e.Context[key] = value
	return e
}

// IsRetryable determines if an error is retryable
func (e *AnalyzerError) IsRetryable() bool {
	switch e.Type {
	case ErrorTypeNetwork, ErrorTypeTimeout:
		return true
	case ErrorTypeGit:
		// Some git errors are retryable (network issues), others are not
		return e.Message == "failed to clone repository" || e.Message == "failed to fetch"
	default:
		return false
	}
}

// GetRetryDelay returns the suggested retry delay for retryable errors
func (e *AnalyzerError) GetRetryDelay() time.Duration {
	switch e.Type {
	case ErrorTypeNetwork:
		return 5 * time.Second
	case ErrorTypeTimeout:
		return 10 * time.Second
	case ErrorTypeGit:
		return 3 * time.Second
	default:
		return 0
	}
}

// ErrorHandler handles errors with retry logic and logging
type ErrorHandler struct {
	MaxRetries int
	Logger     interface {
		Errorf(format string, args ...interface{})
		Warnf(format string, args ...interface{})
		Infof(format string, args ...interface{})
	}
}

// NewErrorHandler creates a new ErrorHandler
func NewErrorHandler(maxRetries int, logger interface {
	Errorf(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Infof(format string, args ...interface{})
}) *ErrorHandler {
	return &ErrorHandler{
		MaxRetries: maxRetries,
		Logger:     logger,
	}
}

// HandleWithRetry executes a function with retry logic for retryable errors
func (eh *ErrorHandler) HandleWithRetry(operation func() error, operationName string) error {
	var lastErr error
	
	for attempt := 0; attempt <= eh.MaxRetries; attempt++ {
		err := operation()
		if err == nil {
			if attempt > 0 {
				eh.Logger.Infof("Operation '%s' succeeded after %d retries", operationName, attempt)
			}
			return nil
		}
		
		lastErr = err
		
		// Check if error is retryable
		var analyzerErr *AnalyzerError
		if ae, ok := err.(*AnalyzerError); ok {
			analyzerErr = ae
		} else {
			// Wrap unknown errors
			analyzerErr = NewAnalyzerError(ErrorTypeUnknown, "Unknown error occurred", err)
		}
		
		if !analyzerErr.IsRetryable() || attempt == eh.MaxRetries {
			break
		}
		
		delay := analyzerErr.GetRetryDelay()
		eh.Logger.Warnf("Operation '%s' failed (attempt %d/%d): %v. Retrying in %v...", 
			operationName, attempt+1, eh.MaxRetries+1, err, delay)
		
		time.Sleep(delay)
	}
	
	// Log final error
	eh.Logger.Errorf("Operation '%s' failed after %d attempts: %v", operationName, eh.MaxRetries+1, lastErr)
	return lastErr
}

// WrapError wraps a standard error with context
func WrapError(err error, errorType ErrorType, message string, context map[string]interface{}) *AnalyzerError {
	analyzerErr := NewAnalyzerError(errorType, message, err)
	for k, v := range context {
		analyzerErr.WithContext(k, v)
	}
	return analyzerErr
}