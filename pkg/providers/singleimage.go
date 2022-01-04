package providers

import (
	"net/url"
	"path"
	"regexp"

	"github.com/jwalton/pixdl/pkg/download"
	"github.com/jwalton/pixdl/pkg/pixdl/meta"
)

// Regex that matches known image/movie file extensions.
var knownImageExtensions = regexp.MustCompile(`(?i)\.(jpg|jpeg|jpe|jif|jfif|jfi|png|bmp|tiff|tif|heic|heif|raw|cr2|jp2|j2k|jpf|jpx|jpm|mj2|gif|webm|mov|mp4|mkv|)^`)

// IsImageByExtension returns true if the given URL appears to point to an image, based on the file extension.
func IsImageByExtension(url string) bool {
	return knownImageExtensions.MatchString(url)
}

// SingleImageProvider returns a new instance of the singleimage provider.
func SingleImageProvider() URLProvider {
	return singleimageProvider{}
}

type singleimageProvider struct{}

func (singleimageProvider) Name() string {
	return "singleimage"
}

// CanDownload returns true if this downloader can download an album from the given URL.
func (singleimageProvider) CanDownload(url string) bool {
	return IsImageByExtension(url)
}

func (singleimageProvider) FetchAlbum(env *Env, params map[string]string, url string, callback ImageCallback) {
	fileInfo, err := env.GetFileInfo(url)
	if err != nil {
		callback(nil, nil, err)
		return
	}

	singleImageAlbum(url, fileInfo, callback)
}

func singleImageAlbum(urlStr string, fileInfo *download.RemoteFileInfo, callback ImageCallback) {
	filename, err := getFilenameFromURL(urlStr)

	if err != nil {
		callback(nil, nil, err)
		return
	}

	album := meta.AlbumMetadata{
		Provider:        "singleimage",
		URL:             urlStr,
		AlbumID:         "",
		TotalImageCount: 1,
		Name:            filename,
	}

	image := meta.ImageMetadata{
		URL:        urlStr,
		Album:      &album,
		Filename:   filename,
		Title:      filename,
		Size:       fileInfo.Size,
		RemoteInfo: fileInfo,
		Timestamp:  fileInfo.LastModified,
		Index:      0,
		Page:       1,
	}

	cont := callback(&album, &image, nil)
	if cont {
		callback(&album, nil, nil)
	}
}

func getFilenameFromURL(urlStr string) (string, error) {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}
	return path.Base(parsedURL.Path), nil
}
