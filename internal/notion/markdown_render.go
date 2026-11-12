package notion

import (
	"fmt"
	"regexp"
	"strings"
)

// Notion-flavored Markdown mixes Markdown with HTML-like tags (<toggle>,
// <callout>, <columns>, <b>, <mention>, ...). RenderableMarkdown rewrites it
// into plain Markdown that a terminal renderer can display. It is for display
// and export only: the result is not sent back to Notion.

var (
	attrPattern        = regexp.MustCompile(`([a-zA-Z][\w-]*)="([^"]*)"`)
	headingOpenPattern = regexp.MustCompile(`^<h([1-6])(\s[^>]*)?>$`)
	headingLinePattern = regexp.MustCompile(`^<h([1-6])(?:\s[^>]*)?>(.*)</h[1-6]>$`)
	titledBlockPattern = regexp.MustCompile(`^<(database|page)(\s[^>]*)?>(.*)</(?:database|page)>$`)
	cellPattern        = regexp.MustCompile(`^<t[dh](?:\s[^>]*)?>(.*)</t[dh]>$`)

	codePattern        = regexp.MustCompile(`<code>(.*?)</code>`)
	boldPattern        = regexp.MustCompile(`<(?:b|strong)>(.*?)</(?:b|strong)>`)
	italicPattern      = regexp.MustCompile(`<(?:i|em)>(.*?)</(?:i|em)>`)
	strikePattern      = regexp.MustCompile(`<(?:s|del|strike)>(.*?)</(?:s|del|strike)>`)
	mentionPattern     = regexp.MustCompile(`<(mention[\w-]*)(\s[^>]*)?>(.*?)</mention[\w-]*>`)
	mentionVoidPattern = regexp.MustCompile(`<(mention[\w-]*)(\s[^>]*?)?\s*/>`)
	unknownInline      = regexp.MustCompile(`<unknown_mention(\s[^>]*?)?\s*/>`)
	lineBreakPattern   = regexp.MustCompile(`<br\s*/?>`)
	backtickSpan       = regexp.MustCompile("`[^`]+`")
	fenceOpenPattern   = regexp.MustCompile("^(`{3,}|~{3,})")

	// notionTagPattern matches the tags Notion-flavored Markdown uses. Only
	// these are removed when they have no Markdown form: other text in
	// angle brackets ("Vec<String>", "press <Enter>") is the user's content.
	notionTagPattern = regexp.MustCompile(`</?(?:span|u|p|b|strong|i|em|s|del|strike|code|br|` +
		`mention[\w-]*|unknown[\w-]*|toggle|callout|columns|column|details|summary|` +
		`h[1-6]|table|colgroup|col|tr|td|th|database|page|empty-block|table_of_contents|` +
		`synced_block[\w-]*|custom-block|audio|video|file|pdf)(\s[^<>]*)?/?>`)
)

// codePlaceholder marks where protected inline code is put back. The
// control characters cannot occur in page text.
const codePlaceholder = "\x00%d\x00"

// container is an open <toggle> or <callout> whose content is shown quoted.
type container struct {
	kind         string // "toggle" or "callout"
	icon         string
	awaitingHead bool // The next text line is the container's first line
}

// titledBlock is an open <database> or <page> waiting for its title line.
type titledBlock struct {
	kind string
}

// tableState collects the rows of an open <table>.
type tableState struct {
	hasHeader bool
	rows      [][]string
	row       []string
	inRow     bool
	cell      []string // Lines of a cell whose content spans several lines
	inCell    bool
}

type markdownRenderer struct {
	out        []string
	containers []container
	heading    int    // Level of an open multi-line <hN> block, 0 if none
	headIndent string // Indent of that block's opening tag
	titled     *titledBlock
	table      *tableState
	fence      string // Marker that opened the current code block, "" outside one
}

// RenderableMarkdown converts Notion-flavored Markdown into plain Markdown.
func RenderableMarkdown(source string) string {
	r := &markdownRenderer{}
	for _, line := range strings.Split(strings.ReplaceAll(source, "\r\n", "\n"), "\n") {
		r.line(line)
	}
	return collapseBlankLines(r.out)
}

func (r *markdownRenderer) line(raw string) {
	body := strings.TrimSpace(raw)
	indent := strings.ReplaceAll(raw[:len(raw)-len(strings.TrimLeft(raw, " \t"))], "\t", "    ")

	// Code blocks are literal: no tag handling and no whitespace changes
	// inside them (tabs matter in Makefiles and Go code).
	if r.fence != "" {
		if closesFence(body, r.fence) {
			r.fence = ""
			r.emit(indent, body)
			return
		}
		r.emit("", raw)
		return
	}
	if marker := fenceOpenPattern.FindString(body); marker != "" {
		r.fence = marker
		r.emit(indent, body)
		return
	}

	if r.table != nil {
		r.tableLine(indent, body)
		return
	}

	switch {
	case body == "":
		r.blank()

	case body == "<p />" || body == "<p/>" || body == "<empty-block/>" || body == "<empty-block />":
		r.blank()

	case body == "<p>" || body == "</p>" || body == "<columns>" || body == "</columns>":
		// Layout-only tags: their content lines carry the text.

	case strings.HasPrefix(body, "<column") && strings.HasSuffix(body, ">"):
		// A column opens; columns are shown one after another.

	case body == "</column>":
		r.blank()

	case strings.HasPrefix(body, "<toggle") && strings.HasSuffix(body, ">"):
		r.blank()
		r.containers = append(r.containers, container{kind: "toggle", awaitingHead: true})

	case strings.HasPrefix(body, "<callout") && strings.HasSuffix(body, ">"):
		r.blank()
		r.containers = append(r.containers, container{kind: "callout", icon: emojiIcon(attr(body, "icon")), awaitingHead: true})

	case body == "</toggle>" || body == "</callout>":
		if len(r.containers) > 0 {
			r.containers = r.containers[:len(r.containers)-1]
		}
		r.blank()

	case strings.HasPrefix(body, "<table") && strings.HasSuffix(body, ">"):
		r.table = &tableState{hasHeader: attr(body, "header-row") == "true"}

	case headingLinePattern.MatchString(body):
		m := headingLinePattern.FindStringSubmatch(body)
		r.emit(indent, headingPrefix(m[1])+inlineMarkdown(m[2]))

	case headingOpenPattern.MatchString(body):
		r.heading = int(headingOpenPattern.FindStringSubmatch(body)[1][0] - '0')
		r.headIndent = indent

	case strings.HasPrefix(body, "</h") && strings.HasSuffix(body, ">"):
		r.heading = 0

	case titledBlockPattern.MatchString(body):
		m := titledBlockPattern.FindStringSubmatch(body)
		r.emit(indent, titledBlockLine(m[1], m[3]))

	case (strings.HasPrefix(body, "<database") || strings.HasPrefix(body, "<page")) && strings.HasSuffix(body, ">"):
		kind := "page"
		if strings.HasPrefix(body, "<database") {
			kind = "database"
		}
		r.titled = &titledBlock{kind: kind}

	case body == "</database>" || body == "</page>":
		r.titled = nil

	case strings.HasPrefix(body, "<unknown ") && strings.HasSuffix(body, "/>"):
		r.emit(indent, unsupportedNotice(attr(body, "alt")))

	case strings.HasPrefix(body, "<table_of_contents"):
		// A table of contents has no content of its own in a terminal view.

	default:
		r.text(indent, body)
	}
}

// closesFence reports whether a line ends a code block opened with marker:
// the same character, at least as many of them, and nothing else.
func closesFence(body, marker string) bool {
	if len(body) < len(marker) || strings.Trim(body, marker[:1]) != "" {
		return false
	}
	return true
}

// text emits a content line, applying any open heading, titled block or
// container state.
func (r *markdownRenderer) text(indent, body string) {
	switch {
	case r.heading > 0:
		// The text line may carry its own leading space; a heading must
		// start at the indent of its tag to be recognized as a heading.
		r.emit(r.headIndent, strings.Repeat("#", r.heading)+" "+strings.TrimSpace(inlineMarkdown(body)))
		return
	case r.titled != nil:
		r.emit(indent, titledBlockLine(r.titled.kind, body))
		return
	}
	r.emit(indent, inlineMarkdown(body))
}

// emit writes one output line, quoted once per open container. A toggle's
// first line is its title and sits one quote level higher.
func (r *markdownRenderer) emit(indent, text string) {
	depth := len(r.containers)
	if depth > 0 {
		top := &r.containers[depth-1]
		if top.awaitingHead {
			top.awaitingHead = false
			switch top.kind {
			case "toggle":
				outer := strings.Repeat("> ", depth-1)
				r.out = append(r.out, indent+outer+"▸ "+text, strings.TrimRight(outer, " "))
				return
			case "callout":
				if top.icon != "" {
					text = top.icon + " " + text
				}
			}
		}
	}
	r.out = append(r.out, indent+strings.Repeat("> ", depth)+text)
}

// blank writes an empty line, keeping the quote of open containers so that
// their paragraphs stay inside the quote.
func (r *markdownRenderer) blank() {
	depth := len(r.containers)
	if depth > 0 && !r.containers[depth-1].awaitingHead {
		r.out = append(r.out, strings.TrimRight(strings.Repeat("> ", depth), " "))
		return
	}
	r.out = append(r.out, "")
}

func (r *markdownRenderer) tableLine(indent, body string) {
	t := r.table
	switch {
	case body == "</table>":
		r.table = nil
		r.blank()
		for _, line := range tableMarkdown(t) {
			r.emit(indent, line)
		}
		r.blank()
	case strings.HasPrefix(body, "<tr") && strings.HasSuffix(body, ">"):
		t.row, t.inRow = nil, true
	case body == "</tr>":
		if t.inRow {
			t.rows = append(t.rows, t.row)
		}
		t.row, t.inRow = nil, false
	case cellPattern.MatchString(body):
		t.addCell(cellPattern.FindStringSubmatch(body)[1])
	case (strings.HasPrefix(body, "<td") || strings.HasPrefix(body, "<th")) && strings.HasSuffix(body, ">"):
		t.cell, t.inCell = nil, true
	case body == "</td>" || body == "</th>":
		if t.inCell {
			t.addCell(strings.Join(t.cell, " "))
		}
		t.cell, t.inCell = nil, false
	case t.inCell && body != "":
		t.cell = append(t.cell, body)
	}
}

// addCell appends a cell to the current row. Pipes are escaped because they
// separate cells in Markdown tables.
func (t *tableState) addCell(content string) {
	cell := inlineMarkdown(content)
	t.row = append(t.row, strings.ReplaceAll(cell, "|", `\|`))
}

// tableMarkdown renders collected rows as a Markdown table. Markdown needs a
// header row, so a table without one gets an empty header.
func tableMarkdown(t *tableState) []string {
	width := 0
	for _, row := range t.rows {
		if len(row) > width {
			width = len(row)
		}
	}
	if width == 0 {
		return nil
	}

	format := func(cells []string) string {
		var sb strings.Builder
		sb.WriteString("|")
		for i := 0; i < width; i++ {
			cell := ""
			if i < len(cells) {
				cell = cells[i]
			}
			sb.WriteString(" " + cell + " |")
		}
		return sb.String()
	}

	body := t.rows
	var lines []string
	if t.hasHeader {
		lines = append(lines, format(t.rows[0]))
		body = t.rows[1:]
	} else {
		lines = append(lines, format(nil))
	}
	lines = append(lines, "|"+strings.Repeat(" --- |", width))
	for _, row := range body {
		lines = append(lines, format(row))
	}
	return lines
}

// inlineMarkdown rewrites inline tags as Markdown and drops the Notion tags
// that have no terminal equivalent (colors, underline).
func inlineMarkdown(text string) string {
	// Inline code is literal. Take it out first so nothing inside it is
	// rewritten, and put it back at the end.
	var protected []string
	protect := func(code string) string {
		protected = append(protected, code)
		return fmt.Sprintf(codePlaceholder, len(protected)-1)
	}
	text = codePattern.ReplaceAllStringFunc(text, func(tag string) string {
		return protect("`" + codePattern.FindStringSubmatch(tag)[1] + "`")
	})
	text = backtickSpan.ReplaceAllStringFunc(text, protect)

	text = unknownInline.ReplaceAllStringFunc(text, func(tag string) string {
		alt := attr(tag, "alt")
		if alt == "" || alt == "custom_emoji" {
			return ""
		}
		return "[" + alt + "]"
	})
	// Self-closing mentions first: otherwise one would be read as the
	// opening tag of the next paired mention and swallow the text between.
	text = mentionVoidPattern.ReplaceAllStringFunc(text, func(tag string) string {
		if start := attr(tag, "start"); start != "" {
			if end := attr(tag, "end"); end != "" {
				return start + " → " + end
			}
			return start
		}
		return attr(tag, "url")
	})
	text = mentionPattern.ReplaceAllStringFunc(text, func(tag string) string {
		m := mentionPattern.FindStringSubmatch(tag)
		label, target := m[3], attr(m[2], "url")
		if target == "" {
			return label
		}
		return fmt.Sprintf("[%s](%s)", label, target)
	})
	text = lineBreakPattern.ReplaceAllString(text, " ")
	text = wrapTight(text, boldPattern, "**")
	text = wrapTight(text, italicPattern, "*")
	text = wrapTight(text, strikePattern, "~~")
	// Remaining Notion tags (<span color>, <u>, ...) have no Markdown form.
	text = notionTagPattern.ReplaceAllString(text, "")

	for index, code := range protected {
		text = strings.ReplaceAll(text, fmt.Sprintf(codePlaceholder, index), code)
	}
	return text
}

// wrapTight replaces a tag pair with a Markdown marker, keeping surrounding
// whitespace outside the markers ("<b>word </b>" becomes "**word** ").
func wrapTight(text string, pattern *regexp.Regexp, marker string) string {
	return pattern.ReplaceAllStringFunc(text, func(match string) string {
		inner := pattern.FindStringSubmatch(match)[1]
		trimmed := strings.TrimSpace(inner)
		if trimmed == "" {
			return inner
		}
		start := strings.Index(inner, trimmed)
		return inner[:start] + marker + trimmed + marker + inner[start+len(trimmed):]
	})
}

// attr returns the value of an attribute in a tag, or "".
func attr(tag, name string) string {
	for _, m := range attrPattern.FindAllStringSubmatch(tag, -1) {
		if m[1] == name {
			return m[2]
		}
	}
	return ""
}

// emojiIcon returns the icon if it is an emoji rather than an image path.
func emojiIcon(icon string) string {
	if icon == "" || strings.ContainsAny(icon, "/.") {
		return ""
	}
	return icon
}

func headingPrefix(level string) string {
	return strings.Repeat("#", int(level[0]-'0')) + " "
}

func titledBlockLine(kind, title string) string {
	title = titleOrUntitled(strings.TrimSpace(inlineMarkdown(title)))
	if kind == "database" {
		return fmt.Sprintf("🗂 **%s** (database)", title)
	}
	return fmt.Sprintf("📄 **%s**", title)
}

func unsupportedNotice(kind string) string {
	if kind == "" {
		return "*[unsupported block]*"
	}
	return fmt.Sprintf("*[unsupported block: %s]*", kind)
}

// collapseBlankLines joins lines, squeezing runs of blank lines (and of
// empty quote lines) into one and trimming them at both ends.
func collapseBlankLines(lines []string) string {
	isQuoteOnly := func(line string) bool {
		return line != "" && strings.Trim(line, "> ") == ""
	}
	isEmpty := func(line string) bool {
		return strings.TrimSpace(line) == "" || isQuoteOnly(line)
	}

	var out []string
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			line = ""
		}
		// An empty line after another empty one adds nothing, and an empty
		// quote line at the start would open a quote with nothing in it.
		if isEmpty(line) && (len(out) == 0 || isEmpty(out[len(out)-1])) {
			continue
		}
		out = append(out, line)
	}
	for len(out) > 0 && isEmpty(out[len(out)-1]) {
		out = out[:len(out)-1]
	}
	return strings.Join(out, "\n")
}
