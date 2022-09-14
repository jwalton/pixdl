package providers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/jwalton/pixdl/pkg/pixdl/meta"
)

const imgurClientID = "7c8b3491d6207b4"

type imgurGalleryResponse struct {
	ID          string `json:"id"`
	Title       string
	Description string
	ImageCount  int64        `json:"image_count"`
	AlbumURL    string       `json:"url"`
	Media       []imgurImage `json:"media"`
}

type imgurImage struct {
	// ID is a unique ID.  Can download the actual image from `https://i.imgur.com/${Hash}.${Ext}`.
	ID string `json:"id"`
	// Name is the original name of the file, with extension.
	Name string `json:"name"`
	// URL is the URL to download the image from.
	URL string `json:"url"`
	// Width is the width of the image
	Width int64 `json:"width"`
	// Height is the height of the image
	Height int64 `json:"height"`
	// Size is the size of the image, in bytes.
	Size int64 `json:"size"`
	// Ext is the extension of the file.
	Ext string `json:"ext"`
	// CreatedAt is in format "2017-07-31T12:25:20Z".
	CreatedAt string `json:"created_at"`
}

type imgurProvider struct{}

var imgurRegex = regexp.MustCompile(`^(https://)?imgur.com/(?:gallery|a)/(\w*)/?$`)

func (imgurProvider) Name() string {
	return "imgur"
}

// CanDownload returns true if this downloader can download an album from the given URL.
func (imgurProvider) CanDownload(url string) bool {
	return imgurRegex.MatchString(url)
}

func (provider imgurProvider) Get(env *Env, url string) (*http.Response, error) {
	req, err := env.NewGetRequest(url)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Client-ID "+imgurClientID)
	return http.DefaultClient.Do(req)
}

func (provider imgurProvider) FetchAlbum(env *Env, params map[string]string, url string, callback ImageCallback) {
	match := imgurRegex.FindStringSubmatch(url)
	if match == nil {
		callback(nil, nil, fmt.Errorf("invalid imgur album: %s", url))
		return
	}

	albumID := match[2]

	resp, err := provider.Get(env, "https://api.imgur.com/post/v1/albums/"+albumID+"?include=media")
	if err != nil {
		callback(nil, nil, fmt.Errorf("unable to fetch album: %s: %v", url, err))
		return
	}
	if resp.StatusCode != 200 {
		callback(nil, nil, fmt.Errorf("unexpected response from server: %d", resp.StatusCode))
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
		Name:     albumData.Title,
		// TotalImageCount will be -1 if the total image count is unknown.
		TotalImageCount: int(albumData.ImageCount),
	}

	provider.parseImages(album, albumData.Media, callback)
}

func (provider imgurProvider) parseImages(album *meta.AlbumMetadata, images []imgurImage, callback ImageCallback) {
	for index, image := range images {
		var timestamp *time.Time

		if time, err := time.Parse("2006-01-02T15:04:05Z", image.CreatedAt); err == nil {
			timestamp = &time
		}

		var filename string
		if image.Name != "" {
			filename = image.Name
			if image.Ext != "" && !strings.HasSuffix(filename, image.Ext) {
				filename = filename + "." + image.Ext
			}
		} else {
			filename = image.ID + image.Ext
		}

		callback(
			album,
			&meta.ImageMetadata{
				Album:     album,
				URL:       image.URL,
				Filename:  filename,
				Title:     image.Name,
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
