package notion

import (
	"testing"

	"github.com/jomei/notionapi"
	"github.com/stretchr/testify/assert"
)

func TestGetRichTextString(t *testing.T) {
	t.Parallel()

	t.Run("GetRichTextString", func(t *testing.T) {
		t.Parallel()

		text := []notionapi.RichText{
			{PlainText: "Hello "},
			{PlainText: "World"},
		}
		result := GetRichTextString(text)
		assert.Equal(t, "Hello World", result)
	})

	t.Run("GetRichTextString empty", func(t *testing.T) {
		t.Parallel()

		result := GetRichTextString([]notionapi.RichText{})
		assert.Equal(t, "", result)
	})
}
