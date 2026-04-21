package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"
)

var (
	moveProjectBrowsePattern = regexp.MustCompile(`/browse/([A-Z]+-\d+)`)
	moveProjectTitlePattern  = regexp.MustCompile(`\[([A-Z]+-\d+)\]`)
)

// MoveProjectParams holds parameters for moving an issue between projects.
type MoveProjectParams struct {
	IssueKey      string
	TargetProject string
	IssueType     string // empty = keep same
	DryRun        bool
}

// MoveProjectResult holds the result of a project move.
type MoveProjectResult struct {
	OldKey     string
	NewKey     string
	OldProject string
	NewProject string
	IssueType  string
}

type issueInfo struct {
	ID     string `json:"id"`
	Key    string `json:"key"`
	Fields struct {
		Project struct {
			ID  string `json:"id"`
			Key string `json:"key"`
		} `json:"project"`
		IssueType struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"issuetype"`
	} `json:"fields"`
}

type projectInfo struct {
	ID         string `json:"id"`
	Key        string `json:"key"`
	IssueTypes []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"issueTypes"`
}

func fetchIssueInfo(client *Client, issueKey string) (*issueInfo, error) {
	issueResp, err := client.GetV2(context.Background(), "/issue/"+issueKey+"?fields=project,issuetype", nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = issueResp.Body.Close() }()

	issueBody, err := io.ReadAll(issueResp.Body)
	if err != nil {
		return nil, err
	}

	var info issueInfo
	if err := json.Unmarshal(issueBody, &info); err != nil {
		return nil, fmt.Errorf("parse issue response: %w", err)
	}

	if strings.TrimSpace(info.ID) == "" {
		return nil, fmt.Errorf("issue %q missing id", issueKey)
	}
	if strings.TrimSpace(info.Fields.Project.Key) == "" {
		return nil, fmt.Errorf("issue %q missing project key", issueKey)
	}
	if strings.TrimSpace(info.Fields.IssueType.Name) == "" {
		return nil, fmt.Errorf("issue %q missing issue type", issueKey)
	}

	return &info, nil
}

func fetchProjectInfo(client *Client, projectKey string) (*projectInfo, error) {
	targetResp, err := client.GetV2(context.Background(), "/project/"+projectKey, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = targetResp.Body.Close() }()

	targetBody, err := io.ReadAll(targetResp.Body)
	if err != nil {
		return nil, err
	}

	var info projectInfo
	if err := json.Unmarshal(targetBody, &info); err != nil {
		return nil, fmt.Errorf("parse target project response: %w", err)
	}

	if strings.TrimSpace(info.ID) == "" {
		return nil, fmt.Errorf("target project %q missing id", projectKey)
	}
	if strings.TrimSpace(info.Key) == "" {
		return nil, fmt.Errorf("target project %q missing key", projectKey)
	}

	return &info, nil
}

func resolveTargetIssueType(issue *issueInfo, project *projectInfo, requestedType string) (string, string, error) {
	issueTypeName := strings.TrimSpace(requestedType)
	if issueTypeName == "" {
		issueTypeName = issue.Fields.IssueType.Name
	}

	for _, targetIssueType := range project.IssueTypes {
		if targetIssueType.Name == issueTypeName {
			return issueTypeName, targetIssueType.ID, nil
		}
	}

	return "", "", fmt.Errorf("issue type %q is not available in project %q", issueTypeName, project.Key)
}

func extractMovedIssueKey(result *MoveProjectResult, originalKey, finalURL, body string) bool {
	match := moveProjectBrowsePattern.FindStringSubmatch(finalURL)
	if len(match) == 2 && match[1] != originalKey {
		result.NewKey = match[1]
		return true
	}

	for _, m := range moveProjectTitlePattern.FindAllStringSubmatch(body, -1) {
		if len(m) == 2 && m[1] != originalKey {
			result.NewKey = m[1]
			return true
		}
	}

	for _, m := range moveProjectBrowsePattern.FindAllStringSubmatch(body, -1) {
		if len(m) == 2 && m[1] != originalKey {
			result.NewKey = m[1]
			return true
		}
	}

	return false
}

// MoveProject moves an issue from one Jira project to another using the legacy JSP workflow.
func MoveProject(sc *SessionClient, client *Client, params MoveProjectParams) (*MoveProjectResult, error) {
	if sc == nil {
		return nil, fmt.Errorf("session client is required")
	}
	if client == nil {
		return nil, fmt.Errorf("client is required")
	}
	if strings.TrimSpace(params.IssueKey) == "" {
		return nil, fmt.Errorf("issue key is required")
	}
	if strings.TrimSpace(params.TargetProject) == "" {
		return nil, fmt.Errorf("target project is required")
	}

	issueInfo, err := fetchIssueInfo(client, params.IssueKey)
	if err != nil {
		return nil, err
	}

	targetInfo, err := fetchProjectInfo(client, params.TargetProject)
	if err != nil {
		return nil, err
	}

	if strings.EqualFold(issueInfo.Fields.Project.Key, targetInfo.Key) {
		return nil, fmt.Errorf("issue %q is already in project %q", params.IssueKey, targetInfo.Key)
	}

	issueTypeName, issueTypeID, err := resolveTargetIssueType(issueInfo, targetInfo, params.IssueType)
	if err != nil {
		return nil, err
	}

	result := &MoveProjectResult{
		OldKey:     issueInfo.Key,
		OldProject: issueInfo.Fields.Project.Key,
		NewProject: targetInfo.Key,
		IssueType:  issueTypeName,
	}

	if params.DryRun {
		return result, nil
	}

	if _, err := sc.Get("/secure/MoveIssue!default.jspa?id=" + issueInfo.ID); err != nil {
		return nil, err
	}

	stepTwo := url.Values{}
	stepTwo.Set("id", issueInfo.ID)
	stepTwo.Set("pid", targetInfo.ID)
	stepTwo.Set("issuetype", issueTypeID)
	if _, _, err := sc.PostForm("/secure/MoveIssue.jspa", stepTwo); err != nil {
		return nil, err
	}

	stepThree := url.Values{}
	stepThree.Set("id", issueInfo.ID)
	if _, _, err := sc.PostForm("/secure/MoveIssueUpdateFields.jspa", stepThree); err != nil {
		return nil, err
	}

	stepFour := url.Values{}
	stepFour.Set("id", issueInfo.ID)
	stepFour.Set("confirm", "true")
	stepFour.Set("Move", "Move")
	body, finalURL, err := sc.PostForm("/secure/MoveIssueConfirm.jspa", stepFour)
	if err != nil {
		return nil, err
	}

	if extractMovedIssueKey(result, issueInfo.Key, finalURL, body) {
		return result, nil
	}

	return nil, fmt.Errorf("could not extract new issue key from response")
}
