package jira

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIssueFieldsUnmarshalCustomFields(t *testing.T) {
	raw := `{
		"summary": "Test issue",
		"customfield_10111": 5,
		"customfield_10001": {"value": "Option A"}
	}`

	var fields IssueFields
	assert.NoError(t, json.Unmarshal([]byte(raw), &fields))
	assert.Equal(t, "Test issue", fields.Summary)
	assert.Len(t, fields.CustomFields, 2)
	assert.Equal(t, "5", string(fields.CustomFields["customfield_10111"]))
	assert.Equal(t, `{"value": "Option A"}`, string(fields.CustomFields["customfield_10001"]))
}

func TestIssueFieldsMarshalCustomFields(t *testing.T) {
	fields := IssueFields{
		Summary: "Test issue",
		CustomFields: map[string]json.RawMessage{
			"customfield_10111": json.RawMessage(`5`),
		},
	}

	out, err := json.Marshal(fields)
	assert.NoError(t, err)

	var decoded map[string]any
	assert.NoError(t, json.Unmarshal(out, &decoded))
	assert.Equal(t, "Test issue", decoded["summary"])
	assert.Equal(t, float64(5), decoded["customfield_10111"])
}

func TestFieldIdentifier(t *testing.T) {
	assert.Equal(t, "story-points", FieldIdentifier("Story Points"))
	assert.Equal(t, "original-story-points", FieldIdentifier(" Original story points "))
}

func TestFormatFieldValue(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "null", raw: "null", want: ""},
		{name: "number", raw: "3", want: "3"},
		{name: "string", raw: `"hello"`, want: "hello"},
		{name: "option", raw: `{"value":"High"}`, want: "High"},
		{name: "user", raw: `{"displayName":"Jane Doe"}`, want: "Jane Doe"},
		{name: "array", raw: `[{"value":"A"},{"value":"B"}]`, want: "A, B"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, FormatFieldValue(json.RawMessage(tc.raw)))
		})
	}
}
