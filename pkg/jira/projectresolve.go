package jira

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	projectFromJQLQuoted = regexp.MustCompile(`(?i)\bproject\s*(?:=|in)\s*\(?\s*['"]([^'"]+)['"]`)
	projectFromJQLKey    = regexp.MustCompile(`(?i)\bproject\s*(?:=|in)\s*\(?\s*([A-Z][A-Z0-9_]+)\b`)
)

// ParseProjectFromJQL extracts a project key or name from a simple JQL project clause.
func ParseProjectFromJQL(jql string) string {
	if match := projectFromJQLQuoted.FindStringSubmatch(jql); len(match) > 1 {
		return strings.TrimSpace(match[1])
	}
	if match := projectFromJQLKey.FindStringSubmatch(jql); len(match) > 1 {
		return strings.ToUpper(match[1])
	}
	return ""
}

// ResolveProjectKey resolves a project key or display name to a project key.
func (c *Client) ResolveProjectKey(identifier string) (string, error) {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return "", nil
	}

	projects, err := c.Project()
	if err != nil {
		return "", err
	}

	byKey := make(map[string]string, len(projects))
	byName := make(map[string]string, len(projects))
	for _, project := range projects {
		byKey[project.Key] = project.Key
		byName[strings.ToLower(project.Name)] = project.Key
	}

	if key, ok := byKey[identifier]; ok {
		return key, nil
	}
	if key, ok := byName[strings.ToLower(identifier)]; ok {
		return key, nil
	}

	return "", fmt.Errorf("could not resolve project %q", identifier)
}
