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

var moveProjectBrowsePattern = regexp.MustCompile(`/browse/([A-Z]+-\d+)`)

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

	type issueResponse struct {
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

	type projectResponse struct {
		ID         string `json:"id"`
		Key        string `json:"key"`
		IssueTypes []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"issueTypes"`
	}

	issueResp, err := client.GetV2(context.Background(), "/issue/"+params.IssueKey+"?fields=project,issuetype", nil)
	if err != nil {
		return nil, err
	}
	defer issueResp.Body.Close()

	issueBody, err := io.ReadAll(issueResp.Body)
	if err != nil {
		return nil, err
	}

	var issueInfo issueResponse
	if err := json.Unmarshal(issueBody, &issueInfo); err != nil {
		return nil, fmt.Errorf("parse issue response: %w", err)
	}

	if strings.TrimSpace(issueInfo.ID) == "" {
		return nil, fmt.Errorf("issue %q missing id", params.IssueKey)
	}
	if strings.TrimSpace(issueInfo.Fields.Project.Key) == "" {
		return nil, fmt.Errorf("issue %q missing project key", params.IssueKey)
	}
	if strings.TrimSpace(issueInfo.Fields.IssueType.Name) == "" {
		return nil, fmt.Errorf("issue %q missing issue type", params.IssueKey)
	}

	targetResp, err := client.GetV2(context.Background(), "/project/"+params.TargetProject, nil)
	if err != nil {
		return nil, err
	}
	defer targetResp.Body.Close()

	targetBody, err := io.ReadAll(targetResp.Body)
	if err != nil {
		return nil, err
	}

	var targetInfo projectResponse
	if err := json.Unmarshal(targetBody, &targetInfo); err != nil {
		return nil, fmt.Errorf("parse target project response: %w", err)
	}

	if strings.TrimSpace(targetInfo.ID) == "" {
		return nil, fmt.Errorf("target project %q missing id", params.TargetProject)
	}
	if strings.TrimSpace(targetInfo.Key) == "" {
		return nil, fmt.Errorf("target project %q missing key", params.TargetProject)
	}

	if strings.EqualFold(issueInfo.Fields.Project.Key, targetInfo.Key) {
		return nil, fmt.Errorf("issue %q is already in project %q", params.IssueKey, targetInfo.Key)
	}

	issueTypeName := strings.TrimSpace(params.IssueType)
	if issueTypeName == "" {
		issueTypeName = issueInfo.Fields.IssueType.Name
	}

	issueTypeID := ""
	for _, targetIssueType := range targetInfo.IssueTypes {
		if targetIssueType.Name == issueTypeName {
			issueTypeID = targetIssueType.ID
			break
		}
	}
	if issueTypeID == "" {
		return nil, fmt.Errorf("issue type %q is not available in project %q", issueTypeName, targetInfo.Key)
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
	_, finalURL, err := sc.PostForm("/secure/MoveIssueConfirm.jspa", stepFour)
	if err != nil {
		return nil, err
	}

	match := moveProjectBrowsePattern.FindStringSubmatch(finalURL)
	if len(match) != 2 {
		return nil, fmt.Errorf("could not extract new issue key from redirect URL %q", finalURL)
	}

	result.NewKey = match[1]

	return result, nil
}
