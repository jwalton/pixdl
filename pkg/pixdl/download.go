package pixdl

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/jwalton/pixdl/pkg/download"
	"github.com/jwalton/pixdl/pkg/pixdl/meta"
)

type AlbumMetadata = meta.AlbumMetadata
type ImageMetadata = meta.ImageMetadata

var NewImageMetadata = meta.NewImageMetadata

var client = download.NewDownloadClient()

// Return the file name to store the downloaded image in.
func getDownloadFilename(image *ImageMetadata, remoteInfo *download.RemoteFileInfo) (string, error) {
	filename := image.Filename

	if filename == "" {
		filename = remoteInfo.Filename
	}

	if filename == "" {
		u, err := url.Parse(image.URL)
		if err != nil {
			return filename, fmt.Errorf("Error parsing URL: %w", err)
		}
		filename = path.Base(u.Path)
	}

	if filename == "" {
		// TODO: Do a HEAD request for the file, to figure out the filename
		// from content-disposition.  We could let `grab` do this for us, but
		// then there's no easy way to make this a ".part" file.  `grab` gives
		// us no easy way to edit the filename.  Should maybe HEAD the file
		// anyways as an easy way to get the file size.
		return filename, fmt.Errorf("Could not determine name for file.")
	}

	return filename, nil
}

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
	image *ImageMetadata,
	toFolder string,
	minSizeBytes int64,
	reporter ProgressReporter,
) error {
	var err error

	if image == nil {
		panic("pixdl.DownloadImage requires an image")
	}

	albumMetadata := image.Album

	req, err := http.NewRequest("GET", image.URL, nil)
	if err != nil {
		reporter.ImageSkip(image, err)
		return err
	}

	req.Header.Set("User-Agent", "pixdl")

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
		return err
	}
	destFilename := filepath.Join(toFolder, basename)

	// Make sure the destination directory exists.
	destDir := filepath.Dir(destFilename)
	os.MkdirAll(destDir, 0755)

	// Verify image doesn't already exist before downloading
	exists, err := fileExists(destFilename)
	if err != nil || exists {
		// If the already exists, or we can't check for some reason, skip it.
		if reporter != nil {
			reporter.ImageSkip(image, err)
		}
		return err
	}

	// If the image is beneath our minimum size threshold, skip it.
	if minSizeBytes > 0 {
		if (remoteInfo.Size > -1 && remoteInfo.Size < minSizeBytes) ||
			(image.Size != -1 && image.Size < minSizeBytes) {
			reporter.ImageSkip(image, nil)
			return nil
		}
	}

	if reporter != nil {
		reporter.ImageStart(image)
		defer func() { reporter.ImageEnd(image, err) }()
	}

	// Get the file...
	_, err = client.DoWithFileInfo(req, destFilename, remoteInfo, newDownloadProgressWrapper(reporter, albumMetadata, image))
	if err != nil {
		return err
	}

	// Update modified time, if the image has a timestamp.
	// If this fails, ignore the error.
	if image.Timestamp != nil {
		_ = os.Chtimes(destFilename, time.Now(), *image.Timestamp)
	}

	return nil
}
