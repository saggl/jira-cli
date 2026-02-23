package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

// applyAuth applies authentication to the HTTP request.
func (c *Client) applyAuth(req *http.Request) {
	// Set default auth type to `basic`.
	if c.authType == nil {
		basic := AuthTypeBasic
		c.authType = &basic
	}

	// Apply authentication
	switch c.authType.String() {
	case string(AuthTypeMTLS):
		if c.token != "" {
			req.Header.Add("Authorization", "Bearer "+c.token)
		}
	case string(AuthTypeBearer):
		req.Header.Add("Authorization", "Bearer "+c.token)
	case string(AuthTypeBasic):
		req.SetBasicAuth(c.login, c.token)
	}
}

// DownloadAttachment downloads an attachment from the given URL to the specified file path.
func (c *Client) DownloadAttachment(url, destPath string) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	c.applyAuth(req)

	httpClient := &http.Client{Transport: c.transport}
	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download attachment: %s", res.Status)
	}

	// Create the destination file
	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	// Copy the response body to the file
	_, err = io.Copy(out, res.Body)
	return err
}

// UploadAttachment uploads a file as an attachment to the specified issue using v3 API.
func (c *Client) UploadAttachment(key, filePath string) ([]Attachment, error) {
	return c.uploadAttachment(key, filePath, apiVersion3)
}

// UploadAttachmentV2 uploads a file as an attachment to the specified issue using v2 API.
func (c *Client) UploadAttachmentV2(key, filePath string) ([]Attachment, error) {
	return c.uploadAttachment(key, filePath, apiVersion2)
}

func (c *Client) uploadAttachment(key, filePath, ver string) ([]Attachment, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	// Create a buffer to hold the multipart form data
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Create a form file field
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return nil, err
	}

	// Copy the file content to the form field
	_, err = io.Copy(part, file)
	if err != nil {
		return nil, err
	}

	// Close the writer to finalize the multipart message
	err = writer.Close()
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/issue/%s/attachments", key)

	// Set custom headers for multipart upload
	headers := Header{
		"Content-Type":      writer.FormDataContentType(),
		"X-Atlassian-Token": "no-check", // Required to bypass CSRF protection
	}

	var res *http.Response

	switch ver {
	case apiVersion2:
		res, err = c.postWithHeaders(context.Background(), c.server+baseURLv2+path, body.Bytes(), headers)
	default:
		res, err = c.postWithHeaders(context.Background(), c.server+baseURLv3+path, body.Bytes(), headers)
	}

	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, ErrEmptyResponse
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return nil, formatUnexpectedResponse(res)
	}

	var attachments []Attachment
	err = json.NewDecoder(res.Body).Decode(&attachments)
	if err != nil {
		return nil, err
	}

	return attachments, nil
}

// postWithHeaders is a helper method to send POST requests with custom body and headers.
func (c *Client) postWithHeaders(ctx context.Context, endpoint string, body []byte, headers Header) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	c.applyAuth(req)

	httpClient := &http.Client{Transport: c.transport}
	return httpClient.Do(req.WithContext(ctx))
}

// DeleteAttachment deletes an attachment using v3 API.
func (c *Client) DeleteAttachment(attachmentID string) error {
	return c.deleteAttachment(attachmentID, apiVersion3)
}

// DeleteAttachmentV2 deletes an attachment using v2 API.
func (c *Client) DeleteAttachmentV2(attachmentID string) error {
	return c.deleteAttachment(attachmentID, apiVersion2)
}

func (c *Client) deleteAttachment(attachmentID, ver string) error {
	path := fmt.Sprintf("/attachment/%s", attachmentID)

	var (
		res *http.Response
		err error
	)

	switch ver {
	case apiVersion2:
		res, err = c.DeleteV2(context.Background(), path, nil)
	default:
		// v3 doesn't have Delete method, need to add it to client
		res, err = c.delete(context.Background(), c.server+baseURLv3+path, nil)
	}

	if err != nil {
		return err
	}
	if res == nil {
		return ErrEmptyResponse
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusNoContent && res.StatusCode != http.StatusOK {
		return formatUnexpectedResponse(res)
	}

	return nil
}

// delete is a helper method to send DELETE requests (needed for v3 API).
func (c *Client) delete(ctx context.Context, endpoint string, headers Header) (*http.Response, error) {
	return c.request(ctx, http.MethodDelete, endpoint, nil, headers)
}
