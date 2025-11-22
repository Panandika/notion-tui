package ui

import (
	"testing"
)

func TestNewNavigator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		input          NewNavigatorInput
		wantCurrent    PageID
		wantMaxHistory int
	}{
		{
			name: "with valid max history",
			input: NewNavigatorInput{
				InitialPage: PageList,
				MaxHistory:  100,
			},
			wantCurrent:    PageList,
			wantMaxHistory: 100,
		},
		{
			name: "with default max history (zero)",
			input: NewNavigatorInput{
				InitialPage: PageDetail,
				MaxHistory:  0,
			},
			wantCurrent:    PageDetail,
			wantMaxHistory: DefaultMaxHistory,
		},
		{
			name: "with negative max history",
			input: NewNavigatorInput{
				InitialPage: PageEdit,
				MaxHistory:  -1,
			},
			wantCurrent:    PageEdit,
			wantMaxHistory: DefaultMaxHistory,
		},
		{
			name: "with empty initial page",
			input: NewNavigatorInput{
				InitialPage: PageID(""),
				MaxHistory:  50,
			},
			wantCurrent:    PageID(""),
			wantMaxHistory: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			nav := NewNavigator(tt.input)

			if nav.CurrentPage() != tt.wantCurrent {
				t.Errorf("CurrentPage() = %v, want %v", nav.CurrentPage(), tt.wantCurrent)
			}

			if nav.maxHistory != tt.wantMaxHistory {
				t.Errorf("maxHistory = %v, want %v", nav.maxHistory, tt.wantMaxHistory)
			}

			if len(nav.History()) != 0 {
				t.Errorf("History() length = %v, want 0", len(nav.History()))
			}
		})
	}
}

func TestNavigator_NavigateTo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		initialPage        PageID
		navigations        []PageID
		wantCurrent        PageID
		wantHistoryLen     int
		wantFirstInHistory PageID
	}{
		{
			name:               "single navigation",
			initialPage:        PageList,
			navigations:        []PageID{PageDetail},
			wantCurrent:        PageDetail,
			wantHistoryLen:     1,
			wantFirstInHistory: PageList,
		},
		{
			name:               "multiple navigations",
			initialPage:        PageList,
			navigations:        []PageID{PageDetail, PageEdit, PageList},
			wantCurrent:        PageList,
			wantHistoryLen:     3,
			wantFirstInHistory: PageList,
		},
		{
			name:               "navigation from empty page",
			initialPage:        PageID(""),
			navigations:        []PageID{PageList},
			wantCurrent:        PageList,
			wantHistoryLen:     0, // Empty page not added to history
			wantFirstInHistory: PageID(""),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			nav := NewNavigator(NewNavigatorInput{
				InitialPage: tt.initialPage,
				MaxHistory:  50,
			})

			for _, page := range tt.navigations {
				nav.NavigateTo(page)
			}

			if nav.CurrentPage() != tt.wantCurrent {
				t.Errorf("CurrentPage() = %v, want %v", nav.CurrentPage(), tt.wantCurrent)
			}

			history := nav.History()
			if len(history) != tt.wantHistoryLen {
				t.Errorf("History() length = %v, want %v", len(history), tt.wantHistoryLen)
			}

			if tt.wantHistoryLen > 0 && history[0] != tt.wantFirstInHistory {
				t.Errorf("First history entry = %v, want %v", history[0], tt.wantFirstInHistory)
			}
		})
	}
}

func TestNavigator_NavigateTo_HistorySizeLimit(t *testing.T) {
	t.Parallel()

	nav := NewNavigator(NewNavigatorInput{
		InitialPage: PageList,
		MaxHistory:  3,
	})

	// Navigate 5 times (should keep only last 3 in history)
	nav.NavigateTo(PageID("page1"))
	nav.NavigateTo(PageID("page2"))
	nav.NavigateTo(PageID("page3"))
	nav.NavigateTo(PageID("page4"))
	nav.NavigateTo(PageID("page5"))

	history := nav.History()
	if len(history) != 3 {
		t.Errorf("History() length = %v, want 3", len(history))
	}

	// Should have page2, page3, page4 (oldest page1 and PageList removed)
	expectedHistory := []PageID{PageID("page2"), PageID("page3"), PageID("page4")}
	for i, expected := range expectedHistory {
		if history[i] != expected {
			t.Errorf("History()[%d] = %v, want %v", i, history[i], expected)
		}
	}

	if nav.CurrentPage() != PageID("page5") {
		t.Errorf("CurrentPage() = %v, want page5", nav.CurrentPage())
	}
}

func TestNavigator_Back(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setup       func() Navigator
		wantPage    PageID
		wantOK      bool
		wantCurrent PageID
	}{
		{
			name: "back from single navigation",
			setup: func() Navigator {
				nav := NewNavigator(NewNavigatorInput{
					InitialPage: PageList,
					MaxHistory:  50,
				})
				nav.NavigateTo(PageDetail)
				return nav
			},
			wantPage:    PageList,
			wantOK:      true,
			wantCurrent: PageList,
		},
		{
			name: "back with no history",
			setup: func() Navigator {
				return NewNavigator(NewNavigatorInput{
					InitialPage: PageList,
					MaxHistory:  50,
				})
			},
			wantPage:    PageID(""),
			wantOK:      false,
			wantCurrent: PageList,
		},
		{
			name: "multiple backs",
			setup: func() Navigator {
				nav := NewNavigator(NewNavigatorInput{
					InitialPage: PageList,
					MaxHistory:  50,
				})
				nav.NavigateTo(PageDetail)
				nav.NavigateTo(PageEdit)
				// First back should return PageDetail
				nav.Back()
				return nav
			},
			wantPage:    PageList,
			wantOK:      true,
			wantCurrent: PageList,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			nav := tt.setup()
			gotPage, gotOK := nav.Back()

			if gotPage != tt.wantPage {
				t.Errorf("Back() page = %v, want %v", gotPage, tt.wantPage)
			}

			if gotOK != tt.wantOK {
				t.Errorf("Back() ok = %v, want %v", gotOK, tt.wantOK)
			}

			if nav.CurrentPage() != tt.wantCurrent {
				t.Errorf("CurrentPage() = %v, want %v", nav.CurrentPage(), tt.wantCurrent)
			}
		})
	}
}

func TestNavigator_CanGoBack(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func() Navigator
		want  bool
	}{
		{
			name: "can go back after navigation",
			setup: func() Navigator {
				nav := NewNavigator(NewNavigatorInput{
					InitialPage: PageList,
					MaxHistory:  50,
				})
				nav.NavigateTo(PageDetail)
				return nav
			},
			want: true,
		},
		{
			name: "cannot go back with no history",
			setup: func() Navigator {
				return NewNavigator(NewNavigatorInput{
					InitialPage: PageList,
					MaxHistory:  50,
				})
			},
			want: false,
		},
		{
			name: "cannot go back after clearing history",
			setup: func() Navigator {
				nav := NewNavigator(NewNavigatorInput{
					InitialPage: PageList,
					MaxHistory:  50,
				})
				nav.NavigateTo(PageDetail)
				nav.ClearHistory()
				return nav
			},
			want: false,
		},
		{
			name: "cannot go back after exhausting history",
			setup: func() Navigator {
				nav := NewNavigator(NewNavigatorInput{
					InitialPage: PageList,
					MaxHistory:  50,
				})
				nav.NavigateTo(PageDetail)
				nav.Back()
				return nav
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			nav := tt.setup()
			got := nav.CanGoBack()

			if got != tt.want {
				t.Errorf("CanGoBack() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNavigator_History(t *testing.T) {
	t.Parallel()

	nav := NewNavigator(NewNavigatorInput{
		InitialPage: PageList,
		MaxHistory:  50,
	})

	nav.NavigateTo(PageDetail)
	nav.NavigateTo(PageEdit)

	history := nav.History()

	// Test that we get a copy
	history[0] = PageID("modified")

	// Original history should be unchanged
	originalHistory := nav.History()
	if originalHistory[0] == PageID("modified") {
		t.Error("History() should return a copy, not the original slice")
	}

	// Verify expected history
	expected := []PageID{PageList, PageDetail}
	if len(originalHistory) != len(expected) {
		t.Errorf("History() length = %v, want %v", len(originalHistory), len(expected))
	}

	for i, page := range expected {
		if originalHistory[i] != page {
			t.Errorf("History()[%d] = %v, want %v", i, originalHistory[i], page)
		}
	}
}

func TestNavigator_ClearHistory(t *testing.T) {
	t.Parallel()

	nav := NewNavigator(NewNavigatorInput{
		InitialPage: PageList,
		MaxHistory:  50,
	})

	nav.NavigateTo(PageDetail)
	nav.NavigateTo(PageEdit)

	currentBefore := nav.CurrentPage()
	nav.ClearHistory()

	if len(nav.History()) != 0 {
		t.Errorf("History() length after clear = %v, want 0", len(nav.History()))
	}

	if nav.CurrentPage() != currentBefore {
		t.Errorf("CurrentPage() changed after clear: got %v, want %v", nav.CurrentPage(), currentBefore)
	}

	if nav.CanGoBack() {
		t.Error("CanGoBack() = true after clear, want false")
	}
}

func TestNavigator_Reset(t *testing.T) {
	t.Parallel()

	nav := NewNavigator(NewNavigatorInput{
		InitialPage: PageList,
		MaxHistory:  50,
	})

	nav.NavigateTo(PageDetail)
	nav.NavigateTo(PageEdit)

	resetPage := PageID("reset-page")
	nav.Reset(resetPage)

	if nav.CurrentPage() != resetPage {
		t.Errorf("CurrentPage() after reset = %v, want %v", nav.CurrentPage(), resetPage)
	}

	if len(nav.History()) != 0 {
		t.Errorf("History() length after reset = %v, want 0", len(nav.History()))
	}

	if nav.CanGoBack() {
		t.Error("CanGoBack() = true after reset, want false")
	}
}

func TestNavigator_ComplexNavigationScenario(t *testing.T) {
	t.Parallel()

	nav := NewNavigator(NewNavigatorInput{
		InitialPage: PageList,
		MaxHistory:  50,
	})

	// Navigate forward
	nav.NavigateTo(PageID("page1"))
	nav.NavigateTo(PageID("page2"))
	nav.NavigateTo(PageID("page3"))

	// Go back twice
	page, ok := nav.Back()
	if !ok || page != PageID("page2") {
		t.Errorf("First back returned %v, %v; want page2, true", page, ok)
	}

	page, ok = nav.Back()
	if !ok || page != PageID("page1") {
		t.Errorf("Second back returned %v, %v; want page1, true", page, ok)
	}

	// Navigate forward from middle of history
	nav.NavigateTo(PageID("page4"))

	// After backing twice from page3, we're at page1 with history [PageList]
	// Navigating to page4 pushes page1 to history: [PageList, page1]
	// Current should be: page4
	if nav.CurrentPage() != PageID("page4") {
		t.Errorf("CurrentPage() = %v, want page4", nav.CurrentPage())
	}

	history := nav.History()
	expectedLen := 2
	if len(history) != expectedLen {
		t.Errorf("History() length = %v, want %v (history: %v)", len(history), expectedLen, history)
	}

	// Verify history contents
	if len(history) >= 2 {
		if history[0] != PageList {
			t.Errorf("History()[0] = %v, want PageList", history[0])
		}
		if history[1] != PageID("page1") {
			t.Errorf("History()[1] = %v, want page1", history[1])
		}
	}
}
