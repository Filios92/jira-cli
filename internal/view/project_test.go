package view

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

func TestProjectRender(t *testing.T) {
	var b bytes.Buffer

	//nolint:unused
	type lead struct {
		Name string `json:"displayName"`
	}

	data := []*jira.Project{
		{Key: "FRST", Name: "First", Lead: lead{Name: "Person A"}, Type: jira.ProjectTypeClassic},
		{Key: "SCND", Name: "[2] Second", Lead: lead{Name: "Person B"}, Type: jira.ProjectTypeNextGen},
		{Key: "THIRD", Name: "Third", Lead: lead{Name: "Person C"}, Type: jira.ProjectTypeClassic},
	}
	project := NewProject(data, WithProjectWriter(&b))
	assert.NoError(t, project.Render())

	expected := `KEY	NAME	TYPE	LEAD
FRST	First	classic	Person A
SCND	[2] Second	next-gen	Person B
THIRD	Third	classic	Person C
`
	assert.Equal(t, expected, b.String())
}

func TestProjectRenderPlain(t *testing.T) {
	var b bytes.Buffer

	//nolint:unused
	type lead struct {
		Name string `json:"displayName"`
	}

	data := []*jira.Project{
		{Key: "FRST", Name: "First", Lead: lead{Name: "Person A"}, Type: jira.ProjectTypeClassic},
		{Key: "SCND", Name: "[2] Second", Lead: lead{Name: "Person B"}, Type: jira.ProjectTypeNextGen},
	}
	project := NewProject(
		data,
		WithProjectWriter(&b),
		WithProjectDisplay(DisplayFormat{Plain: true}),
	)
	assert.NoError(t, project.Render())

	expected := "KEY\tNAME\tTYPE\tLEAD\nFRST\tFirst\tclassic\tPerson A\nSCND\t[2] Second\tnext-gen\tPerson B\n"
	assert.Equal(t, expected, b.String())
}

func TestProjectRenderPlainNoHeadersWithDelimiter(t *testing.T) {
	var b bytes.Buffer

	//nolint:unused
	type lead struct {
		Name string `json:"displayName"`
	}

	data := []*jira.Project{
		{Key: "FRST", Name: "First", Lead: lead{Name: "Person A"}, Type: jira.ProjectTypeClassic},
	}
	project := NewProject(
		data,
		WithProjectWriter(&b),
		WithProjectDisplay(DisplayFormat{Plain: true, NoHeaders: true, Delimiter: "|"}),
	)
	assert.NoError(t, project.Render())

	expected := "FRST|First|classic|Person A\n"
	assert.Equal(t, expected, b.String())
}

func TestProjectRenderNoHeadersIgnoredWithoutPlain(t *testing.T) {
	var b bytes.Buffer

	//nolint:unused
	type lead struct {
		Name string `json:"displayName"`
	}

	data := []*jira.Project{
		{Key: "FRST", Name: "First", Lead: lead{Name: "Person A"}, Type: jira.ProjectTypeClassic},
	}
	project := NewProject(
		data,
		WithProjectWriter(&b),
		WithProjectDisplay(DisplayFormat{NoHeaders: true}),
	)
	assert.NoError(t, project.Render())

	expected := "KEY\tNAME\tTYPE\tLEAD\nFRST\tFirst\tclassic\tPerson A\n"
	assert.Equal(t, expected, b.String())
}

func TestProjectRenderPlainWithDelimiterKeepsHeaders(t *testing.T) {
	var b bytes.Buffer

	//nolint:unused
	type lead struct {
		Name string `json:"displayName"`
	}

	data := []*jira.Project{
		{Key: "FRST", Name: "First", Lead: lead{Name: "Person A"}, Type: jira.ProjectTypeClassic},
	}
	project := NewProject(
		data,
		WithProjectWriter(&b),
		WithProjectDisplay(DisplayFormat{Plain: true, Delimiter: "|"}),
	)
	assert.NoError(t, project.Render())

	expected := "KEY|NAME|TYPE|LEAD\nFRST|First|classic|Person A\n"
	assert.Equal(t, expected, b.String())
}

func TestProjectRenderPlainUsesStdoutWithoutCustomWriter(t *testing.T) {
	t.Cleanup(func() {
		os.Stdout = os.NewFile(uintptr(1), "/dev/stdout")
	})

	reader, writer, err := os.Pipe()
	assert.NoError(t, err)

	originalStdout := os.Stdout
	os.Stdout = writer

	//nolint:unused
	type lead struct {
		Name string `json:"displayName"`
	}

	data := []*jira.Project{
		{Key: "SOS", Name: "Service Operations", Lead: lead{Name: "Marks, Philipp"}, Type: jira.ProjectTypeClassic},
	}
	project := NewProject(data, WithProjectDisplay(DisplayFormat{Plain: true}))
	assert.NoError(t, project.Render())
	assert.NoError(t, writer.Close())
	os.Stdout = originalStdout

	var b bytes.Buffer
	_, err = b.ReadFrom(reader)
	assert.NoError(t, err)
	assert.NoError(t, reader.Close())

	out := b.String()
	assert.Contains(t, out, "KEY\tNAME")
	assert.Contains(t, out, "TYPE\tLEAD")
	assert.Contains(t, out, "SOS\tService Operations\tclassic\tMarks, Philipp\n")
}
