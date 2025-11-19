package notion

import (
	"fmt"
	"strings"

	"github.com/jomei/notionapi"
)

// GetRichTextString extracts plain text from a slice of RichText objects.
func GetRichTextString(text []notionapi.RichText) string {
	if len(text) == 0 {
		return ""
	}

	var sb strings.Builder
	for _, rt := range text {
		sb.WriteString(rt.PlainText)
	}
	return sb.String()
}

// HasChildren checks if a block has child blocks.
func HasChildren(block notionapi.Block) bool {
	if block == nil {
		return false
	}
	return block.GetHasChildren()
}

// GetBlockType returns the type of a block as a string.
func GetBlockType(block notionapi.Block) string {
	if block == nil {
		return ""
	}
	return string(block.GetType())
}

// ExtractLinks extracts all URLs from a slice of RichText objects.
func ExtractLinks(text []notionapi.RichText) []string {
	if len(text) == 0 {
		return nil
	}

	var links []string
	for _, rt := range text {
		if rt.Href != "" {
			links = append(links, rt.Href)
		}
	}
	return links
}

// FormatCode formats code with optional language identifier for markdown code blocks.
func FormatCode(code string, language string) string {
	if language == "" {
		language = "text"
	}
	return fmt.Sprintf("```%s\n%s\n```", language, code)
}

// CreateCheckbox creates a markdown checkbox string.
func CreateCheckbox(checked bool, text string) string {
	if checked {
		return fmt.Sprintf("- [x] %s", text)
	}
	return fmt.Sprintf("- [ ] %s", text)
}

// convertAnnotations applies markdown formatting based on RichText annotations.
func convertAnnotations(rt notionapi.RichText) string {
	text := rt.PlainText
	if text == "" {
		return ""
	}

	// Apply annotations in a specific order to ensure proper nesting
	if rt.Annotations != nil {
		if rt.Annotations.Code {
			text = "`" + text + "`"
		}
		if rt.Annotations.Strikethrough {
			text = "~~" + text + "~~"
		}
		if rt.Annotations.Underline {
			// Markdown doesn't have underline, use HTML
			text = "<u>" + text + "</u>"
		}
		if rt.Annotations.Italic {
			text = "*" + text + "*"
		}
		if rt.Annotations.Bold {
			text = "**" + text + "**"
		}
	}

	// Handle links
	if rt.Href != "" {
		text = fmt.Sprintf("[%s](%s)", text, rt.Href)
	}

	return text
}
