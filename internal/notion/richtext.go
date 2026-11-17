package notion

import (
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
