package ui

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLayoutSidebarMain(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		input            LayoutSidebarMainInput
		wantEmpty        bool
		wantContains     []string
		wantMinWidth     int
		wantMinLineCount int
	}{
		{
			name: "standard layout",
			input: LayoutSidebarMainInput{
				Sidebar:      "Sidebar\nContent",
				Main:         "Main\nContent",
				SidebarWidth: 20,
				TotalWidth:   80,
				TotalHeight:  24,
			},
			wantEmpty:        false,
			wantContains:     []string{"Sidebar", "Main"},
			wantMinLineCount: 24,
		},
		{
			name: "minimal dimensions",
			input: LayoutSidebarMainInput{
				Sidebar:      "S",
				Main:         "M",
				SidebarWidth: 5,
				TotalWidth:   10,
				TotalHeight:  3,
			},
			wantEmpty:        false,
			wantContains:     []string{"S", "M"},
			wantMinLineCount: 3,
		},
		{
			name: "invalid zero width",
			input: LayoutSidebarMainInput{
				Sidebar:      "Sidebar",
				Main:         "Main",
				SidebarWidth: 0,
				TotalWidth:   80,
				TotalHeight:  24,
			},
			wantEmpty: true,
		},
		{
			name: "invalid zero height",
			input: LayoutSidebarMainInput{
				Sidebar:      "Sidebar",
				Main:         "Main",
				SidebarWidth: 20,
				TotalWidth:   80,
				TotalHeight:  0,
			},
			wantEmpty: true,
		},
		{
			name: "invalid sidebar wider than total",
			input: LayoutSidebarMainInput{
				Sidebar:      "Sidebar",
				Main:         "Main",
				SidebarWidth: 100,
				TotalWidth:   80,
				TotalHeight:  24,
			},
			wantEmpty: true,
		},
		{
			name: "negative dimensions",
			input: LayoutSidebarMainInput{
				Sidebar:      "Sidebar",
				Main:         "Main",
				SidebarWidth: -10,
				TotalWidth:   80,
				TotalHeight:  24,
			},
			wantEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := LayoutSidebarMain(tt.input)

			if tt.wantEmpty {
				assert.Empty(t, result, "Expected empty result for invalid dimensions")
				return
			}

			assert.NotEmpty(t, result, "Expected non-empty result")

			for _, want := range tt.wantContains {
				assert.Contains(t, result, want, "Result should contain expected content")
			}

			if tt.wantMinLineCount > 0 {
				lines := strings.Split(result, "\n")
				assert.GreaterOrEqual(t, len(lines), tt.wantMinLineCount,
					"Result should have minimum number of lines")
			}
		})
	}
}

func TestLayoutWithStatusBar(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		input            LayoutWithStatusBarInput
		wantEmpty        bool
		wantContains     []string
		wantMinLineCount int
	}{
		{
			name: "standard layout",
			input: LayoutWithStatusBarInput{
				Content:   "Main content\nLine 2\nLine 3",
				StatusBar: "Status: Ready",
				Height:    10,
			},
			wantEmpty:        false,
			wantContains:     []string{"Main content", "Status: Ready"},
			wantMinLineCount: 1,
		},
		{
			name: "minimal height",
			input: LayoutWithStatusBarInput{
				Content:   "Content",
				StatusBar: "Status",
				Height:    1,
			},
			wantEmpty:        false,
			wantContains:     []string{"Status"},
			wantMinLineCount: 1,
		},
		{
			name: "invalid zero height",
			input: LayoutWithStatusBarInput{
				Content:   "Content",
				StatusBar: "Status",
				Height:    0,
			},
			wantEmpty: true,
		},
		{
			name: "invalid negative height",
			input: LayoutWithStatusBarInput{
				Content:   "Content",
				StatusBar: "Status",
				Height:    -5,
			},
			wantEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := LayoutWithStatusBar(tt.input)

			if tt.wantEmpty {
				assert.Empty(t, result, "Expected empty result for invalid dimensions")
				return
			}

			assert.NotEmpty(t, result, "Expected non-empty result")

			for _, want := range tt.wantContains {
				assert.Contains(t, result, want, "Result should contain expected content")
			}
		})
	}
}

func TestLayoutCommandPalette(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		input        LayoutCommandPaletteInput
		wantContains []string
	}{
		{
			name: "standard overlay",
			input: LayoutCommandPaletteInput{
				Background: "Background content",
				Palette:    "Command: Run",
				Width:      80,
				Height:     24,
			},
			wantContains: []string{"Command: Run"},
		},
		{
			name: "minimal dimensions",
			input: LayoutCommandPaletteInput{
				Background: "BG",
				Palette:    "CMD",
				Width:      10,
				Height:     5,
			},
			wantContains: []string{"CMD"},
		},
		{
			name: "invalid zero dimensions returns palette",
			input: LayoutCommandPaletteInput{
				Background: "BG",
				Palette:    "CMD",
				Width:      0,
				Height:     0,
			},
			wantContains: []string{"CMD"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := LayoutCommandPalette(tt.input)

			assert.NotEmpty(t, result, "Result should never be empty")

			for _, want := range tt.wantContains {
				assert.Contains(t, result, want, "Result should contain expected content")
			}
		})
	}
}

func TestCenterOverlay(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		input        CenterOverlayInput
		wantContains []string
	}{
		{
			name: "standard overlay",
			input: CenterOverlayInput{
				Overlay: "Centered Content",
				Width:   80,
				Height:  24,
			},
			wantContains: []string{"Centered Content"},
		},
		{
			name: "minimal dimensions",
			input: CenterOverlayInput{
				Overlay: "C",
				Width:   5,
				Height:  3,
			},
			wantContains: []string{"C"},
		},
		{
			name: "invalid zero dimensions returns overlay",
			input: CenterOverlayInput{
				Overlay: "Overlay",
				Width:   0,
				Height:  0,
			},
			wantContains: []string{"Overlay"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := CenterOverlay(tt.input)

			assert.NotEmpty(t, result, "Result should never be empty")

			for _, want := range tt.wantContains {
				assert.Contains(t, result, want, "Result should contain expected content")
			}
		})
	}
}

func TestLayoutThreeColumn(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		input        LayoutThreeColumnInput
		wantEmpty    bool
		wantContains []string
	}{
		{
			name: "standard three column",
			input: LayoutThreeColumnInput{
				Left:   "Left Panel",
				Center: "Center Panel",
				Right:  "Right Panel",
				Widths: [3]int{20, 40, 20},
				Height: 24,
			},
			wantEmpty:    false,
			wantContains: []string{"Left Panel", "Center Panel", "Right Panel"},
		},
		{
			name: "minimal dimensions",
			input: LayoutThreeColumnInput{
				Left:   "L",
				Center: "C",
				Right:  "R",
				Widths: [3]int{1, 1, 1},
				Height: 3,
			},
			wantEmpty:    false,
			wantContains: []string{"L", "C", "R"},
		},
		{
			name: "invalid zero height",
			input: LayoutThreeColumnInput{
				Left:   "L",
				Center: "C",
				Right:  "R",
				Widths: [3]int{10, 10, 10},
				Height: 0,
			},
			wantEmpty: true,
		},
		{
			name: "invalid negative width",
			input: LayoutThreeColumnInput{
				Left:   "L",
				Center: "C",
				Right:  "R",
				Widths: [3]int{-5, 10, 10},
				Height: 24,
			},
			wantEmpty: true,
		},
		{
			name: "invalid all zero widths",
			input: LayoutThreeColumnInput{
				Left:   "L",
				Center: "C",
				Right:  "R",
				Widths: [3]int{0, 0, 0},
				Height: 24,
			},
			wantEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := LayoutThreeColumn(tt.input)

			if tt.wantEmpty {
				assert.Empty(t, result, "Expected empty result for invalid dimensions")
				return
			}

			assert.NotEmpty(t, result, "Expected non-empty result")

			for _, want := range tt.wantContains {
				assert.Contains(t, result, want, "Result should contain expected content")
			}
		})
	}
}

func TestLayoutBoxedContent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		input        LayoutBoxedContentInput
		wantEmpty    bool
		wantContains []string
	}{
		{
			name: "with title",
			input: LayoutBoxedContentInput{
				Content: "Box content here",
				Title:   "Box Title",
				Width:   40,
				Height:  10,
			},
			wantEmpty:    false,
			wantContains: []string{"Box content here", "Box Title"},
		},
		{
			name: "without title",
			input: LayoutBoxedContentInput{
				Content: "Content only",
				Title:   "",
				Width:   30,
				Height:  8,
			},
			wantEmpty:    false,
			wantContains: []string{"Content only"},
		},
		{
			name: "minimal dimensions",
			input: LayoutBoxedContentInput{
				Content: "C",
				Title:   "T",
				Width:   5,
				Height:  3,
			},
			wantEmpty:    false,
			wantContains: []string{"C", "T"},
		},
		{
			name: "invalid zero width",
			input: LayoutBoxedContentInput{
				Content: "Content",
				Title:   "Title",
				Width:   0,
				Height:  10,
			},
			wantEmpty: true,
		},
		{
			name: "invalid zero height",
			input: LayoutBoxedContentInput{
				Content: "Content",
				Title:   "Title",
				Width:   40,
				Height:  0,
			},
			wantEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := LayoutBoxedContent(tt.input)

			if tt.wantEmpty {
				assert.Empty(t, result, "Expected empty result for invalid dimensions")
				return
			}

			assert.NotEmpty(t, result, "Expected non-empty result")

			for _, want := range tt.wantContains {
				assert.Contains(t, result, want, "Result should contain expected content")
			}
		})
	}
}

func TestLayoutSplitVertical(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		input        LayoutSplitVerticalInput
		wantEmpty    bool
		wantContains []string
	}{
		{
			name: "50-50 split",
			input: LayoutSplitVerticalInput{
				Top:    "Top Section",
				Bottom: "Bottom Section",
				Height: 20,
				Split:  0.5,
			},
			wantEmpty:    false,
			wantContains: []string{"Top Section", "Bottom Section"},
		},
		{
			name: "70-30 split",
			input: LayoutSplitVerticalInput{
				Top:    "Top Section",
				Bottom: "Bottom Section",
				Height: 10,
				Split:  0.7,
			},
			wantEmpty:    false,
			wantContains: []string{"Top Section", "Bottom Section"},
		},
		{
			name: "minimal height",
			input: LayoutSplitVerticalInput{
				Top:    "T",
				Bottom: "B",
				Height: 2,
				Split:  0.5,
			},
			wantEmpty:    false,
			wantContains: []string{"T", "B"},
		},
		{
			name: "invalid zero height",
			input: LayoutSplitVerticalInput{
				Top:    "Top",
				Bottom: "Bottom",
				Height: 0,
				Split:  0.5,
			},
			wantEmpty: true,
		},
		{
			name: "invalid split below range",
			input: LayoutSplitVerticalInput{
				Top:    "Top",
				Bottom: "Bottom",
				Height: 10,
				Split:  -0.1,
			},
			wantEmpty: true,
		},
		{
			name: "invalid split above range",
			input: LayoutSplitVerticalInput{
				Top:    "Top",
				Bottom: "Bottom",
				Height: 10,
				Split:  1.5,
			},
			wantEmpty: true,
		},
		{
			name: "edge case split 0.0",
			input: LayoutSplitVerticalInput{
				Top:    "Top",
				Bottom: "Bottom",
				Height: 10,
				Split:  0.0,
			},
			wantEmpty:    false,
			wantContains: []string{"Bottom"},
		},
		{
			name: "edge case split 1.0",
			input: LayoutSplitVerticalInput{
				Top:    "Top",
				Bottom: "Bottom",
				Height: 10,
				Split:  1.0,
			},
			wantEmpty:    false,
			wantContains: []string{"Top"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := LayoutSplitVertical(tt.input)

			if tt.wantEmpty {
				assert.Empty(t, result, "Expected empty result for invalid dimensions")
				return
			}

			assert.NotEmpty(t, result, "Expected non-empty result")

			for _, want := range tt.wantContains {
				assert.Contains(t, result, want, "Result should contain expected content")
			}
		})
	}
}

func TestLayoutIntegration(t *testing.T) {
	t.Parallel()

	// Test combining multiple layout functions
	t.Run("sidebar with status bar", func(t *testing.T) {
		t.Parallel()

		sidebar := "Sidebar"
		main := "Main content"

		sidebarLayout := LayoutSidebarMain(LayoutSidebarMainInput{
			Sidebar:      sidebar,
			Main:         main,
			SidebarWidth: 20,
			TotalWidth:   80,
			TotalHeight:  23,
		})

		finalLayout := LayoutWithStatusBar(LayoutWithStatusBarInput{
			Content:   sidebarLayout,
			StatusBar: "Status: Ready",
			Height:    24,
		})

		assert.NotEmpty(t, finalLayout)
		assert.Contains(t, finalLayout, "Sidebar")
		assert.Contains(t, finalLayout, "Main content")
		assert.Contains(t, finalLayout, "Status: Ready")
	})

	t.Run("three column with boxed content", func(t *testing.T) {
		t.Parallel()

		leftBox := LayoutBoxedContent(LayoutBoxedContentInput{
			Content: "Left",
			Title:   "Panel 1",
			Width:   18,
			Height:  20,
		})

		centerBox := LayoutBoxedContent(LayoutBoxedContentInput{
			Content: "Center",
			Title:   "Panel 2",
			Width:   38,
			Height:  20,
		})

		rightBox := LayoutBoxedContent(LayoutBoxedContentInput{
			Content: "Right",
			Title:   "Panel 3",
			Width:   18,
			Height:  20,
		})

		layout := LayoutThreeColumn(LayoutThreeColumnInput{
			Left:   leftBox,
			Center: centerBox,
			Right:  rightBox,
			Widths: [3]int{20, 40, 20},
			Height: 24,
		})

		assert.NotEmpty(t, layout)
		assert.Contains(t, layout, "Panel 1")
		assert.Contains(t, layout, "Panel 2")
		assert.Contains(t, layout, "Panel 3")
	})
}
