package notion

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

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
