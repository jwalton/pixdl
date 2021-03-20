package download

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const partialSuffix = ".part"
const defaultMaxRetries = 5
const defaultRetryDelay = 5 * time.Second

// DownloadClient is a client to use for downloading files.  Note that you must
// construct a DownloadClient via `NewDownloadClient`.
type DownloadClient struct {
	httpClient *http.Client
	MaxRetires uint
	RetryDelay time.Duration
}

type Option func(client *DownloadClient)

// WithClient is an option for NewDownloadClient that allows you to specify
// the http.Client to use to download files.  If unspecified, the DownloadClient
// will use http.DefaultClient.
func WithClient(httpClient *http.Client) Option {
	return func(client *DownloadClient) {
		client.httpClient = httpClient
	}
}

// MaxRetries is an option for NewDownloadClient that sets the maximum number
// of times the DownloadClient will attempt to download the same file before
// giving up.  DownloadClient will only attempt to retry for "recoverable"
// errors, such as 5xx errors from the server, or similar.
func MaxRetries(retries uint) Option {
	return func(client *DownloadClient) {
		client.MaxRetires = retries
	}
}

// NewDownloadClient creates a new DownloadClient.
func NewDownloadClient(options ...Option) *DownloadClient {
	client := &DownloadClient{
		httpClient: http.DefaultClient,
		MaxRetires: defaultMaxRetries,
	}

	for _, option := range options {
		option(client)
	}

	return client
}

// GetFile downloads a file using a simple GET request to the specified URL.
func (client *DownloadClient) GetFile(url string, filename string, reporter FileProgressCallback) (written int64, err error) {
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		reporter(newErrorProgress(request, url, filename, nil, err))
		return 0, err
	}

	return client.Do(request, filename, reporter)
}

func (client *DownloadClient) Do(
	request *http.Request,
	filename string,
	reporter FileProgressCallback,
) (written int64, err error) {
	return client.DoWithFileInfo(request, filename, nil, reporter)
}

// DoWithFileInfo is simliar to Do(), but will not try to fetch RemoteFileInfo from the
// remote server - this is handy when you've already fetched the file info.  If
// you pass `nil` for remoteInfo, then DoWithFileInfo will still try to fetch
// the RemoteFileInfo.
func (client *DownloadClient) DoWithFileInfo(
	request *http.Request,
	filename string,
	remoteInfo *RemoteFileInfo,
	reporter FileProgressCallback,
) (written int64, err error) {
	if remoteInfo == nil {
		// Ignore error from DoFileInfo - possibly the remote doesn't support HEAD.
		// Press on, and we'll probably error out down below.
		remoteInfo, _ = client.DoFileInfo(request)
	}
	pw := newProgressWriter(request, filename, remoteInfo, reporter)
	var totalWritten int64 = 0

	triesLeft := client.MaxRetires + 1
	downloading := true
	for downloading {
		written, httpErr := client.doDownload(request, filename, remoteInfo, pw)
		totalWritten += written

		if httpErr != nil {
			if httpErr.canRetry {
				if strings.Contains(httpErr.Error(), "INTERNAL_ERROR") && written > 0 && remoteInfo.CanResume {
					// See these fairly often from some servers - if we see these
					// and `remoteInfo.CanResume`, then don't count against retriesLeft,
					// keep going.
				} else {
					pw.Warn(fmt.Sprintf("Error: %v - will retry", httpErr))
					triesLeft--
					// Short pause here, to give the server time to think about it's life choices...
					<-time.After(client.RetryDelay)
				}
			}

			if !httpErr.canRetry || triesLeft <= 0 {
				downloading = false
				err = httpErr
			}
		} else {
			// We're done!
			downloading = false
		}
	}

	pw.Close(err)

	return totalWritten, err
}

// openFile opens a file for writing.
//
// If canResume is true, and the file already exists, we'll open it for appending.
// Returns the file, the place we would like to start writing in the file, and
// an error.
func openFileForWriting(filename string, canResume bool) (file *os.File, size int64, httpErr *httpError) {
	// See if the file already exists.
	info, err := os.Stat(filename)
	if errors.Is(err, os.ErrNotExist) {
		// Drop throught to create a new file case below.
	} else if err != nil {
		return nil, 0, &httpError{message: "Could not stat file " + filename}
	} else if canResume {
		// We can resume!
		file, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			return nil, 0, &httpError{message: "Could not open file for appending"}
		}
		return file, info.Size(), nil
	}

	// Create a new file
	file, err = os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return nil, 0, &httpError{message: "Could not open file"}
	}
	return file, 0, nil
}

func (client *DownloadClient) doDownload(
	request *http.Request,
	filename string,
	remoteInfo *RemoteFileInfo,
	pw *progressWriter,
) (written int64, httpErr *httpError) {
	file, existingSize, httpErr := openFileForWriting(filename+partialSuffix, remoteInfo.CanResume)
	// Don't defer close of the file so we can rename the file after we close it.

	if httpErr != nil {
		return 0, httpErr
	}

	// Start the HTTP request.
	var err error
	var resp *http.Response
	if existingSize > 0 {
		resp, err = client.resumeDownload(request, existingSize, remoteInfo.Size-1)
		pw.setSize(existingSize)
	} else {
		resp, err = client.httpClient.Do(request)
		pw.setSize(0)
	}

	if err != nil {
		return 0, &httpError{message: err.Error()}
	}

	defer resp.Body.Close()

	// Verify the response code.
	if resp.StatusCode >= 500 && resp.StatusCode < 600 {
		// Server error - should retry
		return 0, &httpError{canRetry: true, message: fmt.Sprintf("Server replied with %d", resp.StatusCode)}
	} else if resp.StatusCode < 200 && resp.StatusCode >= 300 {
		return 0, &httpError{canRetry: false, message: fmt.Sprintf("Server replied with %d", resp.StatusCode)}
	}

	pw.progress.Total = existingSize + getContentLength(resp)

	// Copy data from the HTTP request to the file.
	written, err = io.Copy(file, io.TeeReader(resp.Body, pw))
	if err != nil {
		_ = file.Close()
		// Sometimes I see random "stream error: stream ID x; INTERNAL_ERROR" from
		// certain sites.  If we see these, especially if the server supports resume,
		// we should retry.
		httpErr = &httpError{canRetry: true, message: fmt.Sprintf("Error downloading %s: %v", request.URL.String(), err)}
		return written, httpErr
	}

	if err = file.Close(); err != nil {
		httpErr = &httpError{message: fmt.Sprintf("Error closing %s: %v", filename+partialSuffix, err)}
		return written, httpErr
	}

	// Move the file to the final destination.
	if err = os.Rename(filename+partialSuffix, filename); err != nil {
		httpErr = &httpError{
			message: fmt.Sprintf("Error renaming %s to %s: %v", filename+partialSuffix, filename, err),
		}
		return written, httpErr
	}

	// Set the modified time of the file to match the one on the server.
	if remoteInfo.LastModified != nil {
		_ = os.Chtimes(filename, time.Now(), *remoteInfo.LastModified)
	}

	return written, nil
}

func (client *DownloadClient) resumeDownload(request *http.Request, start int64, end int64) (*http.Response, error) {
	req := request.Clone(request.Context())
	req.Header.Add("Range", fmt.Sprintf("bytes=%d-%d", start, end))
	return client.httpClient.Do(req)
}
