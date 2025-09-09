package pkg

import (
	"errors"
	"testing"
	"time"
)

func TestAnalyzerError(t *testing.T) {
	tests := []struct {
		name        string
		errorType   ErrorType
		message     string
		err         error
		expectRetry bool
		expectDelay time.Duration
	}{
		{
			name:        "network error",
			errorType:   ErrorTypeNetwork,
			message:     "connection failed",
			err:         errors.New("connection timeout"),
			expectRetry: true,
			expectDelay: 5 * time.Second,
		},
		{
			name:        "git clone error",
			errorType:   ErrorTypeGit,
			message:     "failed to clone repository",
			err:         errors.New("git clone failed"),
			expectRetry: true,
			expectDelay: 3 * time.Second,
		},
		{
			name:        "parsing error",
			errorType:   ErrorTypeParsing,
			message:     "invalid JSON",
			err:         errors.New("unexpected token"),
			expectRetry: false,
			expectDelay: 0,
		},
		{
			name:        "filesystem error",
			errorType:   ErrorTypeFileSystem,
			message:     "file not found",
			err:         errors.New("no such file"),
			expectRetry: false,
			expectDelay: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzerErr := NewAnalyzerError(tt.errorType, tt.message, tt.err)

			// Test error message
			expectedMsg := "[" + string(tt.errorType) + "] " + tt.message + ": " + tt.err.Error()
			if analyzerErr.Error() != expectedMsg {
				t.Errorf("Expected error message %s, got %s", expectedMsg, analyzerErr.Error())
			}

			// Test unwrap
			if analyzerErr.Unwrap() != tt.err {
				t.Errorf("Expected unwrapped error %v, got %v", tt.err, analyzerErr.Unwrap())
			}

			// Test retry logic
			if analyzerErr.IsRetryable() != tt.expectRetry {
				t.Errorf("Expected retryable %v, got %v", tt.expectRetry, analyzerErr.IsRetryable())
			}

			// Test retry delay
			if analyzerErr.GetRetryDelay() != tt.expectDelay {
				t.Errorf("Expected delay %v, got %v", tt.expectDelay, analyzerErr.GetRetryDelay())
			}
		})
	}
}

func TestAnalyzerErrorWithContext(t *testing.T) {
	err := NewAnalyzerError(ErrorTypeNetwork, "test error", errors.New("underlying error"))
	err = err.WithContext("key1", "value1").WithContext("key2", 42)

	if err.Context["key1"] != "value1" {
		t.Errorf("Expected context key1 to be 'value1', got %v", err.Context["key1"])
	}

	if err.Context["key2"] != 42 {
		t.Errorf("Expected context key2 to be 42, got %v", err.Context["key2"])
	}
}

func TestWrapError(t *testing.T) {
	originalErr := errors.New("original error")
	context := map[string]interface{}{
		"file": "test.json",
		"line": 42,
	}

	wrappedErr := WrapError(originalErr, ErrorTypeParsing, "failed to parse", context)

	if wrappedErr.Type != ErrorTypeParsing {
		t.Errorf("Expected error type %s, got %s", ErrorTypeParsing, wrappedErr.Type)
	}

	if wrappedErr.Message != "failed to parse" {
		t.Errorf("Expected message 'failed to parse', got %s", wrappedErr.Message)
	}

	if wrappedErr.Unwrap() != originalErr {
		t.Errorf("Expected unwrapped error %v, got %v", originalErr, wrappedErr.Unwrap())
	}

	if wrappedErr.Context["file"] != "test.json" {
		t.Errorf("Expected context file to be 'test.json', got %v", wrappedErr.Context["file"])
	}

	if wrappedErr.Context["line"] != 42 {
		t.Errorf("Expected context line to be 42, got %v", wrappedErr.Context["line"])
	}
}

func TestErrorHandler(t *testing.T) {
	// Mock logger
	mockLogger := &mockLogger{}

	errorHandler := NewErrorHandler(1, mockLogger) // Reduced retries for faster tests

	tests := []struct {
		name           string
		operation      func() error
		expectSuccess  bool
	}{
		{
			name: "successful operation",
			operation: func() error {
				return nil
			},
			expectSuccess: true,
		},
		{
			name: "non-retryable error",
			operation: func() error {
				return NewAnalyzerError(ErrorTypeParsing, "invalid JSON", errors.New("parse error"))
			},
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLogger.reset()
			err := errorHandler.HandleWithRetry(tt.operation, tt.name)

			if tt.expectSuccess && err != nil {
				t.Errorf("Expected success but got error: %v", err)
			}

			if !tt.expectSuccess && err == nil {
				t.Errorf("Expected error but got success")
			}
		})
	}
}

// Mock logger for testing
type mockLogger struct {
	retryCount int
}

func (m *mockLogger) Errorf(format string, args ...interface{}) {}
func (m *mockLogger) Warnf(format string, args ...interface{}) {
	m.retryCount++
}
func (m *mockLogger) Infof(format string, args ...interface{}) {}

func (m *mockLogger) reset() {
	m.retryCount = 0
}
