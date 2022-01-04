package providers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"

	"github.com/jwalton/pixdl/pkg/pixdl/meta"
)

type gofileUpload struct {
	// Status is the status of the request - should be "ok".
	Status string `json:"status"`
	// Data is the data for the request.
	Data struct {
		// Code is the unique ID for this album.
		Code string `json:"code"`
		// CreateTime is the unix timestamp this file was created at (e.g. 1618941563)
		CreateTime int64 `json:"createTime"`
		// TotalDownload is the number of times this album has been downloaded.
		TotalDownload int64 `json:"totalDownloadCount"`
		// TotalSize is the total size of all items in this album, in bytes.
		TotalSize int64 `json:"totalSize"`
		// Files is a hash of all files in this album, indexed by md5 hash.
		Files map[string]gofileFile `json:"contents"`
	} `json:"data"`
}

type gofileFile struct {
	// Name is the filename for this file.
	Name string `json:"name"`
	// Size is the size of this file, in bytes.
	Size int64 `json:"size"`
	// Mimetype is the MIME type for this file.
	Mimetype string `json:"mimetype"`
	// Link is the URL to download this file from.
	Link string `json:"link"`
}

type gofileProvider struct{}

var gofileRegex = regexp.MustCompile(`^(https://)?gofile.io/d/(\w*)/?$`)

func (gofileProvider) Name() string {
	return "gofile.io"
}

// CanDownload returns true if this downloader can download an album from the given URL.
func (gofileProvider) CanDownload(url string) bool {
	return gofileRegex.MatchString(url)
}

func (gofileProvider) gofileAPIRequest(env *Env, apiURL string) (*http.Response, error) {
	req, err := env.NewGetRequest(apiURL)
	if err != nil {
		return nil, fmt.Errorf("unable to create request for: %s: %v", apiURL, err)
	}

	req.Header.Add("Accept", "*/*")
	req.Header.Add("Origin", "https://gofile.io")
	req.Header.Add("Referer", "https://gofile.io/")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		resp.Body.Close()
		return nil, fmt.Errorf("%s returned %v", apiURL, resp.StatusCode)
	}
	return resp, nil
}

func (provider gofileProvider) FetchAlbum(env *Env, params map[string]string, url string, callback ImageCallback) {
	match := gofileRegex.FindStringSubmatch(url)
	if match == nil {
		callback(nil, nil, fmt.Errorf("invalid gofile album: %s", url))
		return
	}

	albumID := match[2]

	// apiURL := "https://api.gofile.io/getFolder?folderId=" + albumID
	apiURL := "https://api.gofile.io/getContent?contentId=" + albumID + "&token=" + params["gofile.token"]
	resp, err := provider.gofileAPIRequest(env, apiURL)
	if err != nil {
		callback(nil, nil, fmt.Errorf("unable to create request for: %s: %v", url, err))
		return
	}
	defer resp.Body.Close()

	provider.parseAlbum(url, albumID, resp.Body, callback)
}

func (provider gofileProvider) parseAlbum(url string, albumID string, reader io.Reader, callback ImageCallback) {
	albumData := gofileUpload{}

	err := json.NewDecoder(reader).Decode(&albumData)
	if err != nil {
		callback(&meta.AlbumMetadata{URL: url, AlbumID: albumID}, nil, err)
	}

	album := &meta.AlbumMetadata{
		Provider:        "gofile.io",
		URL:             url,
		AlbumID:         albumID,
		Name:            albumID,
		TotalImageCount: len(albumData.Data.Files),
	}

	files := provider.sortFiles(albumData.Data.Files)

	provider.parseImages(album, files, callback)
}

func (provider gofileProvider) sortFiles(fileMap map[string]gofileFile) []gofileFile {
	files := make([]gofileFile, 0, len(fileMap))
	for _, image := range fileMap {
		files = append(files, image)
	}
	sort.Slice(files, func(i int, j int) bool {
		return files[i].Name < files[j].Name
	})
	return files
}

func (provider gofileProvider) parseImages(album *meta.AlbumMetadata, images []gofileFile, callback ImageCallback) {
	for index, image := range images {
		callback(
			album,
			&meta.ImageMetadata{
				Album:    album,
				URL:      image.Link,
				Filename: image.Name,
				Title:    image.Name,
				Size:     image.Size,
				Index:    index,
				Page:     1,
			},
			nil,
		)
	}

	callback(album, nil, nil)
}
