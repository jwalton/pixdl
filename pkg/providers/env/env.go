// Package env provides a common "environment" object with utility functions
// and settings information for all providers.
package env

import (
	"net/http"

	"github.com/jwalton/pixdl/pkg/download"
)

type Env struct {
	DownloadClient *download.DownloadClient
}

// NewGetRequest creates a new http GET request.
func (*Env) NewGetRequest(url string) (*http.Request, error) {
	req, err := http.NewRequest("GET", url, nil)

	req.Header.Set("User-Agent", "pixdl")

	return req, err
}

// Get will fetch the contents of a URL via HTTP GET.
func (env *Env) Get(url string) (*http.Response, error) {
	req, err := env.NewGetRequest(url)
	if err != nil {
		return nil, err
	}

	return http.DefaultClient.Do(req)
}

func (env *Env) GetFileInfo(url string) (*download.RemoteFileInfo, error) {
	req, err := env.NewGetRequest(url)
	if err != nil {
		return nil, err
	}

	return env.DownloadClient.DoFileInfo(req)
}
