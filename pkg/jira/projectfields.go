package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
)

const createMetaPageSize = 50

type createMetaPage struct {
	StartAt    int  `json:"startAt"`
	MaxResults int  `json:"maxResults"`
	Total      int  `json:"total"`
	IsLast     bool `json:"isLast"`
	Values     []struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Subtask bool   `json:"subtask"`
		FieldID string `json:"fieldId"`
		Key     string `json:"key"`
	} `json:"values"`
	Fields []struct {
		FieldID string `json:"fieldId"`
		Name    string `json:"name"`
		Key     string `json:"key"`
	} `json:"fields"`
}

func (p createMetaPage) items() []struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Subtask bool   `json:"subtask"`
	FieldID string `json:"fieldId"`
	Key     string `json:"key"`
} {
	if len(p.Fields) > 0 {
		out := make([]struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Subtask bool   `json:"subtask"`
			FieldID string `json:"fieldId"`
			Key     string `json:"key"`
		}, len(p.Fields))
		for i, f := range p.Fields {
			out[i].FieldID = f.FieldID
			out[i].Name = f.Name
			out[i].Key = f.Key
		}
		return out
	}
	return p.Values
}

func (p createMetaPage) done(fetched int) bool {
	if p.IsLast {
		return true
	}
	if p.Total > 0 && p.StartAt+fetched >= p.Total {
		return true
	}
	return fetched == 0
}

func (c *Client) getCreateMetaPage(path string) (*createMetaPage, error) {
	res, err := c.GetV2(context.Background(), path, nil)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, ErrEmptyResponse
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return nil, formatUnexpectedResponse(res)
	}

	var out createMetaPage
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, err
	}

	return &out, nil
}

func parseCustomFieldsFromPageItems(items []struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Subtask bool   `json:"subtask"`
	FieldID string `json:"fieldId"`
	Key     string `json:"key"`
}) []IssueTypeField {
	fields := make([]IssueTypeField, 0)
	for _, item := range items {
		key := item.FieldID
		if key == "" {
			key = item.Key
		}
		if key == "" || !strings.HasPrefix(key, "customfield_") {
			continue
		}
		fields = append(fields, IssueTypeField{
			Name: item.Name,
			Key:  key,
		})
	}
	return fields
}

// GetCreateMetaIssueTypeFields gets create-screen fields for a project issue type.
func (c *Client) GetCreateMetaIssueTypeFields(projectKey, issueTypeID string) ([]IssueTypeField, error) {
	startAt := 0
	fields := make([]IssueTypeField, 0)

	for {
		path := fmt.Sprintf(
			"/issue/createmeta/%s/issuetypes/%s?startAt=%d&maxResults=%d",
			projectKey, issueTypeID, startAt, createMetaPageSize,
		)

		page, err := c.getCreateMetaPage(path)
		if err != nil {
			return nil, err
		}

		items := page.items()
		fields = append(fields, parseCustomFieldsFromPageItems(items)...)

		if page.done(len(items)) {
			break
		}
		startAt += len(items)
	}

	return fields, nil
}

func (c *Client) getProjectIssueTypes(projectKey string) ([]struct {
	ID      string
	Name    string
	Subtask bool
}, error) {
	startAt := 0
	issueTypes := make([]struct {
		ID      string
		Name    string
		Subtask bool
	}, 0)

	for {
		path := fmt.Sprintf(
			"/issue/createmeta/%s/issuetypes?startAt=%d&maxResults=%d",
			projectKey, startAt, createMetaPageSize,
		)

		page, err := c.getCreateMetaPage(path)
		if err != nil {
			return nil, err
		}

		items := page.items()
		for _, it := range items {
			issueTypes = append(issueTypes, struct {
				ID      string
				Name    string
				Subtask bool
			}{
				ID:      it.ID,
				Name:    it.Name,
				Subtask: it.Subtask,
			})
		}

		if page.done(len(items)) {
			break
		}
		startAt += len(items)
	}

	return issueTypes, nil
}

func mergeProjectCustomFields(dst map[string]IssueTypeField, fields map[string]IssueTypeField) {
	for key, field := range fields {
		if !strings.HasPrefix(key, "customfield_") {
			continue
		}
		if field.Key == "" {
			field.Key = key
		}
		dst[field.Key] = field
	}
}

func projectCustomFieldsFromCreateMeta(meta *CreateMetaResponse) []IssueTypeField {
	fieldsMap := make(map[string]IssueTypeField)
	if len(meta.Projects) == 0 {
		return nil
	}

	for _, it := range meta.Projects[0].IssueTypes {
		mergeProjectCustomFields(fieldsMap, it.Fields)
	}

	return sortedProjectCustomFields(fieldsMap)
}

func sortedProjectCustomFields(fieldsMap map[string]IssueTypeField) []IssueTypeField {
	out := make([]IssueTypeField, 0, len(fieldsMap))
	for _, field := range fieldsMap {
		out = append(out, field)
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}

func shouldUsePaginatedCreateMeta(installation string, versionMajor, versionMinor int) bool {
	if installation != InstallationTypeLocal {
		return false
	}
	if versionMajor == 0 {
		return true
	}
	return versionMajor > 8 || (versionMajor == 8 && versionMinor >= 4)
}

func (c *Client) projectCustomFieldsPaginated(projectKey string) ([]IssueTypeField, error) {
	issueTypes, err := c.getProjectIssueTypes(projectKey)
	if err != nil {
		return nil, err
	}

	fieldsMap := make(map[string]IssueTypeField)
	for _, it := range issueTypes {
		fields, err := c.GetCreateMetaIssueTypeFields(projectKey, it.ID)
		if err != nil {
			return nil, err
		}
		for _, field := range fields {
			fieldsMap[field.Key] = field
		}
	}

	return sortedProjectCustomFields(fieldsMap), nil
}

// ProjectCustomFields returns custom fields configured on the project's create screens.
func (c *Client) ProjectCustomFields(projectKey, installation string, versionMajor, versionMinor int) ([]IssueTypeField, error) {
	if shouldUsePaginatedCreateMeta(installation, versionMajor, versionMinor) {
		return c.projectCustomFieldsPaginated(projectKey)
	}

	meta, err := c.GetCreateMeta(&CreateMetaRequest{
		Projects: projectKey,
		Expand:   "projects.issuetypes.fields",
	})
	if err != nil {
		return nil, err
	}

	return projectCustomFieldsFromCreateMeta(meta), nil
}

// FilterCustomFields keeps only custom fields that belong to the project.
func FilterCustomFields(fields map[string]json.RawMessage, allowed []IssueTypeField) map[string]json.RawMessage {
	if len(fields) == 0 || len(allowed) == 0 {
		return nil
	}

	allowedKeys := make(map[string]struct{}, len(allowed))
	for _, field := range allowed {
		allowedKeys[field.Key] = struct{}{}
	}

	out := make(map[string]json.RawMessage)
	for key, val := range fields {
		if _, ok := allowedKeys[key]; ok {
			out[key] = val
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
