package view

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

func TestFormatAttachmentSize(t *testing.T) {
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
			bytes:    1536,
			expected: "1.50 KB",
		},
		{
			name:     "megabytes",
			bytes:    1048576,
			expected: "1.00 MB",
		},
		{
			name:     "megabytes decimal",
			bytes:    5242880,
			expected: "5.00 MB",
		},
		{
			name:     "gigabytes",
			bytes:    1073741824,
			expected: "1.00 GB",
		},
		{
			name:     "gigabytes decimal",
			bytes:    2147483648,
			expected: "2.00 GB",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			actual := formatAttachmentSize(tc.bytes)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestIssueAttachmentsDisplay(t *testing.T) {
	t.Parallel()

	data := &jira.Issue{
		Key: "TEST-1",
		Fields: jira.IssueFields{
			Summary: "Test issue with attachments",
			Attachments: []jira.Attachment{
				{
					ID:       "10001",
					Filename: "document.pdf",
					Author: jira.User{
						DisplayName: "John Doe",
						AccountID:   "123",
					},
					Created:  "2020-12-01T10:00:00.000+0100",
					Size:     1048576,
					MimeType: "application/pdf",
					Content:  "https://example.com/attachment/10001",
				},
				{
					ID:       "10002",
					Filename: "screenshot.png",
					Author: jira.User{
						DisplayName: "Jane Smith",
						AccountID:   "456",
					},
					Created:  "2020-12-02T15:30:00.000+0100",
					Size:     524288,
					MimeType: "image/png",
					Content:  "https://example.com/attachment/10002",
				},
			},
			IssueType: jira.IssueType{Name: "Task"},
			Status: struct {
				Name string `json:"name"`
			}{Name: "To Do"},
			Priority: struct {
				Name string `json:"name"`
			}{Name: "Medium"},
			Reporter: struct {
				Name string `json:"displayName"`
			}{Name: "Reporter"},
			Comment: struct {
				Comments []struct {
					ID      string      `json:"id"`
					Author  jira.User   `json:"author"`
					Body    interface{} `json:"body"`
					Created string      `json:"created"`
				} `json:"comments"`
				Total int `json:"total"`
			}{Total: 0},
			Watches: struct {
				IsWatching bool `json:"isWatching"`
				WatchCount int  `json:"watchCount"`
			}{IsWatching: false, WatchCount: 0},
			Created: "2020-12-01T09:00:00.000+0100",
			Updated: "2020-12-02T16:00:00.000+0100",
		},
	}

	issue := Issue{
		Server:  "https://test.local",
		Data:    data,
		Display: DisplayFormat{Plain: false},
	}

	attachmentsOutput := issue.attachments()
	assert.Contains(t, attachmentsOutput, "ATTACHMENTS")
	assert.Contains(t, attachmentsOutput, "document.pdf")
	assert.Contains(t, attachmentsOutput, "screenshot.png")
	assert.Contains(t, attachmentsOutput, "John Doe")
	assert.Contains(t, attachmentsOutput, "Jane Smith")
	assert.Contains(t, attachmentsOutput, "1.00 MB")
	assert.Contains(t, attachmentsOutput, "512.00 KB")
}

func TestIssueWithoutAttachments(t *testing.T) {
	t.Parallel()

	data := &jira.Issue{
		Key: "TEST-1",
		Fields: jira.IssueFields{
			Summary:     "Test issue without attachments",
			Attachments: []jira.Attachment{},
			IssueType:   jira.IssueType{Name: "Task"},
			Status: struct {
				Name string `json:"name"`
			}{Name: "To Do"},
			Priority: struct {
				Name string `json:"name"`
			}{Name: "Medium"},
			Reporter: struct {
				Name string `json:"displayName"`
			}{Name: "Reporter"},
			Comment: struct {
				Comments []struct {
					ID      string      `json:"id"`
					Author  jira.User   `json:"author"`
					Body    interface{} `json:"body"`
					Created string      `json:"created"`
				} `json:"comments"`
				Total int `json:"total"`
			}{Total: 0},
			Watches: struct {
				IsWatching bool `json:"isWatching"`
				WatchCount int  `json:"watchCount"`
			}{IsWatching: false, WatchCount: 0},
			Created: "2020-12-01T09:00:00.000+0100",
			Updated: "2020-12-02T16:00:00.000+0100",
		},
	}

	issue := Issue{
		Server:  "https://test.local",
		Data:    data,
		Display: DisplayFormat{Plain: false},
	}

	attachmentsOutput := issue.attachments()
	assert.Equal(t, "", attachmentsOutput)

	// Ensure attachments section is not in the full output
	fullOutput := issue.String()
	assert.NotContains(t, fullOutput, "Attachments")
}
