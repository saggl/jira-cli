package list

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

func TestFormatSize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{
			name:     "bytes",
			bytes:    500,
			expected: "500 B",
		},
		{
			name:     "kilobytes",
			bytes:    2048,
			expected: "2.00 KB",
		},
		{
			name:     "megabytes",
			bytes:    5242880,
			expected: "5.00 MB",
		},
		{
			name:     "gigabytes",
			bytes:    2147483648,
			expected: "2.00 GB",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			actual := formatSize(tc.bytes)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestFormatDate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		date     string
		expected string
	}{
		{
			name:     "full date",
			date:     "2020-12-01T10:00:00.000+0100",
			expected: "2020-12-01",
		},
		{
			name:     "short date",
			date:     "2020-12-01",
			expected: "2020-12-01",
		},
		{
			name:     "very short",
			date:     "2020",
			expected: "2020",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			actual := formatDate(tc.date)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestEscapeCSV(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no special chars",
			input:    "normal text",
			expected: "normal text",
		},
		{
			name:     "with comma",
			input:    "text, with comma",
			expected: `"text, with comma"`,
		},
		{
			name:     "with quote",
			input:    `text "quoted"`,
			expected: `"text "quoted""`,
		},
		{
			name:     "with newline",
			input:    "text\nwith newline",
			expected: "\"text\nwith newline\"",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			actual := escapeCSV(tc.input)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestRenderTable(t *testing.T) {
	t.Parallel()

	attachments := []jira.Attachment{
		{
			ID:       "10001",
			Filename: "document.pdf",
			Author: jira.User{
				DisplayName: "John Doe",
			},
			Created: "2020-12-01T10:00:00.000+0100",
			Size:    1048576,
		},
		{
			ID:       "10002",
			Filename: "screenshot.png",
			Author: jira.User{
				DisplayName: "Jane Smith",
			},
			Created: "2020-12-02T15:30:00.000+0100",
			Size:    524288,
		},
	}

	var buf bytes.Buffer
	renderTable(&buf, attachments)

	output := buf.String()
	assert.Contains(t, output, "ID")
	assert.Contains(t, output, "FILENAME")
	assert.Contains(t, output, "SIZE")
	assert.Contains(t, output, "AUTHOR")
	assert.Contains(t, output, "CREATED")
	assert.Contains(t, output, "10001")
	assert.Contains(t, output, "document.pdf")
	assert.Contains(t, output, "1.00 MB")
	assert.Contains(t, output, "John Doe")
	assert.Contains(t, output, "2020-12-01")
	assert.Contains(t, output, "10002")
	assert.Contains(t, output, "screenshot.png")
	assert.Contains(t, output, "512.00 KB")
	assert.Contains(t, output, "Jane Smith")
	assert.Contains(t, output, "2020-12-02")
}

func TestRenderPlain(t *testing.T) {
	t.Parallel()

	attachments := []jira.Attachment{
		{
			ID:       "10001",
			Filename: "document.pdf",
			Author: jira.User{
				DisplayName: "John Doe",
			},
			Created: "2020-12-01T10:00:00.000+0100",
			Size:    1048576,
		},
	}

	var buf bytes.Buffer
	renderPlain(&buf, attachments)

	output := buf.String()
	assert.Contains(t, output, "10001")
	assert.Contains(t, output, "document.pdf")
	assert.Contains(t, output, "1.00 MB")
	assert.Contains(t, output, "John Doe")
	assert.NotContains(t, output, "ID") // Plain mode doesn't have headers
}

func TestRenderCSV(t *testing.T) {
	t.Parallel()

	attachments := []jira.Attachment{
		{
			ID:       "10001",
			Filename: "document.pdf",
			Author: jira.User{
				DisplayName: "John Doe",
			},
			Created: "2020-12-01T10:00:00.000+0100",
			Size:    1048576,
		},
		{
			ID:       "10002",
			Filename: "file, with comma.txt",
			Author: jira.User{
				DisplayName: "Jane Smith",
			},
			Created: "2020-12-02T15:30:00.000+0100",
			Size:    524288,
		},
	}

	var buf bytes.Buffer
	renderCSV(&buf, attachments)

	output := buf.String()
	assert.Contains(t, output, "ID,FILENAME,SIZE,AUTHOR,CREATED")
	assert.Contains(t, output, "10001,document.pdf,1048576,John Doe,2020-12-01T10:00:00.000+0100")
	assert.Contains(t, output, `10002,"file, with comma.txt",524288,Jane Smith,2020-12-02T15:30:00.000+0100`)
}
