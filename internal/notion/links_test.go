package notion

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	idA = "4169a22b64ff806c9c42d2caf2d80d06"
	idB = "b2f9a22b64ff832981390180e2c765dd"
	idC = "0c59a22b64ff821e94bd811369732018"
)

func TestPageLinks(t *testing.T) {
	t.Parallel()

	markdown := "Intro with a [guide](https://www.notion.com/help/guides) and an image ![](https://x.test/i.png).\n" +
		"<page url=\"https://app.notion.com/p/" + idA + "\">Example sub-page</page>\n" +
		"<database url=\"https://app.notion.com/p/" + idB + "\" inline=\"true\">\n" +
		"Income (Monthly)\n" +
		"</database>\n" +
		"See <mention url=\"https://app.notion.com/p/" + idC + "\">Other <b>page</b></mention> and [again](https://app.notion.com/p/" + idC + ").\n" +
		"```\n[not a link](https://code.test)\n```\n" +
		"[mail](mailto:a@b.c) [bad](javascript:alert(1)) [old style](https://www.notion.so/My-Title-" + idA + ")\n"

	links := PageLinks(markdown)

	assert.Equal(t, []PageLink{
		{Kind: LinkWeb, Title: "guide", URL: "https://www.notion.com/help/guides"},
		{Kind: LinkPage, Title: "Example sub-page", URL: "https://app.notion.com/p/" + idA, ID: idA},
		{Kind: LinkDatabase, Title: "Income (Monthly)", URL: "https://app.notion.com/p/" + idB, ID: idB},
		{Kind: LinkPage, Title: "Other **page**", URL: "https://app.notion.com/p/" + idC, ID: idC},
	}, links, "images, code, non-web schemes and repeats of the same target are left out")
}

func TestPageLinks_Empty(t *testing.T) {
	t.Parallel()

	assert.Empty(t, PageLinks(""))
	assert.Empty(t, PageLinks("plain text\n- [ ] a to-do"))
}

func TestNotionID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		url  string
		want string
	}{
		{name: "app link", url: "https://app.notion.com/p/" + idA, want: idA},
		{name: "titled link", url: "https://www.notion.so/workspace/My-Page-" + idA, want: idA},
		{name: "with query and fragment", url: "https://app.notion.com/p/" + idA + "?v=1#abc", want: idA},
		{name: "upper case id is normalized", url: "https://app.notion.com/p/4169A22B64FF806C9C42D2CAF2D80D06", want: idA},
		{name: "not a notion host", url: "https://evil.test/p/" + idA, want: ""},
		{name: "look-alike host", url: "https://notion.so.evil.test/p/" + idA, want: ""},
		{name: "no id", url: "https://www.notion.com/help/guides", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, notionID(tt.url))
		})
	}
}

func TestPageLinks_UnclosedBlockDoesNotSwallowWhatFollows(t *testing.T) {
	t.Parallel()

	markdown := "<page url=\"https://app.notion.com/p/" + idA + "\">\n" +
		"\n" +
		"First\n" +
		"A [web link](https://example.test/a) after it.\n" +
		"<database url=\"https://app.notion.com/p/" + idB + "\">\n" +
		"<page url=\"https://app.notion.com/p/" + idC + "\">\n"

	assert.Equal(t, []PageLink{
		{Kind: LinkPage, Title: "First", URL: "https://app.notion.com/p/" + idA, ID: idA},
		{Kind: LinkWeb, Title: "web link", URL: "https://example.test/a"},
		{Kind: LinkDatabase, Title: "https://app.notion.com/p/" + idB, URL: "https://app.notion.com/p/" + idB, ID: idB},
		{Kind: LinkPage, Title: "https://app.notion.com/p/" + idC, URL: "https://app.notion.com/p/" + idC, ID: idC},
	}, PageLinks(markdown), "a tag takes one line as its title; one left open at the end still counts")
}

func TestPageLinks_BlockWithoutNotionIDIsNotAPage(t *testing.T) {
	t.Parallel()

	links := PageLinks("<page url=\"https://evil.test/x\">Trap</page>\n<page url=\"javascript:alert(1)\">Worse</page>")

	assert.Equal(t, []PageLink{
		{Kind: LinkWeb, Title: "Trap", URL: "https://evil.test/x"},
	}, links, "no ID means nothing to open in the app")
}

func TestPageLinks_SelfClosingMention(t *testing.T) {
	t.Parallel()

	markdown := "See <mention-page url=\"https://app.notion.com/p/" + idA + "\"/> and " +
		"<mention-page url=\"https://app.notion.com/p/" + idB + "\">Second</mention-page>."

	assert.Equal(t, []PageLink{
		{Kind: LinkPage, Title: "https://app.notion.com/p/" + idA, URL: "https://app.notion.com/p/" + idA, ID: idA},
		{Kind: LinkPage, Title: "Second", URL: "https://app.notion.com/p/" + idB, ID: idB},
	}, PageLinks(markdown))
}

func TestPageLinks_URLWithParentheses(t *testing.T) {
	t.Parallel()

	links := PageLinks("Read [Go](https://en.wikipedia.org/wiki/Go_(programming_language)) (really).")

	assert.Equal(t, []PageLink{
		{Kind: LinkWeb, Title: "Go", URL: "https://en.wikipedia.org/wiki/Go_(programming_language)"},
	}, links)
}

func TestNotionID_PeekLinkIsTheRow(t *testing.T) {
	t.Parallel()

	assert.Equal(t, idB, notionID("https://www.notion.so/ws/"+idA+"?v=1&p="+idB+"&pm=s"))
	assert.Equal(t, idA, notionID("https://www.notion.so/ws/"+idA+"?p=not-an-id"))
}
