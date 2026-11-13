package cmd

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Panandika/notion-tui/internal/notion"
)

func TestAddParent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		db, page, defaultDB string
		want                notion.PageParent
		wantErr             string
	}{
		{name: "named database", db: "db-1", defaultDB: "db-0", want: notion.PageParent{DatabaseID: "db-1"}},
		{name: "named page wins over the default database", page: "page-1", defaultDB: "db-0", want: notion.PageParent{PageID: "page-1"}},
		{name: "default database", defaultDB: "db-0", want: notion.PageParent{DatabaseID: "db-0"}},
		{name: "both named", db: "db-1", page: "page-1", wantErr: "not both"},
		{name: "nowhere", wantErr: "nowhere to add to"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := addParent(tt.db, tt.page, tt.defaultDB)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAddText(t *testing.T) {
	t.Parallel()

	text, err := addText([]string{"Call", "the dentist"}, strings.NewReader("ignored"))
	require.NoError(t, err)
	assert.Equal(t, "Call the dentist", text)

	text, err = addText(nil, strings.NewReader("from a pipe\nsecond line\n"))
	require.NoError(t, err)
	assert.Equal(t, "from a pipe\nsecond line\n", text)

	text, err = addText([]string{"-"}, strings.NewReader("dash means stdin"))
	require.NoError(t, err)
	assert.Equal(t, "dash means stdin", text)

	_, err = addText([]string{"  "}, strings.NewReader(""))
	assert.ErrorContains(t, err, "empty")
	_, err = addText(nil, strings.NewReader(" \n"))
	assert.ErrorContains(t, err, "empty")
	_, err = addText(nil, strings.NewReader(strings.Repeat("x", addInputLimit+1)))
	assert.ErrorContains(t, err, "larger than 1 MB")
}

func TestAddMarkdown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name, text, wantTitle, wantContent string
	}{
		{"one line", "Call the dentist", "Call the dentist", ""},
		{"title and body", "Idea\n\n- first\n- second\n", "Idea", "- first\n- second"},
		{"heading marker is not doubled", "# Already a heading\nbody", "Already a heading", "body"},
		{"markdown characters in the title stay literal", "fix *that* [bug]_now", "fix *that* [bug]_now", ""},
		{"a hash that is not a heading stays", "#42 is broken", "#42 is broken", ""},
		{"windows line ends", "Title\r\nbody\r\n", "Title", "body"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			title, content := notion.SplitTitle(addMarkdown(tt.text))
			assert.Equal(t, tt.wantTitle, title)
			assert.Equal(t, tt.wantContent, content)
		})
	}
}
