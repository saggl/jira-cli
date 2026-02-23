package remove

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmdcommon"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/internal/query"
)

const (
	helpText = `Remove deletes an attachment from an issue.`
	examples = `$ jira issue attachment remove ISSUE-1 12345

# Skip confirmation prompt
$ jira issue attachment remove ISSUE-1 12345 --no-input`
)

// NewCmdAttachmentRemove is an attachment remove command.
func NewCmdAttachmentRemove() *cobra.Command {
	cmd := cobra.Command{
		Use:     "remove ISSUE-KEY ATTACHMENT-ID",
		Short:   "Remove an attachment from an issue",
		Long:    helpText,
		Example: examples,
		Aliases: []string{"rm", "delete", "del"},
		Annotations: map[string]string{
			"help:args": "ISSUE-KEY\tIssue key, eg: ISSUE-1\n" +
				"ATTACHMENT-ID\tID of the attachment to remove",
		},
		Run: remove,
	}

	cmd.Flags().Bool("no-input", false, "Skip confirmation prompt")

	return &cmd
}

func remove(cmd *cobra.Command, args []string) {
	params := parseArgsAndFlags(args, cmd.Flags())
	client := api.DefaultClient(params.debug)

	if params.issueKey == "" {
		cmdutil.Failed("ISSUE-KEY is required")
	}

	if params.attachmentID == "" {
		cmdutil.Failed("ATTACHMENT-ID is required")
	}

	// Get issue to verify attachment exists and show filename
	issue, err := api.ProxyGetIssue(client, params.issueKey)
	cmdutil.ExitIfError(err)

	var attachmentFilename string
	found := false
	for _, a := range issue.Fields.Attachments {
		if a.ID == params.attachmentID {
			attachmentFilename = a.Filename
			found = true
			break
		}
	}

	if !found {
		cmdutil.Failed("Attachment with ID %q not found on issue %q", params.attachmentID, params.issueKey)
	}

	// Show confirmation unless --no-input is set
	if !params.noInput {
		answer := struct{ Action string }{}
		err := survey.Ask([]*survey.Question{
			{
				Name: "action",
				Prompt: &survey.Select{
					Message: fmt.Sprintf("Delete attachment %q (ID: %s) from %s?", attachmentFilename, params.attachmentID, params.issueKey),
					Options: []string{
						cmdcommon.ActionSubmit,
						cmdcommon.ActionCancel,
					},
				},
			},
		}, &answer)
		cmdutil.ExitIfError(err)

		if answer.Action == cmdcommon.ActionCancel {
			cmdutil.Failed("Action aborted")
		}
	}

	err = func() error {
		s := cmdutil.Info(fmt.Sprintf("Deleting attachment %s", attachmentFilename))
		defer s.Stop()

		return api.ProxyDeleteAttachment(client, params.attachmentID)
	}()
	cmdutil.ExitIfError(err)

	server := viper.GetString("server")

	cmdutil.Success("Deleted attachment %q from issue %q", attachmentFilename, params.issueKey)
	fmt.Printf("%s\n", cmdutil.GenerateServerBrowseURL(server, params.issueKey))
}

type removeParams struct {
	issueKey     string
	attachmentID string
	noInput      bool
	debug        bool
}

func parseArgsAndFlags(args []string, flags query.FlagParser) *removeParams {
	var issueKey, attachmentID string

	if len(args) >= 1 {
		issueKey = cmdutil.GetJiraIssueKey(viper.GetString("project.key"), args[0])
	}
	if len(args) >= 2 {
		attachmentID = args[1]
	}

	debug, err := flags.GetBool("debug")
	cmdutil.ExitIfError(err)

	noInput, err := flags.GetBool("no-input")
	cmdutil.ExitIfError(err)

	return &removeParams{
		issueKey:     issueKey,
		attachmentID: attachmentID,
		noInput:      noInput,
		debug:        debug,
	}
}
