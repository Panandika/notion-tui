package notion

import (
	"fmt"
	"strings"

	"github.com/jomei/notionapi"
)

// ConvertBlocksToMarkdown converts a slice of Notion blocks to Markdown string.
func ConvertBlocksToMarkdown(blocks []notionapi.Block) (string, error) {
	if len(blocks) == 0 {
		return "", nil
	}

	var result strings.Builder
	var listContext *listState = nil

	for i, block := range blocks {
		if block == nil {
			continue
		}

		// Handle list numbering context
		blockType := block.GetType()
		if blockType == notionapi.BlockTypeNumberedListItem {
			if listContext == nil {
				listContext = &listState{counter: 1}
			}
		} else if listContext != nil && blockType != notionapi.BlockTypeNumberedListItem {
			listContext = nil
		}

		md, err := convertBlock(block, listContext)
		if err != nil {
			return "", fmt.Errorf("convert block %d: %w", i, err)
		}

		if md != "" {
			result.WriteString(md)
			result.WriteString("\n")

			// Add extra newline after certain block types for better readability
			if shouldAddExtraNewline(blockType) {
				result.WriteString("\n")
			}
		}

		// Increment counter for numbered lists
		if listContext != nil && blockType == notionapi.BlockTypeNumberedListItem {
			listContext.counter++
		}
	}

	return strings.TrimRight(result.String(), "\n"), nil
}

// listState tracks the state for numbered list items.
type listState struct {
	counter int
}

// shouldAddExtraNewline determines if an extra newline should be added after a block.
func shouldAddExtraNewline(blockType notionapi.BlockType) bool {
	switch blockType {
	case notionapi.BlockTypeParagraph,
		notionapi.BlockTypeHeading1,
		notionapi.BlockTypeHeading2,
		notionapi.BlockTypeHeading3,
		notionapi.BlockTypeCode,
		notionapi.BlockTypeQuote,
		notionapi.BlockTypeDivider,
		notionapi.BlockTypeImage,
		notionapi.BlockTypeCallout:
		return true
	default:
		return false
	}
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

	default:
		// Return empty string for unsupported block types
		// This allows graceful handling without errors
		return "", nil
	}
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
	language := string(block.Code.Language)
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
