package moveproject

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

const helpText = `Move issue to a different Jira project.`

func NewCmdMoveProject() *cobra.Command {
	cmd := cobra.Command{
		Use:   "move-project ISSUE-KEY TARGET-PROJECT",
		Short: "Move issue to a different project",
		Long:  helpText,
		Args:  cobra.ExactArgs(2),
		Annotations: map[string]string{
			"help:args": `ISSUE-KEY\tIssue key, eg: ISSUE-1
TARGET-PROJECT\tProject key to move the issue to`,
		},
		RunE: moveProject,
	}

	cmd.Flags().String("issue-type", "", "Issue type in the target project")
	cmd.Flags().Bool("dry-run", false, "Preview the project move without submitting it")

	return &cmd
}

func moveProject(cmd *cobra.Command, args []string) error {
	debug, err := cmd.Flags().GetBool("debug")
	cmdutil.ExitIfError(err)

	issueType, err := cmd.Flags().GetString("issue-type")
	cmdutil.ExitIfError(err)

	dryRun, err := cmd.Flags().GetBool("dry-run")
	cmdutil.ExitIfError(err)

	server := viper.GetString("server")
	login := viper.GetString("login")
	token := viper.GetString("api_token")
	authType := jira.AuthType(viper.GetString("auth_type"))
	insecure := viper.GetBool("insecure")

	restClient := api.DefaultClient(debug)

	var sessionOpts []func(*http.Transport)
	if insecure {
		sessionOpts = append(sessionOpts, func(t *http.Transport) {
			if t.TLSClientConfig == nil {
				t.TLSClientConfig = &tls.Config{}
			}
			t.TLSClientConfig.InsecureSkipVerify = true
		})
	}
	if authType == jira.AuthTypeMTLS {
		mtlsConfig := jira.MTLSConfig{
			CaCert:     viper.GetString("mtls.ca_cert"),
			ClientCert: viper.GetString("mtls.client_cert"),
			ClientKey:  viper.GetString("mtls.client_key"),
		}
		sessionOpts = append(sessionOpts, func(t *http.Transport) {
			if t.TLSClientConfig == nil {
				t.TLSClientConfig = &tls.Config{}
			}
			if mtlsConfig.CaCert != "" {
				caCert, readErr := os.ReadFile(mtlsConfig.CaCert)
				cmdutil.ExitIfError(readErr)
				pool := x509.NewCertPool()
				pool.AppendCertsFromPEM(caCert)
				t.TLSClientConfig.RootCAs = pool
			}
			if mtlsConfig.ClientCert != "" || mtlsConfig.ClientKey != "" {
				cert, loadErr := tls.LoadX509KeyPair(mtlsConfig.ClientCert, mtlsConfig.ClientKey)
				cmdutil.ExitIfError(loadErr)
				t.TLSClientConfig.Certificates = []tls.Certificate{cert}
				t.TLSClientConfig.Renegotiation = tls.RenegotiateFreelyAsClient
			}
		})
	}

	sessionClient, err := jira.NewSessionClient(server, login, token, authType, sessionOpts...)
	cmdutil.ExitIfError(err)

	result, err := jira.MoveProject(sessionClient, restClient, jira.MoveProjectParams{
		IssueKey:      cmdutil.GetJiraIssueKey(viper.GetString("project.key"), args[0]),
		TargetProject: args[1],
		IssueType:     issueType,
		DryRun:        dryRun,
	})
	cmdutil.ExitIfError(err)

	if dryRun {
		fmt.Printf("Would move %s from %s to %s as %s\n", result.OldKey, result.OldProject, result.NewProject, result.IssueType)
		return nil
	}

	fmt.Printf("Issue moved: %s → %s\n", result.OldKey, result.NewKey)
	return nil
}
