package imgur

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"time"

	"github.com/jwalton/pixdl/pkg/pixdl/meta"
	"github.com/jwalton/pixdl/pkg/providers/env"
	"github.com/jwalton/pixdl/pkg/providers/types"
)

// Provider returns a new Imgur provider.
func Provider() types.URLProvider {
	return imgurProvider{}
}

type imgurProvider struct{}

var imgurRegex = regexp.MustCompile(`^(https://)?imgur.com/gallery/(\w*)/?$`)

func (imgurProvider) Name() string {
	return "imgur"
}

// CanDownload returns true if this downloader can download an album from the given URL.
func (imgurProvider) CanDownload(url string) bool {
	return imgurRegex.MatchString(url)
}

func (imgurProvider) FetchAlbum(env *env.Env, params map[string]string, url string, callback types.ImageCallback) {
	match := imgurRegex.FindStringSubmatch(url)
	if match == nil {
		callback(nil, nil, fmt.Errorf("invalid imgur album: %s", url))
		return
	}

	albumID := match[2]

	resp, err := env.Get("https://imgur.com/gallery/" + albumID + ".json")
	if err != nil {
		callback(nil, nil, fmt.Errorf("unable to fetch album: %s: %v", url, err))
		return
	}

	defer resp.Body.Close()

	parseAlbum(url, albumID, resp.Body, callback)
}

func parseAlbum(url string, albumID string, reader io.Reader, callback types.ImageCallback) {
	albumData := imgurGalleryResponse{}

	err := json.NewDecoder(reader).Decode(&albumData)
	if err != nil {
		callback(&meta.AlbumMetadata{URL: url, AlbumID: albumID}, nil, err)
	}

	album := &meta.AlbumMetadata{
		Provider: "imgur",
		URL:      url,
		AlbumID:  albumID,
		Name:     albumData.Data.Image.Title,
		Author:   albumData.Data.Image.AccountURL,
		// TotalImageCount will be -1 if the total image count is unknown.
		TotalImageCount: int(albumData.Data.Image.AlbumImages.Count),
	}

	parseImages(album, albumData.Data.Image.AlbumImages.Images, callback)
}

func parseImages(album *meta.AlbumMetadata, images []imgurImage, callback types.ImageCallback) {
	for index, image := range images {
		var timestamp *time.Time

		if time, err := time.Parse("2006-01-02 15:04:05", image.Datetime); err == nil {
			timestamp = &time
		}

		var filename string
		if image.Name != "" {
			filename = image.Name + image.Ext
		} else {
			filename = image.Hash + image.Ext
		}

		callback(
			album,
			&meta.ImageMetadata{
				Album:     album,
				URL:       "https://i.imgur.com/" + image.Hash + image.Ext,
				Filename:  filename,
				Title:     image.Title,
				Size:      image.Size,
				Timestamp: timestamp,
				Index:     index,
				Page:      1,
			},
			nil,
		)
	}

	callback(album, nil, nil)
}
