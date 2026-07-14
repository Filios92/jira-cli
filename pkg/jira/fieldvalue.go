package jira

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// FormatFieldValue formats a raw custom field value for display.
func FormatFieldValue(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}

	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}

	return formatFieldValue(v)
}

func formatFieldValue(v any) string {
	switch val := v.(type) {
	case nil:
		return ""
	case string:
		return val
	case float64:
		if val == float64(int64(val)) {
			return strconv.FormatInt(int64(val), 10)
		}
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	case map[string]any:
		if s, ok := val["value"].(string); ok {
			return s
		}
		if s, ok := val["name"].(string); ok {
			return s
		}
		if s, ok := val["displayName"].(string); ok {
			return s
		}
		if s, ok := val["key"].(string); ok {
			return s
		}
		return ""
	case []any:
		parts := make([]string, 0, len(val))
		for _, item := range val {
			if s := formatFieldValue(item); s != "" {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, ", ")
	default:
		return fmt.Sprintf("%v", val)
	}
}
