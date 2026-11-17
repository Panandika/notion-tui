package pages

import (
	"errors"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Panandika/notion-tui/internal/notion"
	"github.com/Panandika/notion-tui/internal/testhelpers"
)

func testComments() []notion.Comment {
	at := time.Date(2026, 11, 17, 9, 0, 0, 0, time.UTC)
	return []notion.Comment{
		{ID: "c1", ThreadID: "t1", Author: "Ada", Text: "Is the budget final?\nSecond line of the question.", Created: at},
		{ID: "c2", ThreadID: "t1", Author: "Grace", Text: "Yes, signed off.", Created: at.Add(time.Hour)},
		{ID: "c3", ThreadID: "t2", Author: "Linus", Text: "Typo in the heading", Created: at.Add(2 * time.Hour)},
	}
}

// openComments returns a loaded page with its comment list open.
func openComments(t *testing.T, comments []notion.Comment) detailFixture {
	t.Helper()
	fx := newDetailFixture(t, "Some text")
	fx.client.CommentsToReturn = comments
	_, cmd := fx.page.Update(keyRune('c'))
	fx.drain(t, cmd)
	require.Equal(t, modeComments, fx.page.mode)
	return fx
}

func typeInto(fx detailFixture, text string) {
	for _, r := range text {
		fx.page.Update(keyRune(r))
	}
}

func TestComments_List(t *testing.T) {
	t.Parallel()

	fx := openComments(t, testComments())
	view := fx.page.View()

	assert.Contains(t, view, "Ada · ")
	assert.Contains(t, view, "↳ Grace", "a reply sits under the comment it answers")
	assert.NotContains(t, view, "↳ Linus", "a new thread is not a reply")
	assert.Contains(t, view, "Is the budget final? Second line", "the row shows the comment on one line")
	assert.Contains(t, view, "Ada wrote:", "the selected comment is shown in full")
	assert.True(t, fx.page.InMode())

	fx.page.Update(keyRune('j'))
	assert.Contains(t, fx.page.View(), "Grace wrote:")

	fx.page.Update(tea.KeyMsg{Type: tea.KeyEsc})
	assert.Equal(t, modeView, fx.page.mode)
}

func TestComments_NoneAndFailure(t *testing.T) {
	t.Parallel()

	t.Run("a page without comments offers to add one", func(t *testing.T) {
		t.Parallel()
		fx := openComments(t, nil)
		assert.Contains(t, fx.page.View(), "No open comments on this page. Press a to add one.")

		fx.page.Update(keyRune('r'))
		assert.Contains(t, fx.page.View(), "No comment to reply to")
	})

	t.Run("a failed load is shown, not an empty list", func(t *testing.T) {
		t.Parallel()
		fx := newDetailFixture(t, "Some text")
		fx.client.ErrorToReturn = errors.New("comments are not enabled for this connection")
		_, cmd := fx.page.Update(keyRune('c'))
		fx.drain(t, cmd)

		assert.Contains(t, fx.page.View(), "Could not load comments: comments are not enabled")
	})
}

func TestComments_AddAndReply(t *testing.T) {
	t.Parallel()

	t.Run("a new comment goes on the page, then the list reloads", func(t *testing.T) {
		t.Parallel()
		fx := openComments(t, testComments())

		fx.page.Update(keyRune('a'))
		require.Equal(t, modeCommentInput, fx.page.mode)
		typeInto(fx, "a: looks good")
		_, cmd := fx.page.Update(tea.KeyMsg{Type: tea.KeyEnter})
		assert.True(t, fx.page.Busy(), "nothing else happens until Notion answered")
		fx.drain(t, cmd)

		require.Equal(t, []testhelpers.AddCommentCall{{PageID: "page-1", Text: "a: looks good"}}, fx.client.AddCommentCalls,
			"letters typed into the prompt are text, not keys of the list")
		assert.Equal(t, modeComments, fx.page.mode)
		assert.False(t, fx.page.Busy())
		assert.Equal(t, 2, fx.client.ListCommentsCalls, "the list was loaded again")
	})

	t.Run("a reply goes to the thread of the selected comment", func(t *testing.T) {
		t.Parallel()
		fx := openComments(t, testComments())
		fx.page.Update(keyRune('j'))
		fx.page.Update(keyRune('j')) // Linus, thread t2

		fx.page.Update(keyRune('r'))
		assert.Contains(t, fx.page.View(), "Reply")
		typeInto(fx, "fixed")
		_, cmd := fx.page.Update(tea.KeyMsg{Type: tea.KeyEnter})
		fx.drain(t, cmd)

		assert.Equal(t, []testhelpers.AddCommentCall{{PageID: "page-1", ThreadID: "t2", Text: "fixed"}}, fx.client.AddCommentCalls)
	})

	t.Run("an empty comment is not sent", func(t *testing.T) {
		t.Parallel()
		fx := openComments(t, testComments())
		fx.page.Update(keyRune('a'))
		typeInto(fx, "   ")

		_, cmd := fx.page.Update(tea.KeyMsg{Type: tea.KeyEnter})

		assert.Nil(t, cmd)
		assert.Empty(t, fx.client.AddCommentCalls)
		assert.Contains(t, fx.page.View(), "The comment is empty")
	})

	t.Run("esc in the prompt returns to the list without sending", func(t *testing.T) {
		t.Parallel()
		fx := openComments(t, testComments())
		fx.page.Update(keyRune('a'))
		typeInto(fx, "never mind")

		fx.page.Update(tea.KeyMsg{Type: tea.KeyEsc})

		assert.Equal(t, modeComments, fx.page.mode)
		assert.Empty(t, fx.client.AddCommentCalls)
	})

	t.Run("a failed post keeps what was typed", func(t *testing.T) {
		t.Parallel()
		fx := openComments(t, testComments())
		fx.page.Update(keyRune('a'))
		typeInto(fx, "important")
		fx.client.ErrorToReturn = errors.New("insufficient permissions")

		_, cmd := fx.page.Update(tea.KeyMsg{Type: tea.KeyEnter})
		fx.drain(t, cmd)

		assert.Equal(t, modeCommentInput, fx.page.mode)
		assert.False(t, fx.page.Busy())
		view := fx.page.View()
		assert.Contains(t, view, "Could not post the comment: insufficient permissions")
		assert.Contains(t, view, "important")
	})
}

func TestComments_ReadOnly(t *testing.T) {
	t.Parallel()

	fx := openComments(t, testComments())
	fx.page.readOnly = true

	for _, key := range []rune{'a', 'r'} {
		fx.page.Update(keyRune(key))
		assert.Equal(t, modeComments, fx.page.mode, "%c opens no prompt", key)
		assert.Contains(t, fx.page.View(), "Read-only mode")
	}
	assert.Contains(t, fx.page.View(), "Ada", "reading still works")
}

func TestComments_ResultForAnotherPageIsIgnored(t *testing.T) {
	t.Parallel()

	fx := openComments(t, nil)

	fx.page.Update(commentsLoadedMsg{pageID: "other-page", comments: testComments()})

	assert.Empty(t, fx.page.comments)
}

func TestComments_ControlSequencesAreNotDrawn(t *testing.T) {
	t.Parallel()

	fx := openComments(t, []notion.Comment{{ID: "c", ThreadID: "t", Author: "Eve\x1b[31m", Text: "\x1b[2Jwiped?", Created: time.Now()}})

	assert.NotContains(t, fx.page.View(), "\x1b[2J", "text from Notion cannot drive the terminal")
	assert.Contains(t, fx.page.View(), "wiped?")
}
