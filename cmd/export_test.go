package cmd

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jomei/notionapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Panandika/notion-tui/internal/notion"
)

type fakeExporter struct {
	title    string
	markdown *notion.PageMarkdown
	pageErr  error
	bodyErr  error
}

func (f fakeExporter) GetPage(context.Context, string) (*notionapi.Page, error) {
	if f.pageErr != nil {
		return nil, f.pageErr
	}
	return &notionapi.Page{Properties: notionapi.Properties{
		"title": &notionapi.TitleProperty{Title: []notionapi.RichText{{PlainText: f.title}}},
	}}, nil
}

func (f fakeExporter) GetPageMarkdown(context.Context, string) (*notion.PageMarkdown, error) {
	if f.bodyErr != nil {
		return nil, f.bodyErr
	}
	return f.markdown, nil
}

func TestExportMarkdown(t *testing.T) {
	t.Parallel()

	exporter := fakeExporter{title: "Roadmap", markdown: &notion.PageMarkdown{Markdown: "Ship <b>it</b>", Truncated: true}}

	markdown, truncated, err := exportMarkdown(context.Background(), exporter, "page-1")

	require.NoError(t, err)
	assert.Equal(t, "# Roadmap\n\nShip **it**\n", markdown)
	assert.True(t, truncated)
}

func TestExportMarkdown_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		exporter fakeExporter
	}{
		{name: "page lookup fails", exporter: fakeExporter{pageErr: errors.New("not found")}},
		{name: "content fetch fails", exporter: fakeExporter{title: "T", bodyErr: errors.New("forbidden")}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, _, err := exportMarkdown(context.Background(), tt.exporter, "page-1")
			assert.Error(t, err)
		})
	}
}

func TestWriteExportOutput(t *testing.T) {
	t.Parallel()

	t.Run("stdout by default", func(t *testing.T) {
		t.Parallel()
		var stdout bytes.Buffer

		require.NoError(t, writeExportOutput(&stdout, "", "# T\n"))

		assert.Equal(t, "# T\n", stdout.String())
	})

	t.Run("file when a path is given", func(t *testing.T) {
		t.Parallel()
		var stdout bytes.Buffer
		path := filepath.Join(t.TempDir(), "out.md")

		require.NoError(t, writeExportOutput(&stdout, path, "# T\n"))

		raw, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, "# T\n", string(raw))
		assert.Empty(t, stdout.String())
	})

	t.Run("unwritable path is an error", func(t *testing.T) {
		t.Parallel()
		err := writeExportOutput(&bytes.Buffer{}, filepath.Join(t.TempDir(), "missing-dir", "out.md"), "x")
		assert.Error(t, err)
	})
}
