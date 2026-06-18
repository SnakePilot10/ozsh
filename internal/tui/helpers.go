package tui

// helpers.go provides utility functions for the TUI.

// max returns the larger of two ints.
func max(a, b int) int {
    if a > b {
        return a
    }
    return b
}

// min returns the smaller of two ints.
func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}
