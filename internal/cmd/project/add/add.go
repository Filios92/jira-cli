package add

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	jiraConfig "github.com/ankitpokhrel/jira-cli/internal/config"
)

const (
	helpText = `Add registers another project in the configuration file.

You can select a project and board interactively, or pass them as flags.
The entry is stored under 'projects' and can be used with -p / --project.

This command does not replace your existing configuration or default project.`

	examples = `$ jira project add

# Register a project without a board (for issue list/search only)
$ jira project add --project PROJB --no-board

# Register a project and board non-interactively
$ jira project add --project PROJB --board "Team Board"`
)

// NewCmdAdd is an add command.
func NewCmdAdd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "add",
		Short:   "Add registers another project in the config",
		Long:    helpText,
		Example: examples,
		Aliases: []string{"register"},
		Run:     Add,
	}

	cmd.Flags().SortFlags = false
	cmd.Flags().String("project", "", "Project key to register")
	cmd.Flags().String("board", "", "Board name for the project")
	cmd.Flags().Bool("no-board", false, "Register the project without a board")

	return cmd
}

// Add registers a project in the config file.
func Add(cmd *cobra.Command, _ []string) {
	project, err := cmd.Flags().GetString("project")
	cmdutil.ExitIfError(err)

	board, err := cmd.Flags().GetString("board")
	cmdutil.ExitIfError(err)

	noBoard, err := cmd.Flags().GetBool("no-board")
	cmdutil.ExitIfError(err)

	if noBoard && board != "" {
		cmdutil.Failed("Use either --board or --no-board, not both")
	}

	if err := jiraConfig.AddProject(jiraConfig.AddProjectOptions{
		Project: project,
		Board:   board,
		NoBoard: noBoard,
	}); err != nil {
		cmdutil.Failed("Unable to add project: %s", err.Error())
	}

	fmt.Println()
	cmdutil.Success("Project added to configuration: %s", viper.ConfigFileUsed())
}
