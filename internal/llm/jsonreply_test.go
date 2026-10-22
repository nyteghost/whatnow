package llm

import "testing"

func TestJSONReply(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"bare", `{"a":1}`, `{"a":1}`},
		{"whitespace", "\n  {\"a\":1}\n", `{"a":1}`},
		{"array", `[1,2]`, `[1,2]`},
		{"json fence", "```json\n{\"a\":1}\n```", `{"a":1}`},
		{"plain fence", "```\n{\"a\":1}\n```\n", `{"a":1}`},
		{"prose around", "Here is the plan:\n{\"a\":{\"b\":2}}\nHope that helps!", `{"a":{"b":2}}`},
		{"think block", "<think>\nThe user wants {x}.\n</think>\n{\"a\":1}", `{"a":1}`},
		{"think then fence", "<think>hmm</think>\n```json\n{\"a\":1}\n```", `{"a":1}`},
		{"not json", "sorry, I can't", "sorry, I can't"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := JSONReply(tt.in); got != tt.want {
				t.Errorf("JSONReply(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
