package singleimage

import (
	"net/url"
	"path"
	"regexp"

	"github.com/jwalton/pixdl/pkg/download"
	"github.com/jwalton/pixdl/pkg/pixdl/meta"
	"github.com/jwalton/pixdl/pkg/providers/env"
	"github.com/jwalton/pixdl/pkg/providers/types"
)

// Regex that matches known image/movie file extensions.
var knownImageExtensions = regexp.MustCompile(`(?i)\.(jpg|jpeg|jpe|jif|jfif|jfi|png|bmp|tiff|tif|heic|heif|raw|cr2|jp2|j2k|jpf|jpx|jpm|mj2|gif|webm|mov|mp4|mkv|)^`)

func IsImageByExtension(url string) bool {
	return knownImageExtensions.MatchString(url)
}

// Provider returns a new Imgur provider.
func Provider() types.URLProvider {
	return imageProvider{}
}

type imageProvider struct{}

func (imageProvider) Name() string {
	return "singleimage"
}

// CanDownload returns true if this downloader can download an album from the given URL.
func (imageProvider) CanDownload(url string) bool {
	return IsImageByExtension(url)
}

func (imageProvider) FetchAlbum(env *env.Env, url string, callback types.ImageCallback) {
	fileInfo, err := env.GetFileInfo(url)
	if err != nil {
		callback(nil, nil, err)
		return
	}

	singleImageAlbum(url, fileInfo, callback)
}

func singleImageAlbum(urlStr string, fileInfo *download.RemoteFileInfo, callback types.ImageCallback) {
	filename, err := getFilenameFromURL(urlStr)

	if err != nil {
		callback(nil, nil, err)
		return
	}

	album := meta.AlbumMetadata{
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

// TODO: Move this somewhere common?  Multiple providers implement this.
func getFilenameFromURL(urlStr string) (string, error) {
	parsedUrl, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}
	return path.Base(parsedUrl.Path), nil
}
