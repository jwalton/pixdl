package providers

import (
	"net/http"

	"github.com/jwalton/pixdl/pkg/download"
	"golang.org/x/net/html"
)

// Env is a common "environment" object with utility functions and settings
// information that is passed to all providers.
type Env struct {
	// DownloadClient is the client that wil be used to download files.
	// This must be provided.
	DownloadClient *download.Client
}

// NewGetRequest creates a new http GET request.
func (*Env) NewGetRequest(url string) (*http.Request, error) {
	req, err := http.NewRequest("GET", url, nil)

	req.Header.Set("User-Agent", "pixdl")

	return req, err
}

// Get will fetch the contents of a URL via HTTP GET.
func (env *Env) Get(url string) (*http.Response, error) {
	return env.GetWithReferer(url, "")
}

// Get will fetch the contents of a URL via HTTP GET.
func (env *Env) GetWithReferer(url string, referer string) (*http.Response, error) {
	req, err := env.NewGetRequest(url)
	if err != nil {
		return nil, err
	}

	if referer != "" {
		req.Header.Set("Referer", referer)
	}

	return http.DefaultClient.Do(req)
}

// GetHTML will fetch the HTML contents of a URL via HTTP GET, and return the parsed HTML.
func (env *Env) GetHTML(url string) (*html.Node, error) {
	resp, err := env.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	return html.Parse(resp.Body)
}

// GetFileInfo returns information about a file on a server.
func (env *Env) GetFileInfo(url string) (*download.RemoteFileInfo, error) {
	return env.GetFileInfoWithReferer(url, "")
}

// GetFileInfo returns information about a file on a server.
func (env *Env) GetFileInfoWithReferer(url string, referer string) (*download.RemoteFileInfo, error) {
	req, err := env.NewGetRequest(url)
	if err != nil {
		return nil, err
	}

	if referer != "" {
		req.Header.Set("Referer", referer)
	}

	return env.DownloadClient.DoFileInfo(req)
}
