package github

import "testing"

func TestParseIssueReference(t *testing.T) {
	tests := []struct {
		name string
		body string
		want int
	}{
		{"closes keyword", "Closes #42", 42},
		{"fix keyword", "Fix #100", 100},
		{"fixes keyword", "This PR fixes #7 by updating the handler", 7},
		{"resolve keyword", "Resolve #999", 999},
		{"case insensitive", "CLOSES #55", 55},
		{"closed keyword", "closed #33", 33},
		{"resolved keyword", "resolved #88", 88},
		{"first match wins", "Fixes #10, also closes #20", 10},
		{"with newline before", "Some description\n\nCloses #5", 5},
		{"no match", "This is a regular PR body", 0},
		{"hash without keyword", "See #42 for context", 0},
		{"empty body", "", 0},
		{"keyword without number", "Fixes the bug", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseIssueReference(tt.body)
			if got != tt.want {
				t.Errorf("ParseIssueReference(%q) = %d, want %d", tt.body, got, tt.want)
			}
		})
	}
}
