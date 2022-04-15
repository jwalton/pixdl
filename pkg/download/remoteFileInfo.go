package download

import (
	"context"
	"net/http"
	"regexp"
	"time"
)

// RemoteFileInfo represents information about a file on a remote server.
// This is a collection of information useful in deciding how to download a file.
type RemoteFileInfo struct {
	// URL is the actual URL of the file.  Usually this will be the URL passed in,
	// but if there's a redirect, this will be the URL we were ultimately redirected to.
	URL string
	// Size is the size of the file, or -1 if unknown.
	Size int64
	// Filename is the name of the file.
	Filename string
	// MimeType is the MIME type returned by the server.
	MimeType string
	// CanResume is true if the server has a "accept-ranges: bytes" header.
	CanResume bool
	// Last-modified header, if present.
	LastModified *time.Time
}

func newRemoteFileInfo(url string) *RemoteFileInfo {
	return &RemoteFileInfo{
		url,
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
		return newRemoteFileInfo(url), err
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
		return newRemoteFileInfo(request.URL.String()), err
	}

	defer resp.Body.Close()

	resume := false
	if resp.ContentLength > -1 {
		resume = canResume(resp)
	}

	return &RemoteFileInfo{
		// Get the actual URL we ended up fetching from.  This could be different
		// than the original URL if there was a redirect.
		resp.Request.URL.String(),
		resp.ContentLength,
		getFilename(resp),
		parseContentType(resp.Header.Get("content-type")),
		resume,
		getLastModified(resp),
	}, nil
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
