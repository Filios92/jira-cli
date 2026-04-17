package jira

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSubstituteTemplateVars(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		vars     map[string]string
		expected string
	}{
		{"basic substitution", "SOS - [TITLE]", map[string]string{"TITLE": "My Task"}, "SOS - My Task"},
		{"multiple vars", "[A] and [B]", map[string]string{"A": "X", "B": "Y"}, "X and Y"},
		{"nil vars", "SOS - [TITLE]", nil, "SOS - [TITLE]"},
		{"empty vars", "SOS - [TITLE]", map[string]string{}, "SOS - [TITLE]"},
		{"no match", "plain text", map[string]string{"X": "Y"}, "plain text"},
		{"repeated var", "[X] [X]", map[string]string{"X": "Z"}, "Z Z"},
		{"empty value", "SOS - [TITLE]", map[string]string{"TITLE": ""}, "SOS - "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, SubstituteTemplateVars(tt.text, tt.vars))
		})
	}
}
