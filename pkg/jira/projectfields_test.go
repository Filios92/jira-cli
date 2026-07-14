package jira

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestProjectCustomFieldsFromCreateMeta(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/2/issue/createmeta", r.URL.Path)

		resp, err := os.ReadFile("./testdata/createmeta.json")
		assert.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(resp)
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	actual, err := client.ProjectCustomFields("TEST", InstallationTypeCloud, 8, 0)
	assert.NoError(t, err)

	expected := []IssueTypeField{
		{Name: "Epic Link", Key: "customfield_10014"},
		{Name: "Epic Name", Key: "customfield_10011"},
	}
	assert.Equal(t, expected, actual)
}

func TestProjectCustomFieldsPaginated(t *testing.T) {
	var fieldCalls int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/api/2/issue/createmeta/ACME/issuetypes":
			assert.Equal(t, "0", r.URL.Query().Get("startAt"))
			resp := `{
				"startAt": 0,
				"maxResults": 50,
				"total": 1,
				"isLast": true,
				"values": [{"id":"10001","name":"Task","subtask":false}]
			}`
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(resp))
		case "/rest/api/2/issue/createmeta/ACME/issuetypes/10001":
			fieldCalls++
			startAt := r.URL.Query().Get("startAt")
			if startAt == "0" {
				resp := `{
					"startAt": 0,
					"maxResults": 50,
					"total": 3,
					"isLast": false,
					"values": [
						{"fieldId":"customfield_10001","name":"Text 1"},
						{"fieldId":"customfield_10002","name":"Text 2"}
					]
				}`
				_, _ = w.Write([]byte(resp))
				return
			}
			resp := `{
				"startAt": 2,
				"maxResults": 50,
				"total": 3,
				"isLast": true,
				"values": [
					{"fieldId":"customfield_10003","name":"Text 3"}
				]
			}`
			_, _ = w.Write([]byte(resp))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	actual, err := client.ProjectCustomFields("ACME", InstallationTypeLocal, 9, 0)
	assert.NoError(t, err)
	assert.Equal(t, 2, fieldCalls)

	expected := []IssueTypeField{
		{Name: "Text 1", Key: "customfield_10001"},
		{Name: "Text 2", Key: "customfield_10002"},
		{Name: "Text 3", Key: "customfield_10003"},
	}
	assert.Equal(t, expected, actual)
}

func TestFilterCustomFields(t *testing.T) {
	raw := map[string]json.RawMessage{
		"customfield_10011": json.RawMessage(`"Epic"`),
		"customfield_99999": json.RawMessage(`"ignored"`),
	}
	allowed := []IssueTypeField{{Name: "Epic Name", Key: "customfield_10011"}}

	filtered := FilterCustomFields(raw, allowed)
	assert.Len(t, filtered, 1)
	assert.Equal(t, json.RawMessage(`"Epic"`), filtered["customfield_10011"])
}

func TestCreateMetaPageDone(t *testing.T) {
	page := createMetaPage{IsLast: true}
	assert.True(t, page.done(10))

	page = createMetaPage{Total: 5, StartAt: 0}
	assert.True(t, page.done(5))

	page = createMetaPage{Total: 5, StartAt: 0}
	assert.False(t, page.done(1))
}
