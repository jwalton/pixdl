package providers

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"time"

	"github.com/jwalton/pixdl/pkg/pixdl/meta"
)

type imgurGalleryResponse struct {
	Data struct {
		Image struct {
			ID          int64  `json:"id"`
			Hash        string `json:"hash"`
			AccountURL  string `json:"account_url"`
			Title       string `json:"title"`
			AlbumImages struct {
				Count  int64        `json:"count"`
				Images []imgurImage `json:"images"`
			} `json:"album_images"`
		} `jason:"image"`
	} `json:"data"`
}

type imgurImage struct {
	// Hash is a unique ID.  Can download the actual image from `https://i.imgur.com/${Hash}.${Ext}`.
	Hash string `json:"hash"`
	// Title is the title of the file, but is often empty.
	Title string `json:"title"`
	// Size is the size of the image, in bytes.
	Size int64 `json:"size"`
	// Description is a description of the file.
	Description string `json:"description"`
	// Name is the name of the file, without the extension.
	Name string `json:"name"`
	// Ext is the extension of the file.
	Ext string `json:"ext"`
	// Datetime is in format "2017-07-31 12:25:20".
	Datetime string `json:"datetime"`
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

func (provider imgurProvider) FetchAlbum(env *Env, params map[string]string, url string, callback ImageCallback) {
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

	provider.parseAlbum(url, albumID, resp.Body, callback)
}

func (provider imgurProvider) parseAlbum(url string, albumID string, reader io.Reader, callback ImageCallback) {
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

	provider.parseImages(album, albumData.Data.Image.AlbumImages.Images, callback)
}

func (provider imgurProvider) parseImages(album *meta.AlbumMetadata, images []imgurImage, callback ImageCallback) {
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
