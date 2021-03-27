package meta

import (
	"time"

	"github.com/jwalton/pixdl/pkg/download"
)

// AlbumMetadata contains data about an album.
type AlbumMetadata struct {
	URL     string
	AlbumID string
	Name    string
	Author  string
	// TotalImageCount will be -1 if the total image count is unknown.
	TotalImageCount int
}

// ImageMetadata contains data about an image inside an album.
type ImageMetadata struct {
	// Album is the album this image came from
	Album *AlbumMetadata
	// URL is the URL to download this image from.
	URL string
	// Filename for this image.  If left blank, we'll generate a filename
	// from the URL.
	Filename string
	// Title is the title of this image, if available.
	Title string
	// Size is the length of the image in bytes, or -1 if unknown.
	Size int64
	// Timestamp is the creation time of this image, or nil if unknown.
	Timestamp *time.Time
	// Index of this image within the album
	Index int
	// Page is the page number (1 based) this image was on.
	Page int
	// RemoteInfo is information about this file, obtained from DownloadClient.GetFileInfo().
	// This is optional - you only need to provide it when creating an image if
	// you already have it, so download doesn't need to get it again.
	RemoteInfo *download.RemoteFileInfo
}

func NewImageMetadata(album *AlbumMetadata, index int) *ImageMetadata {
	return &ImageMetadata{
		Album: album,
		Size:  -1,
		Index: index,
	}
}

// DownloadOptions represents options about which files to download.
type DownloadOptions struct {
	// The minimum size, in bytes, to download an image.  0 for any size.
	MinSize uint64
}
