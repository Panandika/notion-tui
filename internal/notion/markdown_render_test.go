package notion

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderableMarkdown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		source   string
		expected string
	}{
		{
			name:     "plain markdown passes through",
			source:   "# Title\n\ntext with [link](https://example.com)\n\n- [x] done",
			expected: "# Title\n\ntext with [link](https://example.com)\n\n- [x] done",
		},
		{
			name:     "inline tags become markdown with tight markers",
			source:   "and <b>type </b><b><code>/</code></b><b> </b>to see <i>this</i> and <s>that</s>",
			expected: "and **type** **`/`** to see *this* and ~~that~~",
		},
		{
			name:     "colors and underline are dropped, text kept",
			source:   `<span color="gray">List your </span><span color="gray"><u>income</u></span>`,
			expected: "List your income",
		},
		{
			name:     "mentions become links",
			source:   `(Here’s an <mention url="https://app.notion.com/p/abc">Example sub-page</mention>!)`,
			expected: "(Here’s an [Example sub-page](https://app.notion.com/p/abc)!)",
		},
		{
			name:     "custom emoji placeholders disappear",
			source:   `Click <unknown_mention url="notion://custom_emoji/1/2" alt="custom_emoji" /> here`,
			expected: "Click  here",
		},
		{
			name:     "empty paragraphs collapse",
			source:   "one\n\n<p />\n\n<p />\n\ntwo",
			expected: "one\n\ntwo",
		},
		{
			name:     "paragraph tags are layout only",
			source:   "<p>\nSelect “Status”. \n</p>",
			expected: "Select “Status”.",
		},
		{
			name:     "toggle shows title then quoted content",
			source:   "<toggle>\nMore tips\n\n- first\n- second\n</toggle>\n\nafter",
			expected: "▸ More tips\n\n> - first\n> - second\n\nafter",
		},
		{
			name:     "callout is quoted with emoji icon",
			source:   "<callout icon=\"💡\" color=\"gray_bg\">\n<b>Tip</b>\n\nRead this.\n</callout>",
			expected: "> 💡 **Tip**\n>\n> Read this.",
		},
		{
			name:     "callout with image icon has no icon text",
			source:   "<callout icon=\"/icons/notion_gray.svg\">\nNote\n</callout>",
			expected: "> Note",
		},
		{
			name:     "columns are shown one after another",
			source:   "<columns>\n<column ratio=\"50\">\nleft\n</column>\n<column ratio=\"50\">\nright\n</column>\n</columns>",
			expected: "left\n\nright",
		},
		{
			name:     "multi-line heading tag becomes a heading",
			source:   "<h3 color=\"gray_bg\">\n Mon\n</h3>\n\n- [x] Call Mom",
			expected: "### Mon\n\n- [x] Call Mom",
		},
		{
			name:     "database and page blocks show their titles",
			source:   "<database url=\"https://app.notion.com/p/1\" inline=\"true\">\nTodo List\n</database>\n\n<page url=\"https://app.notion.com/p/2\">Notes</page>",
			expected: "🗂 **Todo List** (database)\n\n📄 **Notes**",
		},
		{
			name:     "unknown block is announced",
			source:   `<unknown url="https://app.notion.com/p/x#y" alt="button" />`,
			expected: "*[unsupported block: button]*",
		},
		{
			name:     "table with header row",
			source:   "<table header-row=\"true\">\n<tr>\n<td>Name</td>\n<td>Qty</td>\n</tr>\n<tr>\n<td>a|b</td>\n<td><b>2</b></td>\n</tr>\n</table>",
			expected: "| Name | Qty |\n| --- | --- |\n| a\\|b | **2** |",
		},
		{
			name:     "table without header row gets an empty header",
			source:   "<table>\n<tr>\n<td>a</td>\n<td>1</td>\n</tr>\n</table>",
			expected: "|  |  |\n| --- | --- |\n| a | 1 |",
		},
		{
			name:     "code blocks are left untouched",
			source:   "```html\n<b>not bold</b>\n<toggle>\n```",
			expected: "```html\n<b>not bold</b>\n<toggle>\n```",
		},
		{
			name:     "self-closing mention does not swallow the text after it",
			source:   `<mention-user url="u1"/> said hi to <mention-page url="p">Page</mention-page>`,
			expected: "u1 said hi to [Page](p)",
		},
		{
			name:     "date mention shows its dates",
			source:   `Due <mention-date start="2026-11-12" end="2026-11-14"/>`,
			expected: "Due 2026-11-12 → 2026-11-14",
		},
		{
			name:     "angle brackets that are not Notion tags are kept",
			source:   "Use Vec<String> or press <Enter>, and a < b",
			expected: "Use Vec<String> or press <Enter>, and a < b",
		},
		{
			name:     "inline code is literal",
			source:   "inline `a<b>c</b>` and <code>List<T></code> stay",
			expected: "inline `a<b>c</b>` and `List<T>` stay",
		},
		{
			name:     "tabs inside code blocks are kept",
			source:   "```make\nall:\n\tgo build\n```",
			expected: "```make\nall:\n\tgo build\n```",
		},
		{
			name:     "tilde fences are code blocks too",
			source:   "~~~\n<b>x</b>\n~~~",
			expected: "~~~\n<b>x</b>\n~~~",
		},
		{
			name:     "a longer fence can contain a shorter one",
			source:   "````md\n```\n<b>x</b>\n```\n````\n\n<b>y</b>",
			expected: "````md\n```\n<b>x</b>\n```\n````\n\n**y**",
		},
		{
			name:     "table cell spanning several lines",
			source:   "<table header-row=\"true\">\n<tr>\n<td>\nmulti\nline\n</td>\n</tr>\n<tr>\n<td>x</td>\n</tr>\n</table>",
			expected: "| multi line |\n| --- |\n| x |",
		},
		{
			name:     "toggle inside a callout stays inside the quote",
			source:   "<callout>\nNote\n\n<toggle>\ninner\n\nbody\n</toggle>\n</callout>",
			expected: "> Note\n>\n> ▸ inner\n>\n> > body",
		},
		{
			name:     "nested list indentation is kept",
			source:   "- [ ] parent\n  - [ ] child",
			expected: "- [ ] parent\n  - [ ] child",
		},
		{
			name:     "tab indentation becomes spaces",
			source:   "- parent\n\t- child",
			expected: "- parent\n    - child",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, RenderableMarkdown(tt.source))
		})
	}
}
