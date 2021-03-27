package pixdl

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/jwalton/pixdl/pkg/download"
	"github.com/jwalton/pixdl/pkg/pixdl/meta"
	"github.com/jwalton/pixdl/pkg/providers/env"
)

// AlbumMetadata contains data about an album.
type AlbumMetadata = meta.AlbumMetadata

// ImageMetadata contains data about an image inside an album.
type ImageMetadata = meta.ImageMetadata

var client = download.NewClient()

// Return the file name to store the downloaded image in.
func getDownloadFilename(image *ImageMetadata, remoteInfo *download.RemoteFileInfo) (string, error) {
	filename := image.Filename

	if filename == "" {
		filename = remoteInfo.Filename
	}

	if filename == "" {
		u, err := url.Parse(image.URL)
		if err != nil {
			return filename, fmt.Errorf("error parsing URL: %w", err)
		}
		filename = path.Base(u.Path)
	}

	if filename == "" {
		return filename, fmt.Errorf("could not determine name for file")
	}

	return filename, nil
}

// fileExists returns true if the local file already exists, false otherwise.
func fileExists(filename string) (bool, error) {
	// Verify image doesn't already exist before downloading
	_, err := os.Stat(filename)
	if err != nil && errors.Is(err, os.ErrNotExist) {
		// File does not exist.
		return false, nil
	} else if err != nil {
		// Couldn't figure out if the file exists...
		return false, err
	} else {
		return true, nil
	}
}

// downloadImage downloads an image and saves it on disk.
// `image` is the image to download, `toFolder` is the file to store it in.
func downloadImage(
	env *env.Env,
	image *ImageMetadata,
	toFolder string,
	minSizeBytes int64,
	reporter ProgressReporter,
) {
	var err error

	if image == nil {
		panic("pixdl.DownloadImage requires an image")
	}

	albumMetadata := image.Album

	req, err := env.NewGetRequest(image.URL)
	if err != nil {
		reporter.ImageSkip(image, err)
		return
	}

	// TODO: Allow the provider pass back a set of headers with each image.
	// This can handle things like sites that need a referrer, or sites
	// that need authentication.
	// req.Header.Set("User-Agent", "pixdl")

	remoteInfo := image.RemoteInfo
	if remoteInfo == nil {
		remoteInfo, _ = client.DoFileInfo(req)
	}

	// Figure out where to store this image
	basename, err := getDownloadFilename(image, remoteInfo)
	if err != nil {
		if reporter != nil {
			reporter.ImageSkip(image, err)
		}
		return
	}
	destFilename := filepath.Join(toFolder, basename)

	// Make sure the destination directory exists.
	destDir := filepath.Dir(destFilename)
	err = os.MkdirAll(destDir, 0755)
	if err != nil {
		if reporter != nil {
			reporter.ImageSkip(image, err)
		}
		return
	}

	// Verify image doesn't already exist before downloading
	exists, err := fileExists(destFilename)
	if err != nil || exists {
		// If the already exists, or we can't check for some reason, skip it.
		if reporter != nil {
			reporter.ImageSkip(image, err)
		}
		return
	}

	// If the image is beneath our minimum size threshold, skip it.
	if minSizeBytes > 0 {
		if (remoteInfo.Size > -1 && remoteInfo.Size < minSizeBytes) ||
			(image.Size != -1 && image.Size < minSizeBytes) {
			reporter.ImageSkip(image, nil)
			return
		}
	}

	if reporter != nil {
		reporter.ImageStart(image)
		defer func() { reporter.ImageEnd(image, err) }()
	}

	// Get the file...
	_, err = client.DoWithFileInfo(req, destFilename, remoteInfo, newDownloadProgressWrapper(reporter, albumMetadata, image))
	if err != nil {
		return
	}

	// Update modified time, if the image has a timestamp.
	// If this fails, ignore the error.
	if image.Timestamp != nil {
		_ = os.Chtimes(destFilename, time.Now(), *image.Timestamp)
	}
}
