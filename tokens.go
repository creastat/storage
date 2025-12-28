package storage

// EstimateTokens estimates the token count for a given text using a Unicode-aware heuristic.
// ASCII characters (English, numbers, punctuation) are weighted at ~4 per token.
// Non-ASCII characters (CJK, Cyrillic, Arabic, Emoji, etc.) are weighted at ~1 per token.
func EstimateTokens(text string) int {
	weight := 0
	for _, r := range text {
		switch {
		case r <= 127: // ASCII (English, numbers, punctuation)
			weight += 1 // ~4 ASCII chars = 1 token
		default: // Non-ASCII (CJK, Cyrillic, Arabic, Emoji, etc.)
			weight += 4 // ~1 non-ASCII char = 1 token (conservative)
		}
	}
	// Result:
	// - English: 4 chars -> 1 token
	// - CJK/Cyrillic: 1 char -> 1 token
	// - Mixed: weighted average
	return (weight + 3) / 4
}
