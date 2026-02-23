package list

import (
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/internal/query"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

const (
	helpText = `List attachments on an issue.`
	examples = `$ jira issue attachment list ISSUE-1

# List attachments in CSV format
$ jira issue attachment list ISSUE-1 --csv

# List attachments in plain text
$ jira issue attachment list ISSUE-1 --plain`
)

// NewCmdAttachmentList is an attachment list command.
func NewCmdAttachmentList() *cobra.Command {
	cmd := cobra.Command{
		Use:     "list ISSUE-KEY",
		Short:   "List attachments on an issue",
		Long:    helpText,
		Example: examples,
		Aliases: []string{"ls"},
		Annotations: map[string]string{
			"help:args": "ISSUE-KEY\tIssue key, eg: ISSUE-1",
		},
		Run: list,
	}

	cmd.Flags().Bool("plain", false, "Plain text output")
	cmd.Flags().Bool("csv", false, "CSV output")

	return &cmd
}

func list(cmd *cobra.Command, args []string) {
	params := parseArgsAndFlags(args, cmd.Flags())
	client := api.DefaultClient(params.debug)

	if params.issueKey == "" {
		cmdutil.Failed("ISSUE-KEY is required")
	}

	issue, err := api.ProxyGetIssue(client, params.issueKey)
	cmdutil.ExitIfError(err)

	if len(issue.Fields.Attachments) == 0 {
		cmdutil.Success("No attachments found for issue %q", params.issueKey)
		return
	}

	if params.csv {
		renderCSV(os.Stdout, issue.Fields.Attachments)
	} else if params.plain {
		renderPlain(os.Stdout, issue.Fields.Attachments)
	} else {
		renderTable(os.Stdout, issue.Fields.Attachments)
	}
}

type listParams struct {
	issueKey string
	plain    bool
	csv      bool
	debug    bool
}

func parseArgsAndFlags(args []string, flags query.FlagParser) *listParams {
	var issueKey string

	if len(args) >= 1 {
		issueKey = cmdutil.GetJiraIssueKey(viper.GetString("project.key"), args[0])
	}

	debug, err := flags.GetBool("debug")
	cmdutil.ExitIfError(err)

	plain, err := flags.GetBool("plain")
	cmdutil.ExitIfError(err)

	csv, err := flags.GetBool("csv")
	cmdutil.ExitIfError(err)

	return &listParams{
		issueKey: issueKey,
		plain:    plain,
		csv:      csv,
		debug:    debug,
	}
}

func renderTable(w io.Writer, attachments []jira.Attachment) {
	tw := tabwriter.NewWriter(w, 0, 8, 1, '\t', 0)

	fmt.Fprintf(tw, "ID\tFILENAME\tSIZE\tAUTHOR\tCREATED\n")

	for _, a := range attachments {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			a.ID,
			a.Filename,
			formatSize(a.Size),
			a.Author.DisplayName,
			formatDate(a.Created),
		)
	}

	_ = tw.Flush()
}

func renderPlain(w io.Writer, attachments []jira.Attachment) {
	tw := tabwriter.NewWriter(w, 0, 8, 1, '\t', 0)

	for _, a := range attachments {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			a.ID,
			a.Filename,
			formatSize(a.Size),
			a.Author.DisplayName,
			formatDate(a.Created),
		)
	}

	_ = tw.Flush()
}

func renderCSV(w io.Writer, attachments []jira.Attachment) {
	fmt.Fprintf(w, "ID,FILENAME,SIZE,AUTHOR,CREATED\n")

	for _, a := range attachments {
		fmt.Fprintf(w, "%s,%s,%d,%s,%s\n",
			a.ID,
			escapeCSV(a.Filename),
			a.Size,
			escapeCSV(a.Author.DisplayName),
			a.Created,
		)
	}
}

func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func formatDate(date string) string {
	// Just return the raw date for now - could be enhanced with better formatting
	if len(date) > 10 {
		return date[:10]
	}
	return date
}

func escapeCSV(s string) string {
	if containsSpecialChar(s) {
		return fmt.Sprintf(`"%s"`, s)
	}
	return s
}

func containsSpecialChar(s string) bool {
	for _, ch := range s {
		if ch == ',' || ch == '"' || ch == '\n' {
			return true
		}
	}
	return false
}
