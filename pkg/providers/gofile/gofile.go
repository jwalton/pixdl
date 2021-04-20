package gofile

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"

	"github.com/jwalton/pixdl/pkg/pixdl/meta"
	"github.com/jwalton/pixdl/pkg/providers/env"
	"github.com/jwalton/pixdl/pkg/providers/types"
)

// Provider returns a new Gofile provider.
func Provider() types.URLProvider {
	return gofileProvider{}
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

func (gofileProvider) FetchAlbum(env *env.Env, url string, callback types.ImageCallback) {
	match := gofileRegex.FindStringSubmatch(url)
	if match == nil {
		callback(nil, nil, fmt.Errorf("invalid gofile album: %s", url))
		return
	}

	albumID := match[2]

	resp, err := env.Get("https://api.gofile.io/getUpload?c=" + albumID)
	if err != nil {
		callback(nil, nil, fmt.Errorf("unable to fetch album: %s: %v", url, err))
		return
	}

	defer resp.Body.Close()

	parseAlbum(url, albumID, resp.Body, callback)
}

func parseAlbum(url string, albumID string, reader io.Reader, callback types.ImageCallback) {
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

	files := sortFiles(albumData.Data.Files)

	parseImages(album, files, callback)
}

func sortFiles(fileMap map[string]gofileFile) []gofileFile {
	files := make([]gofileFile, 0, len(fileMap))
	for _, image := range fileMap {
		files = append(files, image)
	}
	sort.Slice(files, func(i int, j int) bool {
		return files[i].Name < files[j].Name
	})
	return files
}

func parseImages(album *meta.AlbumMetadata, images []gofileFile, callback types.ImageCallback) {
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
