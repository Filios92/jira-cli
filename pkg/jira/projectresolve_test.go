package jira

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseProjectFromJQL(t *testing.T) {
	tests := []struct {
		jql  string
		want string
	}{
		{jql: `project = 'Project Beta' AND status = Open`, want: "Project Beta"},
		{jql: `project = "Project Beta"`, want: "Project Beta"},
		{jql: `project = PROJB AND labels = L1`, want: "PROJB"},
		{jql: `project in (PROJB, PROJA)`, want: "PROJB"},
		{jql: `status = Open`, want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.jql, func(t *testing.T) {
			assert.Equal(t, tc.want, ParseProjectFromJQL(tc.jql))
		})
	}
}
