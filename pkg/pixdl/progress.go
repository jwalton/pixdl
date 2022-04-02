package pixdl

import (
	"github.com/jwalton/pixdl/pkg/download"
)

// ProgressReporter is an interface for receiving progress updates from pixdl.
type ProgressReporter interface {
	AlbumFetch(url string)
	AlbumStart(album *AlbumMetadata)
	AlbumEnd(album *AlbumMetadata, err error)
	ImageSkip(image *ImageMetadata, err error)
	ImageStart(image *ImageMetadata)
	ImageProgress(image *ImageMetadata, progress *download.Progress)
	ImageEnd(image *ImageMetadata, err error)
	Done()
}

// Returns a new download.ProgressReporter which forwards data to the passed in pixdl.ProgressReporter.
func newDownloadProgressWrapper(
	reporter ProgressReporter,
	album *AlbumMetadata,
	image *ImageMetadata,
) download.FileProgressCallback {
	return func(progress *download.Progress) {
		if reporter != nil {
			reporter.ImageProgress(image, progress)
		}
	}
}
