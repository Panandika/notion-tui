package pages

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/Panandika/notion-tui/internal/notion"
	"github.com/Panandika/notion-tui/internal/ui/components"
)

const (
	commentsHelpText     = "↑/↓: move | enter: more/less | a: add | r: reply | esc: close"
	commentInputHelpText = "type the comment | enter: post | esc: cancel"

	// commentLimit is the longest comment that can be typed here.
	commentLimit = 2000
	// commentPreview is how much of a comment its row in the list shows.
	commentPreview = 60
	// commentTextLines is how many lines the selected comment gets below
	// the list, until enter gives it more.
	commentTextLines = 8
	// commentListMin is the least rows the list keeps (title, gap, a row).
	commentListMin = 3
)

// commentsLoadedMsg carries the open comments of a page.
type commentsLoadedMsg struct {
	pageID   string
	seq      int // Which load this answers; older ones are dropped
	comments []notion.Comment
	err      error
}

// commentAddedMsg reports the result of posting a comment.
type commentAddedMsg struct {
	pageID string
	err    error
}

func (commentsLoadedMsg) detailResult() {}
func (commentAddedMsg) detailResult()   {}

// StartComments shows the page's open comments.
func (dp *DetailPage) StartComments() tea.Cmd {
	if !dp.ready() {
		return nil
	}
	dp.comments = nil
	dp.commentsErr = ""
	dp.commentExpanded = false
	dp.picker = components.NewPicker("Comments", nil)
	dp.setMode(modeComments)
	return dp.loadComments()
}

func (dp *DetailPage) loadComments() tea.Cmd {
	dp.commentsLoading = true
	dp.commentsSeq++
	pageID, seq, client := dp.pageID, dp.commentsSeq, dp.notionClient
	return func() tea.Msg {
		comments, err := client.ListComments(context.Background(), pageID)
		return commentsLoadedMsg{pageID: pageID, seq: seq, comments: comments, err: err}
	}
}

func (dp *DetailPage) handleCommentsLoaded(msg commentsLoadedMsg) {
	if msg.pageID != dp.pageID || msg.seq != dp.commentsSeq || dp.mode != modeComments {
		return // Another page, an older load, or the list was closed meanwhile
	}
	dp.commentsLoading = false
	if msg.err != nil {
		dp.commentsErr = "Could not load comments: " + cleanText(msg.err.Error())
		return
	}
	dp.commentsErr = ""

	// Stay on the comment that was selected, wherever it is now
	selected := ""
	if index := dp.picker.Cursor(); index < len(dp.comments) {
		selected = dp.comments[index].ID
	}
	dp.comments = msg.comments
	title := "Comments"
	if len(msg.comments) >= notion.MaxComments {
		title = fmt.Sprintf("Comments (the first %d)", notion.MaxComments)
	}
	dp.picker = components.NewPicker(title, commentPickerItems(msg.comments))
	for index, comment := range msg.comments {
		if comment.ID == selected {
			dp.picker.SetCursor(index)
		}
	}
}

// cleanText makes text from Notion safe to draw: terminal escape sequences
// and control characters are removed, except line breaks. A tab becomes a
// space.
func cleanText(text string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r == '\n':
			return r
		case r == '\t':
			return ' '
		case unicode.IsControl(r):
			return -1
		default:
			return r
		}
	}, ansi.Strip(text))
}

// commentPickerItems draws one row per comment; replies are indented under
// the comment that started their thread.
func commentPickerItems(comments []notion.Comment) []components.PickerItem {
	items := make([]components.PickerItem, len(comments))
	for i, comment := range comments {
		label := oneLine(comment.Author) + " · " + comment.Created.Local().Format("2006-01-02 15:04")
		if i > 0 && comments[i-1].ThreadID == comment.ThreadID {
			label = "  ↳ " + label
		}
		items[i] = components.PickerItem{Label: label, Detail: ansi.Truncate(oneLine(comment.Text), commentPreview, "…")}
	}
	return items
}

// oneLine makes text from Notion safe for one row of a list: cleaned, and
// without line breaks.
func oneLine(text string) string {
	return strings.Join(strings.Fields(cleanText(text)), " ")
}

func (dp *DetailPage) updateComments(msg tea.KeyMsg) tea.Cmd {
	if dp.picker.HandleKey(msg) {
		return nil
	}
	switch msg.String() {
	case "esc", "q":
		dp.commentsLoading = false
		dp.commentsSeq++ // A load still running answers nobody
		dp.setMode(modeView)
	case "enter":
		dp.commentExpanded = !dp.commentExpanded
	case "a":
		return dp.startCommentInput("")
	case "r":
		if index := dp.picker.Cursor(); index < len(dp.comments) {
			return dp.startCommentInput(dp.comments[index].ThreadID)
		}
		dp.inputErr = "No comment to reply to: press a to add one"
	}
	return nil
}

// startCommentInput opens the prompt for a new comment, or for a reply
// when threadID names a thread.
func (dp *DetailPage) startCommentInput(threadID string) tea.Cmd {
	if dp.commentsLoading {
		return nil
	}
	if dp.readOnly {
		dp.inputErr = ReadOnlyNotice
		return nil
	}
	if dp.offline {
		dp.inputErr = "Offline: a comment cannot be posted now"
		return nil
	}
	input := textinput.New()
	input.Prompt = "Comment: "
	if threadID != "" {
		input.Prompt = "Reply: "
	}
	input.CharLimit = commentLimit
	input.Width = max(10, dp.width-8-lipgloss.Width(input.Prompt))
	input.Focus()
	dp.commentInput = input
	dp.commentThread = threadID
	dp.setMode(modeCommentInput)
	return textinput.Blink
}

func (dp *DetailPage) updateCommentInput(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		dp.setMode(modeComments)
		return nil
	case "enter":
		text := strings.TrimSpace(dp.commentInput.Value())
		if text == "" {
			dp.inputErr = "The comment is empty"
			return nil
		}
		dp.busy = true
		dp.statusBar.SetSyncStatus(components.StatusSyncing)
		pageID, threadID, client := dp.pageID, dp.commentThread, dp.notionClient
		return func() tea.Msg {
			return commentAddedMsg{pageID: pageID, err: client.AddComment(context.Background(), pageID, threadID, text)}
		}
	}
	var cmd tea.Cmd
	dp.commentInput, cmd = dp.commentInput.Update(msg)
	return cmd
}

func (dp *DetailPage) handleCommentAdded(msg commentAddedMsg) tea.Cmd {
	if msg.pageID != dp.pageID {
		return nil
	}
	dp.busy = false
	if msg.err != nil {
		dp.statusBar.SetSyncStatus(components.StatusError)
		if dp.mode == modeCommentInput {
			// What was typed is still in the prompt, to try again
			dp.inputErr = "Could not post the comment: " + cleanText(msg.err.Error())
		} else {
			dp.setNotice("Could not post the comment: " + cleanText(msg.err.Error()))
		}
		return nil
	}
	dp.statusBar.SetSyncStatus(components.StatusSynced)
	if dp.mode != modeCommentInput {
		return nil
	}
	dp.setMode(modeComments)
	return dp.loadComments()
}

// renderComments draws the list of comments with the selected one in full
// below it.
func (dp *DetailPage) renderComments(width, height int) string {
	muted := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
	switch {
	case dp.commentsLoading:
		return lipgloss.JoinVertical(lipgloss.Left, lipgloss.NewStyle().Bold(true).Render("Comments"), "", muted.Render("Loading comments..."))
	case dp.commentsErr != "":
		// Not "no comments": that would be false
		return lipgloss.JoinVertical(lipgloss.Left, lipgloss.NewStyle().Bold(true).Render("Comments"), "",
			lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Width(width).Render(dp.commentsErr))
	case len(dp.comments) == 0:
		return lipgloss.JoinVertical(lipgloss.Left, lipgloss.NewStyle().Bold(true).Render("Comments"), "",
			muted.Render("No open comments on this page. Press a to add one."))
	}

	selected := dp.comments[min(dp.picker.Cursor(), len(dp.comments)-1)]

	// The selected comment in full under the list. Expanded, it takes all
	// the room but three rows of the list.
	limit := commentTextLines
	if dp.commentExpanded {
		limit = max(commentTextLines, height-commentListMin-2)
	}
	lines := strings.Split(lipgloss.NewStyle().Width(max(1, width)).Render(cleanText(selected.Text)), "\n")
	if len(lines) > limit {
		hint := "enter: show more"
		if dp.commentExpanded {
			hint = "too long to show here in full"
		}
		more := muted.Render(fmt.Sprintf("… %d more lines (%s)", len(lines)-limit+1, hint))
		lines = append(lines[:limit-1], more)
	}
	text := strings.Join(lines, "\n")

	listHeight := max(commentListMin, height-len(lines)-2)
	return lipgloss.JoinVertical(lipgloss.Left,
		dp.picker.View(width, listHeight),
		"",
		muted.Render(fmt.Sprintf("%s wrote:", oneLine(selected.Author))),
		text)
}
