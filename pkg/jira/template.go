package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const (
	TemplateFieldSummary     = 1
	TemplateFieldDescription = 2
	TemplateFieldLabels      = 4
)

type TemplateField struct {
	FieldType             int    `json:"fieldType"`
	Text1                 string `json:"text1"`
	JiraField             bool   `json:"jiraField"`
	ReadOnly              bool   `json:"readOnly"`
	Overwritable          bool   `json:"overwritable"`
	AtlassianWikiRenderer bool   `json:"atlassianWikiRenderer"`
}

type TemplateVariable struct {
	ID               int      `json:"id"`
	Key              string   `json:"key"`
	FieldType        string   `json:"fieldType"`
	Required         bool     `json:"required"`
	Options          []string `json:"options"`
	Order            int      `json:"order"`
	IsUsedInTemplate bool     `json:"isUsedInTemplate"`
}

type TemplateResponse struct {
	Fields              []TemplateField    `json:"fields"`
	TemplateID          string             `json:"templateID"`
	TemplateName        string             `json:"templateName"`
	PatternFields       []string           `json:"patternFields"`
	UserVariables       []TemplateVariable `json:"userVariables"`
	TemplateDescription string             `json:"templateDescription"`
}

func (c *Client) GetIssueTemplate(templateID, projectID, issueTypeID string) (*TemplateResponse, error) {
	path := fmt.Sprintf(
		"/rest/it/1.0/template/autocomplete?templateId=%s&projectId=%s&issueTypeId=%s",
		templateID,
		projectID,
		issueTypeID,
	)

	res, err := c.GetRaw(context.Background(), path, nil)
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

	var out TemplateResponse

	err = json.NewDecoder(res.Body).Decode(&out)

	return &out, err
}

func (c *Client) GetProjectV2(key string) (*Project, error) {
	res, err := c.GetV2(context.Background(), "/project/"+key, nil)
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

	var out Project

	err = json.NewDecoder(res.Body).Decode(&out)

	return &out, err
}

func SubstituteTemplateVars(text string, vars map[string]string) string {
	if len(vars) == 0 || text == "" {
		return text
	}

	for key, val := range vars {
		text = strings.ReplaceAll(text, "["+key+"]", val)
	}

	return text
}
