package view

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
	"github.com/ankitpokhrel/jira-cli/pkg/tui"
)

// ProjectOption is a functional option to wrap project properties.
type ProjectOption func(*Project)

// Project is a project view.
type Project struct {
	data            []*jira.Project
	writer          io.Writer
	buf             *bytes.Buffer
	hasCustomWriter bool
	Display         DisplayFormat
}

// NewProject initializes a project.
func NewProject(data []*jira.Project, opts ...ProjectOption) *Project {
	p := Project{
		data: data,
		buf:  new(bytes.Buffer),
	}
	p.writer = tabwriter.NewWriter(p.buf, 0, tabWidth, 1, '\t', 0)

	for _, opt := range opts {
		opt(&p)
	}
	return &p
}

// WithProjectWriter sets a writer for the project.
func WithProjectWriter(w io.Writer) ProjectOption {
	return func(p *Project) {
		p.writer = w
		p.hasCustomWriter = true
	}
}

func WithProjectDisplay(display DisplayFormat) ProjectOption {
	return func(p *Project) {
		p.Display = display
	}
}

// Render renders the project view.
func (p Project) Render() error {
	if p.Display.Plain || tui.IsDumbTerminal() || tui.IsNotTTY() {
		delimiter := "\t"
		if p.Display.Plain && p.Display.Delimiter != "" {
			delimiter = p.Display.Delimiter
		}
		if p.hasCustomWriter {
			return p.renderPlain(p.writer, delimiter)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, tabWidth, 1, '\t', 0)
		return p.renderPlain(w, delimiter)
	}

	if p.hasCustomWriter {
		w := tabwriter.NewWriter(p.writer, 0, tabWidth, 1, '\t', 0)
		if err := p.renderTable(w); err != nil {
			return err
		}
		return w.Flush()
	}

	if err := p.renderTable(p.writer); err != nil {
		return err
	}
	if w, ok := p.writer.(*tabwriter.Writer); ok {
		err := w.Flush()
		if err != nil {
			return err
		}
	}

	return tui.PagerOut(p.buf.String())
}

func (p Project) renderPlain(w io.Writer, delimiter string) error {
	return renderPlain(w, p.tableData(), delimiter)
}

func (p Project) renderTable(w io.Writer) error {
	for _, items := range p.tableData() {
		_, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", items[0], items[1], items[2], items[3])
		if err != nil {
			return err
		}
	}

	return nil
}

func (p Project) header() []string {
	return []string{
		"KEY",
		"NAME",
		"TYPE",
		"LEAD",
	}
}

func (p Project) tableData() tui.TableData {
	data := tui.TableData{}
	if !p.Display.Plain || !p.Display.NoHeaders {
		data = append(data, p.header())
	}

	for _, d := range p.data {
		data = append(data, []string{d.Key, prepareTitle(d.Name), d.Type, d.Lead.Name})
	}

	return data
}
