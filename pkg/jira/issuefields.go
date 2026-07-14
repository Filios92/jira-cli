package jira

import (
	"encoding/json"
	"strings"
)

// FieldIdentifier returns a normalized field name identifier.
func FieldIdentifier(name string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(name)), " ", "-")
}

// UnmarshalJSON implements custom unmarshaler to capture custom field values.
func (f *IssueFields) UnmarshalJSON(data []byte) error {
	type alias IssueFields

	if err := json.Unmarshal(data, (*alias)(f)); err != nil {
		return err
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	f.CustomFields = nil
	for k, v := range raw {
		if strings.HasPrefix(k, "customfield_") {
			if f.CustomFields == nil {
				f.CustomFields = make(map[string]json.RawMessage)
			}
			f.CustomFields[k] = v
		}
	}

	return nil
}

// MarshalJSON implements custom marshaler to include custom field values.
func (f IssueFields) MarshalJSON() ([]byte, error) {
	type alias IssueFields

	out := make(map[string]any)
	base, err := json.Marshal(alias(f))
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(base, &out); err != nil {
		return nil, err
	}

	for k, v := range f.CustomFields {
		var val any
		if err := json.Unmarshal(v, &val); err != nil {
			out[k] = string(v)
		} else {
			out[k] = val
		}
	}

	return json.Marshal(out)
}
