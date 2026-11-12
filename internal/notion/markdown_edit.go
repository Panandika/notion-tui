package notion

import (
	"strings"
)

// maxDiffCells bounds the line-diff table. Larger pages fall back to a full
// replace instead of spending memory on a minimal diff.
const maxDiffCells = 4_000_000

// NormalizeMarkdown prepares editor output for comparison with the API's
// Markdown: Unix line endings and no trailing blank lines.
func NormalizeMarkdown(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	return strings.TrimRight(text, "\n")
}

// BuildContentUpdates turns an edited page into search-and-replace edits that
// touch only the changed parts, so untouched blocks keep their identity.
//
// Every result is checked by replaying the edits the way the API applies
// them (in order, each old text matching exactly once): edits are only
// returned if that replay produces the edited text. ok=false means the
// change cannot be expressed safely as edits and the caller should replace
// the page content instead. No edits and ok=true means the texts are equal.
func BuildContentUpdates(original, edited string) (updates []ContentUpdate, ok bool) {
	original, edited = NormalizeMarkdown(original), NormalizeMarkdown(edited)
	if original == edited {
		return nil, true
	}
	if strings.TrimSpace(original) == "" {
		return nil, false
	}

	oldLines := strings.Split(original, "\n")
	newLines := strings.Split(edited, "\n")
	if len(oldLines)*len(newLines) > maxDiffCells {
		return nil, false
	}

	hunks := diffHunks(oldLines, newLines)
	if len(hunks) == 0 {
		return nil, false
	}

	// First choice: one edit per changed region.
	if anchored, fits := anchorHunks(hunks, original, oldLines); fits {
		updates = hunksToUpdates(anchored, oldLines, newLines)
		if replayMatches(original, edited, updates) {
			return updates, true
		}
	}

	// Second choice: a single edit spanning the first through last change.
	span := []hunk{{
		oldStart: hunks[0].oldStart, oldEnd: hunks[len(hunks)-1].oldEnd,
		newStart: hunks[0].newStart, newEnd: hunks[len(hunks)-1].newEnd,
	}}
	if anchored, fits := anchorHunks(span, original, oldLines); fits {
		updates = hunksToUpdates(anchored, oldLines, newLines)
		if replayMatches(original, edited, updates) {
			return updates, true
		}
	}

	return nil, false
}

// ReplayContentUpdates applies edits the way the API does: in order, each
// old text must be non-blank and occur exactly once at the time it is
// applied. ok=false means the API would reject or skip an edit.
func ReplayContentUpdates(text string, updates []ContentUpdate) (result string, ok bool) {
	for _, update := range updates {
		if strings.TrimSpace(update.OldStr) == "" || countMatches(text, update.OldStr) != 1 {
			return "", false
		}
		text = strings.Replace(text, update.OldStr, update.NewStr, 1)
	}
	return text, true
}

// replayMatches reports whether the edits turn original into edited.
func replayMatches(original, edited string, updates []ContentUpdate) bool {
	result, ok := ReplayContentUpdates(original, updates)
	return ok && result == edited
}

// hunk is one changed region: old lines [oldStart, oldEnd) become new lines
// [newStart, newEnd). Lines outside every hunk are identical in both texts,
// and between two neighboring hunks the old and new sides hold the same
// number of lines.
type hunk struct {
	oldStart, oldEnd int
	newStart, newEnd int
}

func hunksToUpdates(hunks []hunk, oldLines, newLines []string) []ContentUpdate {
	updates := make([]ContentUpdate, 0, len(hunks))
	for _, h := range hunks {
		updates = append(updates, ContentUpdate{
			OldStr: strings.Join(oldLines[h.oldStart:h.oldEnd], "\n"),
			NewStr: strings.Join(newLines[h.newStart:h.newEnd], "\n"),
		})
	}
	return updates
}

// locatable reports whether the API can find a hunk's old text: both sides
// hold at least one line (joined lines describe the text between line
// breaks, so an empty side would lose a line break), the old text is not
// blank, and it occurs exactly once in the page.
func (h hunk) locatable(original string, oldLines []string) bool {
	if h.oldEnd <= h.oldStart || h.newEnd <= h.newStart {
		return false
	}
	oldText := strings.Join(oldLines[h.oldStart:h.oldEnd], "\n")
	return strings.TrimSpace(oldText) != "" && countMatches(original, oldText) == 1
}

// anchorHunks grows each hunk with unchanged neighbor lines until the API
// can locate it. A hunk only grows across unchanged lines, where old and new
// sides move together; when it reaches a neighboring hunk the two are merged
// into one. fits=false means a hunk would have to cover the whole page.
func anchorHunks(hunks []hunk, original string, oldLines []string) (anchored []hunk, fits bool) {
	anchored = append(anchored, hunks...)

	for i := 0; i < len(anchored); {
		h := &anchored[i]
		if h.locatable(original, oldLines) {
			i++
			continue
		}

		// Unchanged lines available above and below this hunk.
		floor, ceiling := 0, len(oldLines)
		if i > 0 {
			floor = anchored[i-1].oldEnd
		}
		if i < len(anchored)-1 {
			ceiling = anchored[i+1].oldStart
		}
		roomUp, roomDown := h.oldStart-floor, ceiling-h.oldEnd

		switch {
		case roomUp > 0 && roomUp >= roomDown:
			h.oldStart--
			h.newStart--
		case roomDown > 0:
			h.oldEnd++
			h.newEnd++
		case i > 0:
			// Touching the previous hunk: merge and re-check the result.
			anchored[i-1].oldEnd, anchored[i-1].newEnd = h.oldEnd, h.newEnd
			anchored = append(anchored[:i], anchored[i+1:]...)
			i--
		case i < len(anchored)-1:
			anchored[i+1].oldStart, anchored[i+1].newStart = h.oldStart, h.newStart
			anchored = append(anchored[:i], anchored[i+1:]...)
		default:
			// A single hunk covering the whole page: nothing left to anchor on.
			return nil, false
		}
	}

	// A hunk that is the entire page is a replace, not an edit.
	if len(anchored) == 1 && anchored[0].oldStart == 0 && anchored[0].oldEnd == len(oldLines) {
		return nil, false
	}
	return anchored, true
}

// countMatches counts every position where sub occurs in text, including
// overlapping ones ("x\nx" occurs twice in "x\nx\nx").
func countMatches(text, sub string) int {
	count := 0
	for offset := 0; ; {
		index := strings.Index(text[offset:], sub)
		if index < 0 {
			return count
		}
		count++
		offset += index + 1
	}
}

// diffHunks computes the changed regions between two line slices using a
// longest-common-subsequence table. Hunks are returned in document order
// and never touch each other.
func diffHunks(oldLines, newLines []string) []hunk {
	n, m := len(oldLines), len(newLines)

	// lcs[i][j] is the LCS length of oldLines[i:] and newLines[j:].
	lcs := make([][]int32, n+1)
	for i := range lcs {
		lcs[i] = make([]int32, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			switch {
			case oldLines[i] == newLines[j]:
				lcs[i][j] = lcs[i+1][j+1] + 1
			case lcs[i+1][j] >= lcs[i][j+1]:
				lcs[i][j] = lcs[i+1][j]
			default:
				lcs[i][j] = lcs[i][j+1]
			}
		}
	}

	// Walk the table. A line pair is kept only when it is part of a longest
	// common subsequence; everything else belongs to a hunk.
	var hunks []hunk
	i, j := 0, 0
	for i < n || j < m {
		if i < n && j < m && oldLines[i] == newLines[j] && lcs[i][j] == lcs[i+1][j+1]+1 {
			i++
			j++
			continue
		}

		h := hunk{oldStart: i, newStart: j}
		for i < n || j < m {
			if i < n && j < m && oldLines[i] == newLines[j] && lcs[i][j] == lcs[i+1][j+1]+1 {
				break
			}
			switch {
			case j >= m:
				i++
			case i >= n:
				j++
			case lcs[i+1][j] >= lcs[i][j+1]:
				i++
			default:
				j++
			}
		}
		h.oldEnd, h.newEnd = i, j
		hunks = append(hunks, h)
	}
	return hunks
}
