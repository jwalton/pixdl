package download

import (
	"context"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

// RemoteFileInfo represents information about a file on a remote server.
// This is a collection of information useful in deciding how to download a file.
type RemoteFileInfo struct {
	// Size is the size of the file, or -1 if unknown.
	Size      int64
	Filename  string
	MimeType  string
	CanResume bool
	// Last-modified header, if present.
	LastModified *time.Time
}

func newRemoteFileInfo() *RemoteFileInfo {
	return &RemoteFileInfo{
		-1,
		"",
		"",
		false,
		nil,
	}
}

// GetFileInfo is a convenience function for DoFileInfo.
func (client *Client) GetFileInfo(url string) (*RemoteFileInfo, error) {
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return newRemoteFileInfo(), err
	}

	return client.DoFileInfo(req)
}

// DoFileInfo returns the length, mime type, last modified time, and other
// interesting information about the resource for the specified request.
func (client *Client) DoFileInfo(request *http.Request) (*RemoteFileInfo, error) {
	headReq := request

	if headReq.Method != "HEAD" {
		// Copy the request, make it a HEAD request.
		headReq := request.Clone(context.Background())
		headReq.Method = "HEAD"
	}

	resp, err := client.httpClient.Do(headReq)

	if err != nil || resp.StatusCode != 200 {
		return newRemoteFileInfo(), err
	}

	defer resp.Body.Close()

	size := getContentLength(resp)
	resume := false
	if size > -1 {
		resume = canResume(resp)
	}

	return &RemoteFileInfo{
		size,
		getFilename(resp),
		parseContentType(resp.Header.Get("content-type")),
		resume,
		getLastModified(resp),
	}, nil
}

func getContentLength(resp *http.Response) int64 {
	var contentLength int64

	lenStr := resp.Header.Get("content-length")
	if lenStr == "" {
		contentLength = -1
	} else {
		var err error
		if contentLength, err = strconv.ParseInt(lenStr, 10, 64); err != nil {
			contentLength = -1
		}
	}

	if contentLength < 0 {
		contentLength = -1
	}

	return contentLength
}

var contentDispositionRegex = regexp.MustCompile(`^attachment;.*filename="([^"]*)".*$`)

func getFilename(resp *http.Response) string {
	contentDisposition := resp.Header.Get("content-disposition")
	match := contentDispositionRegex.FindStringSubmatch(contentDisposition)
	if match != nil {
		return match[1]
	}
	return ""
}

// httpToken is the regex to get a "token" from RFC7230, S3.2.6.
const httpToken = "[!#$%&'*+-.^_`|~0-9a-zA-Z]"

// mediaTypeRegex parses media type, as per RFC7231, S3.1.1.1.
var mediaTypeRegex = regexp.MustCompile("(" + httpToken + "*/" + httpToken + "*).*")

func parseContentType(contentType string) string {
	match := mediaTypeRegex.FindStringSubmatch(contentType)
	if match != nil {
		return match[1]
	}
	return ""
}

func canResume(resp *http.Response) bool {
	return resp.Header.Get("accept-ranges") == "bytes"
}

func getLastModified(resp *http.Response) *time.Time {
	header := resp.Header.Get("last-modified")
	if header != "" {
		lastModified, err := time.Parse(http.TimeFormat, header)
		if err != nil {
			return &lastModified
		}
	}
	return nil
}
