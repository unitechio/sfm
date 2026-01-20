package airdrop

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Client for sending files
type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

// SendFile sends a file to a remote device
func (c *Client) SendFile(targetIP string, targetPort int, filePath string, onProgress func(sent, total int64)) error {
	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// Send request first
	metadata := FileMetadata{
		Name: filepath.Base(filePath),
		Size: fileInfo.Size(),
		Mime: "application/octet-stream",
	}

	reqData := TransferRequest{
		Metadata: metadata,
	}

	reqURL := fmt.Sprintf("http://%s:%d/request", targetIP, targetPort)
	reqBody, _ := json.Marshal(reqData)

	resp, err := http.Post(reqURL, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	var transferResp TransferResponse
	if err := json.NewDecoder(resp.Body).Decode(&transferResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !transferResp.Accepted {
		return fmt.Errorf("transfer rejected: %s", transferResp.Message)
	}

	// Send file
	sendURL := fmt.Sprintf("http://%s:%d/send", targetIP, targetPort)

	var body io.Reader = file
	if onProgress != nil {
		body = &progressReader{
			reader:     file,
			total:      fileInfo.Size(),
			onProgress: onProgress,
		}
	}

	req, err := http.NewRequest(http.MethodPost, sendURL, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-File-Name", filepath.Base(filePath))
	req.ContentLength = fileInfo.Size()

	sendResp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send file: %w", err)
	}
	defer sendResp.Body.Close()

	if sendResp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned error: %d", sendResp.StatusCode)
	}

	return nil
}

type progressReader struct {
	reader     io.Reader
	total      int64
	sent       int64
	onProgress func(sent, total int64)
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.sent += int64(n)
	if pr.onProgress != nil {
		pr.onProgress(pr.sent, pr.total)
	}
	return n, err
}
