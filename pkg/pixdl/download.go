package pixdl

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"text/template"
	"time"

	"github.com/jwalton/pixdl/pkg/download"
	"github.com/jwalton/pixdl/pkg/pixdl/meta"
	"github.com/jwalton/pixdl/pkg/providers"
)

// AlbumMetadata contains data about an album.
type AlbumMetadata = meta.AlbumMetadata

// ImageMetadata contains data about an image inside an album.
type ImageMetadata = meta.ImageMetadata

var client = download.NewClient()

// Return the file name to store the downloaded image in.
func getDownloadFilename(
	image *ImageMetadata,
	getRemoteInfo func() (*download.RemoteFileInfo, error),
) (string, error) {
	filename := image.Filename

	if filename == "" {
		remoteInfo, err := getRemoteInfo()
		if err != nil {
			return "", err
		}
		if remoteInfo != nil {
			filename = remoteInfo.Filename
		}
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
	env *providers.Env,
	image *ImageMetadata,
	toFolder string,
	filenameTemplate string,
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
		if reporter != nil {
			reporter.ImageSkip(image, err)
		}
		return
	}

	// TODO: Allow the provider pass back a set of headers with each image.
	// This can handle things like sites that need a referrer, or sites
	// that need authentication.
	// req.Header.Set("User-Agent", "pixdl")

	// Defer getting remoteInfo until we know we need it.  If we already have the
	// filename, and the file exists locally, we can save ourselves and HTTP request.
	cachedRemoteInfo := image.RemoteInfo
	getRemoteInfo := func() (*download.RemoteFileInfo, error) {
		if cachedRemoteInfo == nil {
			cachedRemoteInfo, _ = client.DoFileInfo(req)
		}
		return cachedRemoteInfo, nil
	}

	// Figure out where to store this image
	downloadFilename, err := getDownloadFilename(image, getRemoteInfo)
	if err != nil {
		if reporter != nil {
			reporter.ImageSkip(image, err)
		}
		return
	}

	templateFilename, err := getTemplateFilename(filenameTemplate, downloadFilename, albumMetadata, image)
	if err != nil {
		if reporter != nil {
			reporter.ImageSkip(image, err)
		}
		return
	}

	destFilename := filepath.Join(toFolder, templateFilename)

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

	remoteInfo, err := getRemoteInfo()
	if err != nil {
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

func validateTemplate(filenameTemplate string) error {
	if filenameTemplate == "" {
		return nil
	}
	_, err := template.New("filename").Parse(filenameTemplate)
	return err
}

func getTemplateFilename(
	filenameTemplate string,
	downloadFilename string,
	albumMetadata *AlbumMetadata,
	imageMetadata *ImageMetadata,
) (string, error) {
	if filenameTemplate == "" {
		return downloadFilename, nil
	}

	template, err := template.New("filename").Parse(filenameTemplate)
	if err != nil {
		return downloadFilename, err
	}

	var b bytes.Buffer
	err = template.Execute(&b, map[string]interface{}{
		"Filename": downloadFilename,
		"Album":    albumMetadata,
		"Image":    imageMetadata,
	})
	if err != nil {
		return downloadFilename, err
	}

	return b.String(), nil
}
