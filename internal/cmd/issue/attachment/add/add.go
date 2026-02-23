package add

import (
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmdcommon"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/internal/query"
)

const (
	helpText = `Add uploads files as attachments to an issue.`
	examples = `$ jira issue attachment add ISSUE-1 file.pdf

# Upload multiple files
$ jira issue attachment add ISSUE-1 file1.pdf file2.png file3.txt

# Skip confirmation prompt
$ jira issue attachment add ISSUE-1 file.pdf --no-input`
)

// NewCmdAttachmentAdd is an attachment add command.
func NewCmdAttachmentAdd() *cobra.Command {
	cmd := cobra.Command{
		Use:     "add ISSUE-KEY FILE [FILE...]",
		Short:   "Add attachments to an issue",
		Long:    helpText,
		Example: examples,
		Aliases: []string{"upload"},
		Annotations: map[string]string{
			"help:args": "ISSUE-KEY\tIssue key, eg: ISSUE-1\n" +
				"FILE\tPath to file(s) to upload",
		},
		Run: add,
	}

	cmd.Flags().Bool("no-input", false, "Skip confirmation prompt")

	return &cmd
}

func add(cmd *cobra.Command, args []string) {
	params := parseArgsAndFlags(args, cmd.Flags())
	client := api.DefaultClient(params.debug)

	if params.issueKey == "" {
		cmdutil.Failed("ISSUE-KEY is required")
	}

	if len(params.files) == 0 {
		cmdutil.Failed("At least one file path is required")
	}

	// Validate that all files exist
	for _, file := range params.files {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			cmdutil.Failed("File %q does not exist", file)
		}
	}

	// Show confirmation unless --no-input is set
	if !params.noInput {
		fileList := ""
		for _, file := range params.files {
			fileList += fmt.Sprintf("  - %s\n", file)
		}

		answer := struct{ Action string }{}
		err := survey.Ask([]*survey.Question{
			{
				Name: "action",
				Prompt: &survey.Select{
					Message: fmt.Sprintf("Upload %d file(s) to %s?\n%s", len(params.files), params.issueKey, fileList),
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

	// Upload each file
	for _, file := range params.files {
		err := func() error {
			s := cmdutil.Info(fmt.Sprintf("Uploading %s", file))
			defer s.Stop()

			_, err := api.ProxyUploadAttachment(client, params.issueKey, file)
			return err
		}()
		cmdutil.ExitIfError(err)

		cmdutil.Success("Uploaded %q to issue %q", file, params.issueKey)
	}

	server := viper.GetString("server")
	fmt.Printf("%s\n", cmdutil.GenerateServerBrowseURL(server, params.issueKey))
}

type addParams struct {
	issueKey string
	files    []string
	noInput  bool
	debug    bool
}

func parseArgsAndFlags(args []string, flags query.FlagParser) *addParams {
	var issueKey string
	var files []string

	if len(args) >= 1 {
		issueKey = cmdutil.GetJiraIssueKey(viper.GetString("project.key"), args[0])
	}
	if len(args) >= 2 {
		files = args[1:]
	}

	debug, err := flags.GetBool("debug")
	cmdutil.ExitIfError(err)

	noInput, err := flags.GetBool("no-input")
	cmdutil.ExitIfError(err)

	return &addParams{
		issueKey: issueKey,
		files:    files,
		noInput:  noInput,
		debug:    debug,
	}
}
