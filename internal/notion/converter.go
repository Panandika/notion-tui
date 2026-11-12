package notion

import (
	"fmt"
	"strings"

	"github.com/jomei/notionapi"
)

// maxRenderDepth guards against runaway recursion on malformed block trees.
const maxRenderDepth = 8

// ConvertBlocksToMarkdown converts a slice of Notion blocks to a Markdown string.
// Nested children attached to blocks (see SetBlockChildren) are rendered too.
func ConvertBlocksToMarkdown(blocks []notionapi.Block) (string, error) {
	return renderBlocks(blocks, 0)
}

// listState tracks the state for numbered list items.
type listState struct {
	counter int
}

// renderBlocks renders sibling blocks, separating them so that Markdown keeps
// them apart: list items stay tight, everything else gets a blank line.
func renderBlocks(blocks []notionapi.Block, depth int) (string, error) {
	var result strings.Builder
	var prevType notionapi.BlockType
	wrote := false
	counter := 0

	for i, block := range blocks {
		if block == nil {
			continue
		}

		blockType := block.GetType()
		if blockType == notionapi.BlockTypeNumberedListItem {
			counter++
		} else {
			counter = 0
		}

		md, err := renderBlock(block, &listState{counter: counter}, depth)
		if err != nil {
			return "", fmt.Errorf("convert block %d: %w", i, err)
		}
		if md == "" {
			continue
		}

		if wrote {
			if isListItem(prevType) && isListItem(blockType) {
				result.WriteString("\n")
			} else {
				result.WriteString("\n\n")
			}
		}
		result.WriteString(md)
		prevType = blockType
		wrote = true
	}

	return result.String(), nil
}

// renderBlock renders one block followed by its nested children, if any.
func renderBlock(block notionapi.Block, listCtx *listState, depth int) (string, error) {
	md, err := convertBlock(block, listCtx)
	if err != nil {
		return "", err
	}

	// Tables consume their own children (rows) in convertBlock.
	if block.GetType() == notionapi.BlockTypeTableBlock {
		return md, nil
	}

	children := BlockChildren(block)
	if len(children) == 0 || depth >= maxRenderDepth {
		return md, nil
	}

	nested, err := renderBlocks(children, depth+1)
	if err != nil {
		return "", err
	}
	if nested == "" {
		return md, nil
	}
	if md == "" {
		// Pure containers: column lists, columns, synced blocks.
		return nested, nil
	}

	switch block.GetType() {
	case notionapi.BlockTypeBulletedListItem,
		notionapi.BlockTypeNumberedListItem,
		notionapi.BlockTypeToDo:
		sep := "\n\n"
		if isListItem(firstRenderedType(children)) {
			sep = "\n"
		}
		return md + sep + prefixLines(nested, "    "), nil

	case notionapi.BlockTypeQuote, notionapi.BlockTypeCallout:
		return md + "\n>\n" + prefixLines(nested, "> "), nil

	default:
		return md + "\n\n" + nested, nil
	}
}

// isListItem reports whether a block type renders as a Markdown list item.
func isListItem(blockType notionapi.BlockType) bool {
	switch blockType {
	case notionapi.BlockTypeBulletedListItem,
		notionapi.BlockTypeNumberedListItem,
		notionapi.BlockTypeToDo:
		return true
	default:
		return false
	}
}

// firstRenderedType returns the type of the first non-nil block.
func firstRenderedType(blocks []notionapi.Block) notionapi.BlockType {
	for _, block := range blocks {
		if block != nil {
			return block.GetType()
		}
	}
	return ""
}

// prefixLines prepends prefix to every non-empty line of text.
// Empty lines get the prefix without trailing whitespace.
func prefixLines(text, prefix string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if line == "" {
			lines[i] = strings.TrimRight(prefix, " ")
			continue
		}
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}

// convertBlock converts a single Notion block to Markdown.
func convertBlock(block notionapi.Block, listCtx *listState) (string, error) {
	if block == nil {
		return "", nil
	}

	switch b := block.(type) {
	case *notionapi.ParagraphBlock:
		return convertParagraph(b), nil

	case *notionapi.Heading1Block:
		return convertHeading1(b), nil

	case *notionapi.Heading2Block:
		return convertHeading2(b), nil

	case *notionapi.Heading3Block:
		return convertHeading3(b), nil

	case *notionapi.BulletedListItemBlock:
		return convertBulletedListItem(b), nil

	case *notionapi.NumberedListItemBlock:
		return convertNumberedListItem(b, listCtx), nil

	case *notionapi.ToDoBlock:
		return convertToDo(b), nil

	case *notionapi.QuoteBlock:
		return convertQuote(b), nil

	case *notionapi.CodeBlock:
		return convertCode(b), nil

	case *notionapi.DividerBlock:
		return "---", nil

	case *notionapi.ImageBlock:
		return convertImage(b), nil

	case *notionapi.BookmarkBlock:
		return convertBookmark(b), nil

	case *notionapi.TableOfContentsBlock:
		return "[Table of Contents]", nil

	case *notionapi.CalloutBlock:
		return convertCallout(b), nil

	case *notionapi.ToggleBlock:
		return "▸ " + convertRichText(b.Toggle.RichText), nil

	case *notionapi.TableBlock:
		return convertTable(b), nil

	case *notionapi.ChildPageBlock:
		return fmt.Sprintf("📄 **%s**", titleOrUntitled(b.ChildPage.Title)), nil

	case *notionapi.ChildDatabaseBlock:
		return fmt.Sprintf("🗂 **%s** (database)", titleOrUntitled(b.ChildDatabase.Title)), nil

	case *notionapi.EquationBlock:
		return FormatCode(b.Equation.Expression, "latex"), nil

	case *notionapi.EmbedBlock:
		return convertLink("Embed", b.Embed.Caption, b.Embed.URL), nil

	case *notionapi.VideoBlock:
		return convertLink("Video", b.Video.Caption, fileURL(b.Video.File, b.Video.External)), nil

	case *notionapi.FileBlock:
		return convertLink("File", b.File.Caption, fileURL(b.File.File, b.File.External)), nil

	case *notionapi.PdfBlock:
		return convertLink("PDF", b.Pdf.Caption, fileURL(b.Pdf.File, b.Pdf.External)), nil

	case *notionapi.LinkPreviewBlock:
		return convertLink("Link", nil, b.LinkPreview.URL), nil

	case *notionapi.LinkToPageBlock:
		return convertLinkToPage(b), nil

	case *notionapi.UnsupportedBlock:
		return "*[unsupported block]*", nil

	default:
		// Containers (column lists, columns, synced blocks) have no text of
		// their own; their children are rendered by renderBlock.
		return "", nil
	}
}

// titleOrUntitled returns a placeholder for empty titles, which would
// otherwise leave bare emphasis markers in the output.
func titleOrUntitled(title string) string {
	if strings.TrimSpace(title) == "" {
		return "Untitled"
	}
	return title
}

// convertTable converts a table block and its row children to a Markdown table.
func convertTable(block *notionapi.TableBlock) string {
	if block == nil {
		return ""
	}

	var rows [][]string
	width := block.Table.TableWidth
	for _, child := range block.Table.Children {
		row, ok := child.(*notionapi.TableRowBlock)
		if !ok || row == nil {
			continue
		}
		cells := make([]string, 0, len(row.TableRow.Cells))
		for _, cell := range row.TableRow.Cells {
			text := convertRichText(cell)
			text = strings.ReplaceAll(text, "\n", " ")
			text = strings.ReplaceAll(text, "|", "\\|")
			cells = append(cells, text)
		}
		if len(cells) > width {
			width = len(cells)
		}
		rows = append(rows, cells)
	}
	if len(rows) == 0 || width == 0 {
		return ""
	}

	var sb strings.Builder
	writeRow := func(cells []string) {
		sb.WriteString("|")
		for i := 0; i < width; i++ {
			cell := ""
			if i < len(cells) {
				cell = cells[i]
			}
			sb.WriteString(" " + cell + " |")
		}
		sb.WriteString("\n")
	}

	// Markdown tables need a header row, so the first row always acts as one.
	writeRow(rows[0])
	sb.WriteString("|" + strings.Repeat(" --- |", width) + "\n")
	for _, cells := range rows[1:] {
		writeRow(cells)
	}
	return strings.TrimRight(sb.String(), "\n")
}

// fileURL returns the URL of a Notion-hosted or external file object.
func fileURL(file, external *notionapi.FileObject) string {
	if file != nil && file.URL != "" {
		return file.URL
	}
	if external != nil {
		return external.URL
	}
	return ""
}

// convertLink renders a labelled Markdown link, preferring the caption as label.
func convertLink(label string, caption []notionapi.RichText, url string) string {
	if url == "" {
		return ""
	}
	if text := GetRichTextString(caption); text != "" {
		label = text
	}
	return fmt.Sprintf("[%s](%s)", label, url)
}

// convertLinkToPage renders a link_to_page block as a link to the Notion page.
func convertLinkToPage(block *notionapi.LinkToPageBlock) string {
	if block == nil {
		return ""
	}
	id := string(block.LinkToPage.PageID)
	label := "Linked page"
	if id == "" {
		id = string(block.LinkToPage.DatabaseID)
		label = "Linked database"
	}
	if id == "" {
		return ""
	}
	return fmt.Sprintf("[%s](https://www.notion.so/%s)", label, strings.ReplaceAll(id, "-", ""))
}

// convertRichText converts a slice of RichText to formatted Markdown.
func convertRichText(text []notionapi.RichText) string {
	if len(text) == 0 {
		return ""
	}

	var result strings.Builder
	for _, rt := range text {
		result.WriteString(convertAnnotations(rt))
	}
	return result.String()
}

// convertParagraph converts a paragraph block to Markdown.
func convertParagraph(block *notionapi.ParagraphBlock) string {
	if block == nil {
		return ""
	}
	return convertRichText(block.Paragraph.RichText)
}

// convertHeading1 converts a heading 1 block to Markdown.
func convertHeading1(block *notionapi.Heading1Block) string {
	if block == nil {
		return ""
	}
	return "# " + convertRichText(block.Heading1.RichText)
}

// convertHeading2 converts a heading 2 block to Markdown.
func convertHeading2(block *notionapi.Heading2Block) string {
	if block == nil {
		return ""
	}
	return "## " + convertRichText(block.Heading2.RichText)
}

// convertHeading3 converts a heading 3 block to Markdown.
func convertHeading3(block *notionapi.Heading3Block) string {
	if block == nil {
		return ""
	}
	return "### " + convertRichText(block.Heading3.RichText)
}

// convertBulletedListItem converts a bulleted list item block to Markdown.
func convertBulletedListItem(block *notionapi.BulletedListItemBlock) string {
	if block == nil {
		return ""
	}
	return "- " + convertRichText(block.BulletedListItem.RichText)
}

// convertNumberedListItem converts a numbered list item block to Markdown.
func convertNumberedListItem(block *notionapi.NumberedListItemBlock, listCtx *listState) string {
	if block == nil {
		return ""
	}
	counter := 1
	if listCtx != nil {
		counter = listCtx.counter
	}
	return fmt.Sprintf("%d. %s", counter, convertRichText(block.NumberedListItem.RichText))
}

// convertToDo converts a to-do block to Markdown checkbox.
func convertToDo(block *notionapi.ToDoBlock) string {
	if block == nil {
		return ""
	}
	text := convertRichText(block.ToDo.RichText)
	return CreateCheckbox(block.ToDo.Checked, text)
}

// convertQuote converts a quote block to Markdown blockquote.
func convertQuote(block *notionapi.QuoteBlock) string {
	if block == nil {
		return ""
	}
	text := convertRichText(block.Quote.RichText)
	// Handle multi-line quotes
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = "> " + line
	}
	return strings.Join(lines, "\n")
}

// convertCode converts a code block to Markdown code fence.
func convertCode(block *notionapi.CodeBlock) string {
	if block == nil {
		return ""
	}
	code := GetRichTextString(block.Code.RichText)
	language := block.Code.Language
	return FormatCode(code, language)
}

// convertImage converts an image block to Markdown image syntax.
func convertImage(block *notionapi.ImageBlock) string {
	if block == nil {
		return ""
	}

	var url string
	switch block.Image.Type {
	case notionapi.FileTypeExternal:
		if block.Image.External != nil {
			url = block.Image.External.URL
		}
	case notionapi.FileTypeFile:
		if block.Image.File != nil {
			url = block.Image.File.URL
		}
	}

	caption := ""
	if len(block.Image.Caption) > 0 {
		caption = GetRichTextString(block.Image.Caption)
	}

	if url == "" {
		return ""
	}

	return fmt.Sprintf("![%s](%s)", caption, url)
}

// convertBookmark converts a bookmark block to Markdown link.
func convertBookmark(block *notionapi.BookmarkBlock) string {
	if block == nil {
		return ""
	}

	url := block.Bookmark.URL
	caption := url
	if len(block.Bookmark.Caption) > 0 {
		caption = GetRichTextString(block.Bookmark.Caption)
	}

	return fmt.Sprintf("[%s](%s)", caption, url)
}

// convertCallout converts a callout block to Markdown blockquote with emoji.
func convertCallout(block *notionapi.CalloutBlock) string {
	if block == nil {
		return ""
	}

	var emoji string
	if block.Callout.Icon != nil && block.Callout.Icon.Emoji != nil {
		emoji = string(*block.Callout.Icon.Emoji) + " "
	}

	text := convertRichText(block.Callout.RichText)
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if i == 0 {
			lines[i] = "> " + emoji + line
		} else {
			lines[i] = "> " + line
		}
	}
	return strings.Join(lines, "\n")
}
