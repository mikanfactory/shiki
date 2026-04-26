package tui

import (
	"github.com/mikanfactory/yakumo/internal/model"
)

// NextSelectable returns the next selectable index after current, or current if none.
func NextSelectable(items []model.NavigableItem, current int) int {
	for i := current + 1; i < len(items); i++ {
		if items[i].Selectable {
			return i
		}
	}
	return current
}

// PrevSelectable returns the previous selectable index before current, or current if none.
func PrevSelectable(items []model.NavigableItem, current int) int {
	for i := current - 1; i >= 0; i-- {
		if items[i].Selectable {
			return i
		}
	}
	return current
}

// FirstSelectable returns the index of the first selectable item, or 0.
func FirstSelectable(items []model.NavigableItem) int {
	for i, item := range items {
		if item.Selectable {
			return i
		}
	}
	return 0
}

// recomputeScroll updates m.scrollOff based on current cursor, items, and
// height. Call after any change that moves the cursor or changes the viewport.
func recomputeScroll(m Model) Model {
	if len(m.items) == 0 {
		m.scrollOff = 0
		return m
	}
	heights := itemHeights(m.items, m.cursor, m.sidebarWidth)
	vp := viewportHeight(m.height)
	m.scrollOff = adjustScroll(m.cursor, vp, heights)
	return m
}

// adjustScroll returns the smallest scroll offset that keeps the cursor item
// inside the viewport, using per-item rendered heights rather than a fixed
// 1-row-per-item assumption. Returns the scroll offset such that
// heights[scrollOff..cursor] sums to at most viewportHeight, maximizing the
// number of items visible above the cursor.
//
// Per-item heights are required because action rows (PaddingTop(1)) span 2
// rows; treating them as 1 row each would underflow the viewport and push
// the topmost rows off-screen.
func adjustScroll(cursor, viewportHeight int, heights []int) int {
	if len(heights) == 0 || viewportHeight <= 0 {
		return 0
	}
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= len(heights) {
		cursor = len(heights) - 1
	}
	scrollOff := cursor
	used := heights[cursor]
	for scrollOff > 0 && used+heights[scrollOff-1] <= viewportHeight {
		scrollOff--
		used += heights[scrollOff]
	}
	return scrollOff
}
