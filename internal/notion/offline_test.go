package notion

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/jomei/notionapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

func TestIsOffline(t *testing.T) {
	t.Parallel()

	// A server that is gone: connecting to it is refused
	gone := httptest.NewServer(http.NotFoundHandler())
	goneURL := gone.URL
	gone.Close()
	client := NewClient("test-token")
	client.baseURL = goneURL
	client.limiter = rate.NewLimiter(rate.Inf, 1)

	_, err := client.GetPageMarkdown(context.Background(), "page-1")
	require.Error(t, err)
	assert.True(t, IsOffline(err), "no answer at all: %v", err)
	assert.True(t, IsOffline(fmt.Errorf("fetch content: %w", err)), "also when wrapped on the way up")

	// A server that answers with an error is reachable
	answering, _ := newMarkdownTestClient(t, func(w http.ResponseWriter, _ int) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"object":"error","status":404,"code":"object_not_found","message":"gone"}`))
	})
	_, err = answering.GetPageMarkdown(context.Background(), "page-1")
	require.Error(t, err)
	assert.False(t, IsOffline(err), "Notion said no: %v", err)

	assert.True(t, IsOffline(context.DeadlineExceeded), "a timeout counts as not reachable")
	assert.False(t, IsOffline(errors.New("validation failed")))
	assert.False(t, IsOffline(nil))
}

func TestIsOffline_WhatCountsAndWhatDoesNot(t *testing.T) {
	t.Parallel()

	wrap := func(err error) error {
		return fmt.Errorf("fetch page: %w", &url.Error{Op: "Get", URL: "https://api.notion.com", Err: err})
	}

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"connection refused", wrap(&net.OpError{Op: "dial", Err: errors.New("connection refused")}), true},
		{"name not resolved", wrap(&net.DNSError{Err: "no such host", Name: "api.notion.com"}), true},
		{"Notion is down", &notionapi.Error{Status: 503, Message: "service unavailable"}, true},
		{"Notion refuses", &notionapi.Error{Status: 404, Message: "not found"}, false},
		{"rate limited", &notionapi.Error{Status: 429, Message: "slow down"}, false},
		{"certificate not trusted", wrap(x509.UnknownAuthorityError{}), false},
		{"certificate for another host", wrap(x509.HostnameError{Host: "api.notion.com"}), false},
		{"certificate check failed", wrap(&tls.CertificateVerificationError{Err: errors.New("bad chain")}), false},
		{"cancelled", wrap(context.Canceled), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, IsOffline(tt.err))
		})
	}
}
