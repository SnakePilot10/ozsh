package main

// min returns the smallest of three ints.
func min(a, b, c int) int {
    m := a
    if b < m {
        m = b
    }
    if c < m {
        m = c
    }
    return m
}
