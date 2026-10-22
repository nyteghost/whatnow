package llm

import (
	"regexp"
	"strings"
)

var thinkBlock = regexp.MustCompile(`(?s)<think>.*?</think>`)

// JSONReply digs the JSON out of a structured reply. Hosted models return it bare, but smaller
// local ones often wrap it in a ```json fence, put a sentence before or after it, or think aloud
// in a <think> block first. A reply that is already bare JSON comes back unchanged.
func JSONReply(reply string) string {
	s := strings.TrimSpace(thinkBlock.ReplaceAllString(reply, ""))
	if strings.HasPrefix(s, "```") {
		if nl := strings.IndexByte(s, '\n'); nl >= 0 {
			s = s[nl+1:]
		}
		if end := strings.LastIndex(s, "```"); end >= 0 {
			s = s[:end]
		}
		s = strings.TrimSpace(s)
	}
	if strings.HasPrefix(s, "{") || strings.HasPrefix(s, "[") {
		return s
	}
	// Prose around the object: keep from the first brace to the last.
	start, end := strings.IndexByte(s, '{'), strings.LastIndexByte(s, '}')
	if start >= 0 && end > start {
		return s[start : end+1]
	}
	return s
}
