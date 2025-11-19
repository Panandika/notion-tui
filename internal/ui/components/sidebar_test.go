package components

import (
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestItem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		title      string
		desc       string
		id         string
		wantTitle  string
		wantDesc   string
		wantFilter string
		wantID     string
	}{
		{
			name:       "basic item",
			title:      "Test Page",
			desc:       "A test page",
			id:         "page-123",
			wantTitle:  "Test Page",
			wantDesc:   "A test page",
			wantFilter: "Test Page",
			wantID:     "page-123",
		},
		{
			name:       "empty description",
			title:      "Page Without Desc",
			desc:       "",
			id:         "page-456",
			wantTitle:  "Page Without Desc",
			wantDesc:   "",
			wantFilter: "Page Without Desc",
			wantID:     "page-456",
		},
		{
			name:       "special characters",
			title:      "Page @#$% Special!",
			desc:       "Contains <special> chars",
			id:         "id-with-dash",
			wantTitle:  "Page @#$% Special!",
			wantDesc:   "Contains <special> chars",
			wantFilter: "Page @#$% Special!",
			wantID:     "id-with-dash",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			item := NewItem(tt.title, tt.desc, tt.id)

			assert.Equal(t, tt.wantTitle, item.Title())
			assert.Equal(t, tt.wantDesc, item.Description())
			assert.Equal(t, tt.wantFilter, item.FilterValue())
			assert.Equal(t, tt.wantID, item.ID())
		})
	}
}

func TestNewSidebar(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		input          NewSidebarInput
		wantWidth      int
		wantHeight     int
		wantSelectedID string
		wantItemCount  int
		wantTitle      string
	}{
		{
			name: "with items",
			input: NewSidebarInput{
				Items: []Item{
					NewItem("Page 1", "First page", "id-1"),
					NewItem("Page 2", "Second page", "id-2"),
					NewItem("Page 3", "Third page", "id-3"),
				},
				Width:  40,
				Height: 20,
				Title:  "My Pages",
			},
			wantWidth:      40,
			wantHeight:     20,
			wantSelectedID: "id-1",
			wantItemCount:  3,
			wantTitle:      "My Pages",
		},
		{
			name: "empty items",
			input: NewSidebarInput{
				Items:  []Item{},
				Width:  30,
				Height: 15,
				Title:  "Empty List",
			},
			wantWidth:      30,
			wantHeight:     15,
			wantSelectedID: "",
			wantItemCount:  0,
			wantTitle:      "Empty List",
		},
		{
			name: "default title",
			input: NewSidebarInput{
				Items: []Item{
					NewItem("Single", "One item", "single-id"),
				},
				Width:  25,
				Height: 10,
				Title:  "",
			},
			wantWidth:      25,
			wantHeight:     10,
			wantSelectedID: "single-id",
			wantItemCount:  1,
			wantTitle:      "Pages",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sidebar := NewSidebar(tt.input)

			assert.Equal(t, tt.wantWidth, sidebar.Width())
			assert.Equal(t, tt.wantHeight, sidebar.Height())
			assert.Equal(t, tt.wantSelectedID, sidebar.SelectedID())
			assert.Equal(t, tt.wantItemCount, len(sidebar.Items()))
		})
	}
}

func TestSidebarSetItems(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		initialItems   []Item
		newItems       []Item
		wantCount      int
		wantSelectedID string
	}{
		{
			name: "replace all items",
			initialItems: []Item{
				NewItem("Old 1", "Old item", "old-1"),
			},
			newItems: []Item{
				NewItem("New 1", "New item 1", "new-1"),
				NewItem("New 2", "New item 2", "new-2"),
			},
			wantCount:      2,
			wantSelectedID: "new-1",
		},
		{
			name: "clear all items",
			initialItems: []Item{
				NewItem("Item 1", "Desc", "id-1"),
				NewItem("Item 2", "Desc", "id-2"),
			},
			newItems:       []Item{},
			wantCount:      0,
			wantSelectedID: "",
		},
		{
			name:         "add items to empty list",
			initialItems: []Item{},
			newItems: []Item{
				NewItem("First", "First added", "first"),
			},
			wantCount:      1,
			wantSelectedID: "first",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sidebar := NewSidebar(NewSidebarInput{
				Items:  tt.initialItems,
				Width:  40,
				Height: 20,
			})

			sidebar.SetItems(tt.newItems)

			assert.Equal(t, tt.wantCount, len(sidebar.Items()))
			assert.Equal(t, tt.wantSelectedID, sidebar.SelectedID())
		})
	}
}

func TestSidebarSelection(t *testing.T) {
	t.Parallel()

	items := []Item{
		NewItem("Page Alpha", "Alpha description", "alpha"),
		NewItem("Page Beta", "Beta description", "beta"),
		NewItem("Page Gamma", "Gamma description", "gamma"),
	}

	sidebar := NewSidebar(NewSidebarInput{
		Items:  items,
		Width:  40,
		Height: 20,
	})

	assert.Equal(t, "alpha", sidebar.SelectedID())
	assert.Equal(t, 0, sidebar.SelectedIndex())
}

func TestSidebarKeyNavigation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		keys      []string
		wantIndex int
	}{
		{
			name:      "move down once",
			keys:      []string{"down"},
			wantIndex: 1,
		},
		{
			name:      "move down twice",
			keys:      []string{"down", "down"},
			wantIndex: 2,
		},
		{
			name:      "move down then up",
			keys:      []string{"down", "down", "up"},
			wantIndex: 1,
		},
		{
			name:      "j key moves down",
			keys:      []string{"j"},
			wantIndex: 1,
		},
		{
			name:      "k key moves up after down",
			keys:      []string{"j", "k"},
			wantIndex: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			items := []Item{
				NewItem("Item 1", "Desc 1", "id-1"),
				NewItem("Item 2", "Desc 2", "id-2"),
				NewItem("Item 3", "Desc 3", "id-3"),
			}

			sidebar := NewSidebar(NewSidebarInput{
				Items:  items,
				Width:  40,
				Height: 20,
			})

			for _, key := range tt.keys {
				sidebar, _ = sidebar.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
			}

			assert.Equal(t, tt.wantIndex, sidebar.SelectedIndex())
		})
	}
}

func TestSidebarEnterSelect(t *testing.T) {
	t.Parallel()

	items := []Item{
		NewItem("First Page", "First description", "first-id"),
		NewItem("Second Page", "Second description", "second-id"),
	}

	sidebar := NewSidebar(NewSidebarInput{
		Items:  items,
		Width:  40,
		Height: 20,
	})

	sidebar, _ = sidebar.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})

	sidebar, cmd := sidebar.Update(tea.KeyMsg{Type: tea.KeyEnter})

	require.NotNil(t, cmd)

	msg := cmd()
	selectedMsg, ok := msg.(ItemSelectedMsg)
	require.True(t, ok, "expected ItemSelectedMsg")

	assert.Equal(t, "second-id", selectedMsg.ID)
	assert.Equal(t, "Second Page", selectedMsg.Title)
	assert.Equal(t, 1, selectedMsg.Index)
}

func TestSidebarEmptyList(t *testing.T) {
	t.Parallel()

	sidebar := NewSidebar(NewSidebarInput{
		Items:  []Item{},
		Width:  40,
		Height: 20,
		Title:  "Empty",
	})

	assert.Equal(t, "", sidebar.SelectedID())
	assert.Equal(t, 0, len(sidebar.Items()))

	sidebar, cmd := sidebar.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd)

	view := sidebar.View()
	assert.NotEmpty(t, view)
}

func TestSidebarUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		msg       tea.Msg
		checkFunc func(t *testing.T, s Sidebar, cmd tea.Cmd)
	}{
		{
			name: "window size message",
			msg:  tea.WindowSizeMsg{Width: 50, Height: 30},
			checkFunc: func(t *testing.T, s Sidebar, cmd tea.Cmd) {
				assert.Equal(t, 50, s.Width())
				assert.Equal(t, 30, s.Height())
			},
		},
		{
			name: "init returns nil",
			msg:  nil,
			checkFunc: func(t *testing.T, s Sidebar, cmd tea.Cmd) {
				initCmd := s.Init()
				assert.Nil(t, initCmd)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			items := []Item{
				NewItem("Test", "Test desc", "test-id"),
			}

			sidebar := NewSidebar(NewSidebarInput{
				Items:  items,
				Width:  40,
				Height: 20,
			})

			var updatedSidebar Sidebar
			var cmd tea.Cmd

			if tt.msg != nil {
				updatedSidebar, cmd = sidebar.Update(tt.msg)
			} else {
				updatedSidebar = sidebar
			}

			tt.checkFunc(t, updatedSidebar, cmd)
		})
	}
}

func TestSidebarFiltering(t *testing.T) {
	t.Parallel()

	items := []Item{
		NewItem("Apple Document", "Fruit document", "apple"),
		NewItem("Banana Report", "Yellow fruit", "banana"),
		NewItem("Cherry Analysis", "Red fruit", "cherry"),
	}

	sidebar := NewSidebar(NewSidebarInput{
		Items:  items,
		Width:  40,
		Height: 20,
	})

	filterState := sidebar.FilterState()
	assert.Equal(t, list.Unfiltered, filterState)

	assert.False(t, sidebar.IsFiltering())
}

func TestSidebarSetSize(t *testing.T) {
	t.Parallel()

	sidebar := NewSidebar(NewSidebarInput{
		Items:  []Item{NewItem("Test", "Desc", "id")},
		Width:  40,
		Height: 20,
	})

	sidebar.SetSize(60, 35)

	assert.Equal(t, 60, sidebar.Width())
	assert.Equal(t, 35, sidebar.Height())
}

func TestSidebarView(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		items     []Item
		wantEmpty bool
	}{
		{
			name: "with items",
			items: []Item{
				NewItem("Page 1", "Description 1", "id-1"),
				NewItem("Page 2", "Description 2", "id-2"),
			},
			wantEmpty: false,
		},
		{
			name:      "empty list",
			items:     []Item{},
			wantEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sidebar := NewSidebar(NewSidebarInput{
				Items:  tt.items,
				Width:  40,
				Height: 20,
			})

			view := sidebar.View()

			if tt.wantEmpty {
				assert.Empty(t, view)
			} else {
				assert.NotEmpty(t, view)
			}
		})
	}
}

func TestDefaultSidebarStyles(t *testing.T) {
	t.Parallel()

	styles := DefaultSidebarStyles()

	// Test that styles can render text
	testText := "test"
	assert.NotEmpty(t, styles.List.Render(testText))
	assert.NotEmpty(t, styles.Title.Render(testText))
	assert.NotEmpty(t, styles.Item.Render(testText))
	assert.NotEmpty(t, styles.SelectedID.Render(testText))
}

func TestSidebarItems(t *testing.T) {
	t.Parallel()

	originalItems := []Item{
		NewItem("Alpha", "A", "a"),
		NewItem("Beta", "B", "b"),
		NewItem("Gamma", "G", "g"),
	}

	sidebar := NewSidebar(NewSidebarInput{
		Items:  originalItems,
		Width:  40,
		Height: 20,
	})

	retrievedItems := sidebar.Items()

	require.Equal(t, len(originalItems), len(retrievedItems))
	for i, item := range retrievedItems {
		assert.Equal(t, originalItems[i].Title(), item.Title())
		assert.Equal(t, originalItems[i].Description(), item.Description())
		assert.Equal(t, originalItems[i].ID(), item.ID())
	}
}
