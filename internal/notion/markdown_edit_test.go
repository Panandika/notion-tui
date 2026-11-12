package notion

import (
	"math/rand"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// applyUpdates replays edits the way the API applies them and fails the
// test if the API would reject one.
func applyUpdates(t *testing.T, text string, updates []ContentUpdate) string {
	t.Helper()
	result, ok := ReplayContentUpdates(text, updates)
	require.True(t, ok, "every old_str must be non-blank and match exactly once when applied: %+v", updates)
	return result
}

func TestBuildContentUpdates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		original    string
		edited      string
		wantOK      bool
		wantUpdates int // -1 skips the count check
	}{
		{name: "no change", original: "a\nb", edited: "a\nb", wantOK: true, wantUpdates: 0},
		{name: "trailing newline from editor is not a change", original: "a\nb", edited: "a\nb\n", wantOK: true, wantUpdates: 0},
		{name: "windows line endings are not a change", original: "a\nb", edited: "a\r\nb\r\n", wantOK: true, wantUpdates: 0},
		{name: "one line changed", original: "# Title\n\nfirst\n\nsecond", edited: "# Title\n\nfirst!\n\nsecond", wantOK: true, wantUpdates: 1},
		{name: "two separate lines changed", original: "one\n\ntwo\n\nthree\n\nfour\n\nfive", edited: "ONE\n\ntwo\n\nthree\n\nfour\n\nFIVE", wantOK: true, wantUpdates: 2},
		{name: "line inserted between others", original: "alpha\n\nomega", edited: "alpha\n\nmiddle\n\nomega", wantOK: true, wantUpdates: -1},
		{name: "line appended", original: "alpha\n\nomega", edited: "alpha\n\nomega\n\nnew end", wantOK: true, wantUpdates: -1},
		{name: "line deleted", original: "keep\n\ndrop me\n\nkeep too", edited: "keep\n\nkeep too", wantOK: true, wantUpdates: -1},
		{name: "duplicate lines need context to be unique", original: "- [ ] \n- [ ] \n- [ ] \nend", edited: "- [ ] \n- [x] done\n- [ ] \nend", wantOK: true, wantUpdates: -1},
		{name: "empty page cannot be patched", original: "", edited: "hello", wantOK: false},
		{name: "everything rewritten needs a replace", original: "old", edited: "new", wantOK: false},
		{name: "only repeated lines cannot be anchored", original: "x\nx\nx", edited: "x\ny\nx", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			updates, ok := BuildContentUpdates(tt.original, tt.edited)

			require.Equal(t, tt.wantOK, ok)
			if !ok {
				return
			}
			if tt.wantUpdates >= 0 {
				assert.Len(t, updates, tt.wantUpdates)
			}
			got := applyUpdates(t, NormalizeMarkdown(tt.original), updates)
			assert.Equal(t, NormalizeMarkdown(tt.edited), got)
		})
	}
}

func TestBuildContentUpdates_LeavesUntouchedBlocksAlone(t *testing.T) {
	t.Parallel()

	original := "intro\n\n<unknown url=\"u\" alt=\"button\" />\n\nfirst paragraph\n\nsecond paragraph"
	edited := "intro\n\n<unknown url=\"u\" alt=\"button\" />\n\nfirst paragraph\n\nsecond paragraph, edited"

	updates, ok := BuildContentUpdates(original, edited)

	require.True(t, ok)
	require.Len(t, updates, 1)
	assert.NotContains(t, updates[0].OldStr, "<unknown", "blocks the user did not touch must stay out of the edit")
}

func TestBuildContentUpdates_HugePageFallsBackToReplace(t *testing.T) {
	t.Parallel()

	original := strings.Repeat("line\n", 2100) + "tail"
	edited := strings.Repeat("line\n", 2100) + "tail changed"

	_, ok := BuildContentUpdates(original, edited)

	assert.False(t, ok)
}

// Regression cases from review: changes close to each other used to lose
// inserted lines or crash.
func TestBuildContentUpdates_NearbyChanges(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		original string
		edited   string
	}{
		{
			name:     "change next to an insertion keeps the inserted line",
			original: "- [ ] Call\nNotes for today\n- [ ] Call\nEnd",
			edited:   "- [x] Call\nNotes for today\nNew line added\n- [ ] Call\nEnd",
		},
		{name: "insertion after repeated line", original: "b\na\nb", edited: "x\na\na \nb"},
		{name: "deletion among repeats", original: "x\nx\nab\n\n\na", edited: "x\nab\n\na\na"},
		{name: "hunks with different sizes that touch", original: "\na\na\nx", edited: "b\n\nx"},
		{name: "growth into a neighbor", original: "b\n\nab\nb\n", edited: "x\na\nb\nb\n"},
		{name: "tail rewrite among repeats", original: "a \n\na\na\na\nab", edited: "a \n\na\nb\na"},
		{
			name:     "new text of one edit equals old text of a later one",
			original: "Status: draft\n\nbody text\n\nSigned: final",
			edited:   "Signed: final\n\nbody text\n\nSigned: FINAL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			updates, ok := BuildContentUpdates(tt.original, tt.edited)

			// Falling back to a full replace is acceptable; a wrong edit is not.
			if !ok {
				return
			}
			assert.Equal(t, NormalizeMarkdown(tt.edited), applyUpdates(t, NormalizeMarkdown(tt.original), updates))
		})
	}
}

// Property check over many small random pages: whenever edits are returned,
// replaying them like the API must give exactly the edited text.
func TestBuildContentUpdates_RandomPagesNeverLoseText(t *testing.T) {
	t.Parallel()

	vocabulary := []string{"a", "b", "ab", "", " ", "x", "a ", "- [ ] ", "# h"}
	rng := rand.New(rand.NewSource(20261112))
	randomPage := func() []string {
		lines := make([]string, 1+rng.Intn(7))
		for i := range lines {
			lines[i] = vocabulary[rng.Intn(len(vocabulary))]
		}
		return lines
	}
	mutate := func(lines []string) []string {
		out := append([]string(nil), lines...)
		for edits := 1 + rng.Intn(3); edits > 0; edits-- {
			pos := rng.Intn(len(out) + 1)
			word := vocabulary[rng.Intn(len(vocabulary))]
			switch op := rng.Intn(3); {
			case op == 0 || len(out) == 0: // insert
				out = append(out[:pos], append([]string{word}, out[pos:]...)...)
			case op == 1 && pos < len(out): // replace
				out[pos] = word
			case pos < len(out): // delete
				out = append(out[:pos], out[pos+1:]...)
			}
		}
		return out
	}

	edits, replaces := 0, 0
	for i := 0; i < 20000; i++ {
		page := randomPage()
		original := strings.Join(page, "\n")
		edited := strings.Join(mutate(page), "\n")

		updates, ok := BuildContentUpdates(original, edited)
		if !ok {
			replaces++
			continue
		}
		edits++
		result, replayOK := ReplayContentUpdates(NormalizeMarkdown(original), updates)
		require.True(t, replayOK, "API would reject edits for %q -> %q: %+v", original, edited, updates)
		require.Equal(t, NormalizeMarkdown(edited), result, "wrong result for %q -> %q: %+v", original, edited, updates)
	}
	// The property must not pass by always refusing.
	assert.Greater(t, edits, replaces/4, "edits=%d replaces=%d", edits, replaces)
}
