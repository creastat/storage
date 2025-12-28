package storage

import "time"

// TruncateHistory truncates the conversation history based on token and message limits.
// It applies message limit first, then token limit, removing oldest messages as needed.
// Returns the truncated history with the most recent messages preserved.
func TruncateHistory(history []Message, tokenLimit, messageLimit int) []Message {
	if len(history) == 0 {
		return history
	}

	// First, apply message limit
	if len(history) > messageLimit {
		history = history[len(history)-messageLimit:]
	}

	// Then, apply token limit
	totalTokens := 0
	for _, msg := range history {
		totalTokens += msg.TokenCount
	}

	// Remove oldest messages until within token limit
	for totalTokens > tokenLimit && len(history) > 0 {
		totalTokens -= history[0].TokenCount
		history = history[1:]
	}

	return history
}

// AddMessageToHistory appends a message to the conversation history with an estimated token count.
// It calculates the token count using EstimateTokens and returns the updated history.
func AddMessageToHistory(history []Message, role, content string) []Message {
	tokenCount := EstimateTokens(content)
	message := Message{
		Role:       role,
		Content:    content,
		TokenCount: tokenCount,
		Timestamp:  time.Now(),
	}
	return append(history, message)
}
