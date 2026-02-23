package download

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/internal/query"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

const (
	helpText = `Download attachments from an issue.`
	examples = `$ jira issue attachment download ISSUE-1 --all

# Download specific file
$ jira issue attachment download ISSUE-1 document.pdf

# Download by attachment ID
$ jira issue attachment download ISSUE-1 --id 12345

# Download to specific directory
$ jira issue attachment download ISSUE-1 --all --output /path/to/dir`
)

// NewCmdAttachmentDownload is an attachment download command.
func NewCmdAttachmentDownload() *cobra.Command {
	cmd := cobra.Command{
		Use:     "download ISSUE-KEY [FILENAME]",
		Short:   "Download attachments from an issue",
		Long:    helpText,
		Example: examples,
		Aliases: []string{"dl", "get"},
		Annotations: map[string]string{
			"help:args": "ISSUE-KEY\tIssue key, eg: ISSUE-1\n" +
				"FILENAME\tOptional filename to download",
		},
		Run: download,
	}

	cmd.Flags().Bool("all", false, "Download all attachments")
	cmd.Flags().String("id", "", "Download attachment by ID")
	cmd.Flags().StringP("output", "o", ".", "Output directory")

	return &cmd
}

func download(cmd *cobra.Command, args []string) {
	params := parseArgsAndFlags(args, cmd.Flags())
	client := api.DefaultClient(params.debug)

	if params.issueKey == "" {
		cmdutil.Failed("ISSUE-KEY is required")
	}

	issue, err := api.ProxyGetIssue(client, params.issueKey)
	cmdutil.ExitIfError(err)

	if len(issue.Fields.Attachments) == 0 {
		cmdutil.Failed("No attachments found for issue %q", params.issueKey)
	}

	// Create output directory if it doesn't exist
	if params.outputDir != "." {
		err := os.MkdirAll(params.outputDir, 0o755)
		cmdutil.ExitIfError(err)
	}

	// Determine which attachments to download
	var attachmentsToDownload []jira.Attachment
	switch {
	case params.all:
		attachmentsToDownload = issue.Fields.Attachments
	case params.id != "":
		attachmentsToDownload = findAttachmentByID(issue.Fields.Attachments, params.id)
	case params.filename != "":
		attachmentsToDownload = findAttachmentByFilename(issue.Fields.Attachments, params.filename)
	default:
		cmdutil.Failed("Please specify --all, --id, or provide a filename")
	}

	// Download attachments
	for _, a := range attachmentsToDownload {
		destPath := filepath.Join(params.outputDir, a.Filename)

		// Check if file already exists
		if _, err := os.Stat(destPath); err == nil {
			cmdutil.Failed("File %q already exists. Please remove it or use a different output directory", destPath)
		}

		err := func() error {
			s := cmdutil.Info(fmt.Sprintf("Downloading %s", a.Filename))
			defer s.Stop()

			return client.DownloadAttachment(a.Content, destPath)
		}()
		cmdutil.ExitIfError(err)

		cmdutil.Success("Downloaded %q to %s", a.Filename, destPath)
	}
}

type downloadParams struct {
	issueKey  string
	filename  string
	all       bool
	id        string
	outputDir string
	debug     bool
}

func parseArgsAndFlags(args []string, flags query.FlagParser) *downloadParams {
	var issueKey, filename string

	if len(args) >= 1 {
		issueKey = cmdutil.GetJiraIssueKey(viper.GetString("project.key"), args[0])
	}
	if len(args) >= 2 {
		filename = args[1]
	}

	debug, err := flags.GetBool("debug")
	cmdutil.ExitIfError(err)

	all, err := flags.GetBool("all")
	cmdutil.ExitIfError(err)

	id, err := flags.GetString("id")
	cmdutil.ExitIfError(err)

	outputDir, err := flags.GetString("output")
	cmdutil.ExitIfError(err)

	return &downloadParams{
		issueKey:  issueKey,
		filename:  filename,
		all:       all,
		id:        id,
		outputDir: outputDir,
		debug:     debug,
	}
}

func findAttachmentByID(attachments []jira.Attachment, id string) []jira.Attachment {
	for _, a := range attachments {
		if a.ID == id {
			return []jira.Attachment{a}
		}
	}
	cmdutil.Failed("Attachment with ID %q not found", id)
	return nil
}

func findAttachmentByFilename(attachments []jira.Attachment, filename string) []jira.Attachment {
	for _, a := range attachments {
		if a.Filename == filename {
			return []jira.Attachment{a}
		}
	}
	cmdutil.Failed("Attachment with filename %q not found", filename)
	return nil
}
