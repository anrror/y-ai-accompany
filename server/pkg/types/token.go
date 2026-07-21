// Package types defines shared utility functions for token estimation.
package types

import "math"

// EstimateTokens provides a rough token count for a text string using
// character-class heuristics:
//
//	CJK characters (U+4E00–U+9FFF): ~1 token per 1.5 characters
//	Other characters (ASCII, Latin, punctuation): ~1 token per 4 characters
//
// This is a coarse approximation suitable for budget planning. For production
// accuracy, substitute a model-specific tokenizer.
func EstimateTokens(text string) int {
	runes := []rune(text)
	var cjkCount int
	var otherCount int

	for _, r := range runes {
		if r >= 0x4E00 && r <= 0x9FFF {
			cjkCount++
		} else {
			otherCount++
		}
	}

	cjkTokens := float64(cjkCount) / 1.5
	otherTokens := float64(otherCount) / 4.0

	total := cjkTokens + otherTokens
	if total <= 0 {
		return 1
	}
	return int(math.Ceil(total))
}

// EstimateMessagesTokens returns the estimated token count for a slice of
// messages, including ~4 tokens per message for role/metadata overhead.
func EstimateMessagesTokens(msgs []string) int {
	total := 0
	for _, m := range msgs {
		total += EstimateTokens(m)
		total += 4 // role/metadata overhead per message
	}
	return total
}
