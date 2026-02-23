package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

func TestProxyUploadAttachment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		installation     string
		expectedEndpoint string
	}{
		{
			name:             "cloud installation",
			installation:     jira.InstallationTypeCloud,
			expectedEndpoint: "/rest/api/3/issue/TEST-1/attachments",
		},
		{
			name:             "local installation",
			installation:     jira.InstallationTypeLocal,
			expectedEndpoint: "/rest/api/2/issue/TEST-1/attachments",
		},
		{
			name:             "default to cloud",
			installation:     "",
			expectedEndpoint: "/rest/api/3/issue/TEST-1/attachments",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, tc.expectedEndpoint, r.URL.Path)
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "no-check", r.Header.Get("X-Atlassian-Token"))

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(200)
				_, _ = w.Write([]byte(`[{
					"id": "10001",
					"filename": "test.txt",
					"author": {"displayName": "Test User"},
					"created": "2020-12-03T14:05:20.974+0100",
					"size": 12,
					"mimeType": "text/plain",
					"content": "http://example.com/attachment/10001"
				}]`))
			}))
			defer server.Close()

			client := jira.NewClient(jira.Config{
				Server:   server.URL,
				Login:    "test",
				APIToken: "token",
			}, jira.WithTimeout(3*time.Second))

			viper.Set("installation", tc.installation)

			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.txt")
			err := os.WriteFile(testFile, []byte("test content"), 0o644)
			assert.NoError(t, err)

			attachments, err := ProxyUploadAttachment(client, "TEST-1", testFile)
			assert.NoError(t, err)
			assert.Len(t, attachments, 1)
			assert.Equal(t, "10001", attachments[0].ID)
		})
	}
}

func TestProxyDeleteAttachment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		installation     string
		expectedEndpoint string
	}{
		{
			name:             "cloud installation",
			installation:     jira.InstallationTypeCloud,
			expectedEndpoint: "/rest/api/3/attachment/10001",
		},
		{
			name:             "local installation",
			installation:     jira.InstallationTypeLocal,
			expectedEndpoint: "/rest/api/2/attachment/10001",
		},
		{
			name:             "default to cloud",
			installation:     "",
			expectedEndpoint: "/rest/api/3/attachment/10001",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, tc.expectedEndpoint, r.URL.Path)
				assert.Equal(t, "DELETE", r.Method)

				w.WriteHeader(204)
			}))
			defer server.Close()

			client := jira.NewClient(jira.Config{
				Server:   server.URL,
				Login:    "test",
				APIToken: "token",
			}, jira.WithTimeout(3*time.Second))

			viper.Set("installation", tc.installation)

			err := ProxyDeleteAttachment(client, "10001")
			assert.NoError(t, err)
		})
	}
}
