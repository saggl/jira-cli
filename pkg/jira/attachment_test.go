package jira

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDownloadAttachment(t *testing.T) {
	t.Parallel()

	testContent := "This is test attachment content"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/attachments/test.txt", r.URL.Path)
		assert.Equal(t, "Basic dGVzdDp0b2tlbg==", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(testContent))
	}))
	defer server.Close()

	client := NewClient(Config{
		Server:   server.URL,
		Login:    "test",
		APIToken: "token",
	}, WithTimeout(3*time.Second))

	// Create temp file
	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "downloaded.txt")

	err := client.DownloadAttachment(server.URL+"/attachments/test.txt", destPath)
	assert.NoError(t, err)

	// Verify file was created and has correct content
	content, err := os.ReadFile(destPath)
	assert.NoError(t, err)
	assert.Equal(t, testContent, string(content))
}

func TestDownloadAttachmentError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer server.Close()

	client := NewClient(Config{
		Server:   server.URL,
		Login:    "test",
		APIToken: "token",
	}, WithTimeout(3*time.Second))

	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "downloaded.txt")

	err := client.DownloadAttachment(server.URL+"/attachments/test.txt", destPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to download attachment")
}

func TestUploadAttachment(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/issue/TEST-1/attachments", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.Header.Get("Content-Type"), "multipart/form-data")
		assert.Equal(t, "no-check", r.Header.Get("X-Atlassian-Token"))

		// Parse multipart form
		err := r.ParseMultipartForm(10 << 20)
		assert.NoError(t, err)

		file, header, err := r.FormFile("file")
		assert.NoError(t, err)
		assert.Equal(t, "test.txt", header.Filename)

		content, err := io.ReadAll(file)
		assert.NoError(t, err)
		assert.Equal(t, "test content", string(content))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`[{
			"id": "10001",
			"filename": "test.txt",
			"author": {
				"displayName": "Test User",
				"accountId": "123"
			},
			"created": "2020-12-03T14:05:20.974+0100",
			"size": 12,
			"mimeType": "text/plain",
			"content": "http://example.com/attachment/10001"
		}]`))
	}))
	defer server.Close()

	client := NewClient(Config{
		Server:   server.URL,
		Login:    "test",
		APIToken: "token",
	}, WithTimeout(3*time.Second))

	// Create temp file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0o644)
	assert.NoError(t, err)

	attachments, err := client.UploadAttachment("TEST-1", testFile)
	assert.NoError(t, err)
	assert.Len(t, attachments, 1)
	assert.Equal(t, "10001", attachments[0].ID)
	assert.Equal(t, "test.txt", attachments[0].Filename)
	assert.Equal(t, "Test User", attachments[0].Author.DisplayName)
	assert.Equal(t, int64(12), attachments[0].Size)
}

func TestUploadAttachmentV2(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/2/issue/TEST-1/attachments", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`[{
			"id": "10001",
			"filename": "test.txt",
			"author": {
				"displayName": "Test User",
				"name": "testuser"
			},
			"created": "2020-12-03T14:05:20.974+0100",
			"size": 12,
			"mimeType": "text/plain",
			"content": "http://example.com/attachment/10001"
		}]`))
	}))
	defer server.Close()

	client := NewClient(Config{
		Server:   server.URL,
		Login:    "test",
		APIToken: "token",
	}, WithTimeout(3*time.Second))

	// Create temp file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0o644)
	assert.NoError(t, err)

	attachments, err := client.UploadAttachmentV2("TEST-1", testFile)
	assert.NoError(t, err)
	assert.Len(t, attachments, 1)
	assert.Equal(t, "10001", attachments[0].ID)
}

func TestUploadAttachmentFileNotFound(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("Should not reach server")
	}))
	defer server.Close()

	client := NewClient(Config{
		Server:   server.URL,
		Login:    "test",
		APIToken: "token",
	}, WithTimeout(3*time.Second))

	_, err := client.UploadAttachment("TEST-1", "/nonexistent/file.txt")
	assert.Error(t, err)
	assert.True(t, os.IsNotExist(err))
}

func TestDeleteAttachment(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/attachment/10001", r.URL.Path)
		assert.Equal(t, "DELETE", r.Method)

		w.WriteHeader(204)
	}))
	defer server.Close()

	client := NewClient(Config{
		Server:   server.URL,
		Login:    "test",
		APIToken: "token",
	}, WithTimeout(3*time.Second))

	err := client.DeleteAttachment("10001")
	assert.NoError(t, err)
}

func TestDeleteAttachmentV2(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/2/attachment/10001", r.URL.Path)
		assert.Equal(t, "DELETE", r.Method)

		w.WriteHeader(204)
	}))
	defer server.Close()

	client := NewClient(Config{
		Server:   server.URL,
		Login:    "test",
		APIToken: "token",
	}, WithTimeout(3*time.Second))

	err := client.DeleteAttachmentV2("10001")
	assert.NoError(t, err)
}

func TestDeleteAttachmentError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`{"errorMessages":["Attachment not found"]}`))
	}))
	defer server.Close()

	client := NewClient(Config{
		Server:   server.URL,
		Login:    "test",
		APIToken: "token",
	}, WithTimeout(3*time.Second))

	err := client.DeleteAttachment("10001")
	assert.Error(t, err)
	assert.IsType(t, &ErrUnexpectedResponse{}, err)
}
