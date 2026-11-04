package main

import (
	"fmt"
	"strings"
)

// countOf is how many words and characters text has, as the count under
// the note says it.
func countOf(text string) string {
	return fmt.Sprintf("%d words · %d characters", len(strings.Fields(text)), len([]rune(text)))
}
