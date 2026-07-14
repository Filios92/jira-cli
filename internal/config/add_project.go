package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

// AddProjectOptions configures registering an additional project in the config file.
type AddProjectOptions struct {
	Project string
	Board   string
	NoBoard bool
}

// AddProject registers a project (and optional board) in the existing config file.
func AddProject(opts AddProjectOptions) error {
	cfgFile := viper.ConfigFileUsed()
	if !Exists(cfgFile) {
		return fmt.Errorf("missing configuration file, run 'jira init' first")
	}

	board := opts.Board
	if opts.NoBoard {
		board = optionNone
	}

	gen := NewJiraCLIConfigGenerator(&JiraCLIConfig{
		Project: opts.Project,
		Board:   board,
	})
	gen.jiraClient = api.DefaultClient(viper.GetBool("debug"))

	if err := gen.configureProjectAndBoardDetails(); err != nil {
		return err
	}
	if gen.value.project == nil {
		return fmt.Errorf("project not selected")
	}

	displayName := resolveProjectDisplayName(gen.jiraClient, gen.value.project.Key)
	if err := SaveProjectEntry(gen.value.project, projectBoardConf(gen.value.board), displayName); err != nil {
		return err
	}

	return nil
}

func resolveProjectDisplayName(client *jira.Client, key string) string {
	projects, err := client.Project()
	if err != nil {
		return ""
	}
	for _, project := range projects {
		if strings.EqualFold(project.Key, key) {
			return project.Name
		}
	}
	return ""
}
