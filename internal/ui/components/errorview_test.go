package components

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"

	"github.com/jomei/notionapi"
	"github.com/stretchr/testify/assert"
)

// mockNetError implements net.Error for testing network errors.
type mockNetError struct {
	msg       string
	timeout   bool
	temporary bool
}

func (e *mockNetError) Error() string   { return e.msg }
func (e *mockNetError) Timeout() bool   { return e.timeout }
func (e *mockNetError) Temporary() bool { return e.temporary }

func TestNewErrorView(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		expectedMsg string
		expectRetry bool
	}{
		{
			name:        "nil error",
			err:         nil,
			expectedMsg: "An unknown error occurred",
			expectRetry: false,
		},
		{
			name:        "generic error",
			err:         errors.New("something went wrong"),
			expectedMsg: "Something went wrong",
			expectRetry: true,
		},
		{
			name:        "network timeout error",
			err:         &mockNetError{msg: "connection timeout", timeout: true},
			expectedMsg: "Can't connect to Notion",
			expectRetry: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ev := NewErrorView(NewErrorViewInput{
				Err:        tt.err,
				Width:      80,
				Height:     24,
				ShowBorder: true,
			})

			assert.Equal(t, tt.expectedMsg, ev.Message())
			assert.Equal(t, tt.expectRetry, ev.IsRetryable())
		})
	}
}

func TestErrorView_ClassifyHTTPError(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		expectedType ErrorType
		expectedMsg  string
		expectRetry  bool
	}{
		{
			name:         "401 unauthorized",
			statusCode:   http.StatusUnauthorized,
			expectedType: ErrorTypeAuth,
			expectedMsg:  "Invalid Notion token",
			expectRetry:  false,
		},
		{
			name:         "403 forbidden",
			statusCode:   http.StatusForbidden,
			expectedType: ErrorTypeAuth,
			expectedMsg:  "Access forbidden",
			expectRetry:  false,
		},
		{
			name:         "404 not found",
			statusCode:   http.StatusNotFound,
			expectedType: ErrorTypeNotFound,
			expectedMsg:  "Page not found",
			expectRetry:  false,
		},
		{
			name:         "429 rate limit",
			statusCode:   http.StatusTooManyRequests,
			expectedType: ErrorTypeRateLimit,
			expectedMsg:  "Rate limit exceeded",
			expectRetry:  false,
		},
		{
			name:         "400 bad request",
			statusCode:   http.StatusBadRequest,
			expectedType: ErrorTypeValidation,
			expectedMsg:  "Invalid request",
			expectRetry:  false,
		},
		{
			name:         "500 internal server error",
			statusCode:   http.StatusInternalServerError,
			expectedType: ErrorTypeServer,
			expectedMsg:  "Notion server error",
			expectRetry:  true,
		},
		{
			name:         "502 bad gateway",
			statusCode:   http.StatusBadGateway,
			expectedType: ErrorTypeServer,
			expectedMsg:  "Notion server error",
			expectRetry:  true,
		},
		{
			name:         "503 service unavailable",
			statusCode:   http.StatusServiceUnavailable,
			expectedType: ErrorTypeServer,
			expectedMsg:  "Notion server error",
			expectRetry:  true,
		},
		{
			name:         "504 gateway timeout",
			statusCode:   http.StatusGatewayTimeout,
			expectedType: ErrorTypeServer,
			expectedMsg:  "Notion server error",
			expectRetry:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notionErr := &notionapi.Error{
				Status:  tt.statusCode,
				Message: "Test error message",
			}

			ev := NewErrorView(NewErrorViewInput{
				Err:        notionErr,
				Width:      80,
				Height:     24,
				ShowBorder: false,
			})

			assert.Equal(t, tt.expectedType, ev.ErrorType())
			assert.Equal(t, tt.expectedMsg, ev.Message())
			assert.Equal(t, tt.expectRetry, ev.IsRetryable())
		})
	}
}

func TestErrorView_ClassifyErrorByString(t *testing.T) {
	tests := []struct {
		name         string
		errMsg       string
		expectedType ErrorType
		expectedMsg  string
		expectRetry  bool
	}{
		{
			name:         "401 in error string",
			errMsg:       "request failed with 401 unauthorized",
			expectedType: ErrorTypeAuth,
			expectedMsg:  "Invalid Notion token",
			expectRetry:  false,
		},
		{
			name:         "404 in error string",
			errMsg:       "page not found",
			expectedType: ErrorTypeNotFound,
			expectedMsg:  "Page not found",
			expectRetry:  false,
		},
		{
			name:         "429 in error string",
			errMsg:       "rate limit exceeded",
			expectedType: ErrorTypeRateLimit,
			expectedMsg:  "Rate limit exceeded",
			expectRetry:  false,
		},
		{
			name:         "500 in error string",
			errMsg:       "server error 500",
			expectedType: ErrorTypeServer,
			expectedMsg:  "Notion server error",
			expectRetry:  true,
		},
		{
			name:         "timeout error",
			errMsg:       "request timeout",
			expectedType: ErrorTypeNetwork,
			expectedMsg:  "Request timed out",
			expectRetry:  true,
		},
		{
			name:         "connection refused",
			errMsg:       "connection refused",
			expectedType: ErrorTypeNetwork,
			expectedMsg:  "Can't reach Notion",
			expectRetry:  true,
		},
		{
			name:         "no such host",
			errMsg:       "no such host",
			expectedType: ErrorTypeNetwork,
			expectedMsg:  "Can't reach Notion",
			expectRetry:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.New(tt.errMsg)

			ev := NewErrorView(NewErrorViewInput{
				Err:        err,
				Width:      80,
				Height:     24,
				ShowBorder: false,
			})

			assert.Equal(t, tt.expectedType, ev.ErrorType())
			assert.Equal(t, tt.expectedMsg, ev.Message())
			assert.Equal(t, tt.expectRetry, ev.IsRetryable())
		})
	}
}

func TestErrorView_View(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		showBorder     bool
		width          int
		height         int
		expectContains []string
	}{
		{
			name:       "auth error with border",
			err:        &notionapi.Error{Status: 401},
			showBorder: true,
			width:      80,
			height:     24,
			expectContains: []string{
				"Invalid Notion token",
				"esc: Go Back",
			},
		},
		{
			name:       "network error without border",
			err:        &mockNetError{msg: "connection timeout", timeout: true},
			showBorder: false,
			width:      80,
			height:     24,
			expectContains: []string{
				"Can't connect to Notion",
				"r: Retry",
				"esc: Go Back",
			},
		},
		{
			name:       "not found error",
			err:        &notionapi.Error{Status: 404},
			showBorder: true,
			width:      80,
			height:     24,
			expectContains: []string{
				"Page not found",
				"esc: Go Back",
			},
		},
		{
			name:       "rate limit error",
			err:        &notionapi.Error{Status: 429},
			showBorder: false,
			width:      80,
			height:     24,
			expectContains: []string{
				"Rate limit exceeded",
			},
		},
		{
			name:       "server error with retry",
			err:        &notionapi.Error{Status: 500},
			showBorder: false,
			width:      80,
			height:     24,
			expectContains: []string{
				"Notion server error",
				"r: Retry",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ev := NewErrorView(NewErrorViewInput{
				Err:        tt.err,
				Width:      tt.width,
				Height:     tt.height,
				ShowBorder: tt.showBorder,
			})

			view := ev.View()

			// Check that view is not empty
			assert.NotEmpty(t, view)

			// Check for expected content
			for _, expected := range tt.expectContains {
				assert.Contains(t, view, expected,
					fmt.Sprintf("View should contain '%s'", expected))
			}
		})
	}
}

func TestErrorView_GetErrorIcon(t *testing.T) {
	tests := []struct {
		name         string
		errorType    ErrorType
		expectedIcon string
	}{
		{
			name:         "network error icon",
			errorType:    ErrorTypeNetwork,
			expectedIcon: "NETWORK ERROR",
		},
		{
			name:         "auth error icon",
			errorType:    ErrorTypeAuth,
			expectedIcon: "AUTHENTICATION ERROR",
		},
		{
			name:         "not found icon",
			errorType:    ErrorTypeNotFound,
			expectedIcon: "NOT FOUND",
		},
		{
			name:         "rate limit icon",
			errorType:    ErrorTypeRateLimit,
			expectedIcon: "RATE LIMIT",
		},
		{
			name:         "server error icon",
			errorType:    ErrorTypeServer,
			expectedIcon: "SERVER ERROR",
		},
		{
			name:         "validation error icon",
			errorType:    ErrorTypeValidation,
			expectedIcon: "VALIDATION ERROR",
		},
		{
			name:         "unknown error icon",
			errorType:    ErrorTypeUnknown,
			expectedIcon: "ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ev := ErrorView{
				errorType: tt.errorType,
				styles:    DefaultErrorViewStyles(),
			}

			icon := ev.getErrorIcon()
			assert.Contains(t, icon, tt.expectedIcon)
		})
	}
}

func TestErrorView_SetSize(t *testing.T) {
	ev := NewErrorView(NewErrorViewInput{
		Err:        errors.New("test error"),
		Width:      80,
		Height:     24,
		ShowBorder: false,
	})

	assert.Equal(t, 80, ev.width)
	assert.Equal(t, 24, ev.height)

	ev.SetSize(100, 30)

	assert.Equal(t, 100, ev.width)
	assert.Equal(t, 30, ev.height)
}

func TestErrorView_SetError(t *testing.T) {
	ev := NewErrorView(NewErrorViewInput{
		Err:        errors.New("first error"),
		Width:      80,
		Height:     24,
		ShowBorder: false,
	})

	assert.Equal(t, "Something went wrong", ev.Message())

	// Change to auth error
	ev.SetError(&notionapi.Error{Status: 401})

	assert.Equal(t, "Invalid Notion token", ev.Message())
	assert.Equal(t, ErrorTypeAuth, ev.ErrorType())
}

func TestErrorView_Actions(t *testing.T) {
	tests := []struct {
		name            string
		err             error
		expectedActions int
		hasRetry        bool
		hasGoBack       bool
	}{
		{
			name:            "auth error - no retry",
			err:             &notionapi.Error{Status: 401},
			expectedActions: 1,
			hasRetry:        false,
			hasGoBack:       true,
		},
		{
			name:            "network error - has retry",
			err:             &mockNetError{msg: "timeout", timeout: true},
			expectedActions: 2,
			hasRetry:        true,
			hasGoBack:       true,
		},
		{
			name:            "server error - has retry",
			err:             &notionapi.Error{Status: 500},
			expectedActions: 2,
			hasRetry:        true,
			hasGoBack:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ev := NewErrorView(NewErrorViewInput{
				Err:        tt.err,
				Width:      80,
				Height:     24,
				ShowBorder: false,
			})

			actions := ev.Actions()
			assert.Equal(t, tt.expectedActions, len(actions))

			hasRetry := false
			hasGoBack := false
			for _, action := range actions {
				if action.Key == "r" {
					hasRetry = true
				}
				if action.Key == "esc" {
					hasGoBack = true
				}
			}

			assert.Equal(t, tt.hasRetry, hasRetry)
			assert.Equal(t, tt.hasGoBack, hasGoBack)
		})
	}
}

func TestErrorView_Update(t *testing.T) {
	ev := NewErrorView(NewErrorViewInput{
		Err:        errors.New("test error"),
		Width:      80,
		Height:     24,
		ShowBorder: false,
	})

	// Update should return the same error view (currently stateless)
	updated, cmd := ev.Update(nil)

	assert.Equal(t, ev.Message(), updated.Message())
	assert.Nil(t, cmd)
}

func TestErrorView_NetworkError(t *testing.T) {
	// Test with actual net.Error type
	netErr := &net.OpError{
		Op:  "dial",
		Net: "tcp",
		Err: errors.New("connection refused"),
	}

	ev := NewErrorView(NewErrorViewInput{
		Err:        netErr,
		Width:      80,
		Height:     24,
		ShowBorder: false,
	})

	assert.Equal(t, ErrorTypeNetwork, ev.ErrorType())
	assert.Equal(t, "Can't connect to Notion", ev.Message())
	assert.True(t, ev.IsRetryable())
}

func TestErrorView_Context(t *testing.T) {
	tests := []struct {
		name            string
		err             error
		expectedContext string
	}{
		{
			name:            "auth error has context",
			err:             &notionapi.Error{Status: 401},
			expectedContext: "Your API token is invalid or expired",
		},
		{
			name:            "not found has context",
			err:             &notionapi.Error{Status: 404},
			expectedContext: "This page may have been deleted or moved",
		},
		{
			name:            "network error has context",
			err:             &mockNetError{msg: "timeout", timeout: true},
			expectedContext: "Check your internet connection",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ev := NewErrorView(NewErrorViewInput{
				Err:        tt.err,
				Width:      80,
				Height:     24,
				ShowBorder: false,
			})

			context := ev.Context()
			assert.NotEmpty(t, context)
			// Use Contains instead of exact match since context may have more detail
			assert.True(t, strings.Contains(context, tt.expectedContext) ||
				strings.Contains(tt.expectedContext, context))
		})
	}
}
