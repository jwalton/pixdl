package pixdl

import (
	"fmt"
	"strings"

	"github.com/jwalton/pixdl/pkg/providers/env"
	"github.com/jwalton/pixdl/pkg/providers/singleimage"
	"github.com/jwalton/pixdl/pkg/providers/types"
)

type ImageCallback = types.ImageCallback

// getAlbum will fetch all images in an album, and pass each one to the callback.
// If an error occurs fetching images, the callback will be called with an error.
// When all images have been fetched, callback will be called with `nil` for
// the image.  The callback may return false to stop fetching more images.
func getAlbum(env *env.Env, url string, callback ImageCallback) {
	defaultAlbum := &AlbumMetadata{URL: url}

	handled := getAlbumByURL(env, url, callback)

	if !handled {
		var err error

		// Figure out what kind of resource this is.
		req, err := env.NewGetRequest(url)
		if err != nil {
			callback(defaultAlbum, nil, err)
			return
		}

		fileInfo, err := client.DoFileInfo(req)
		if err != nil {
			callback(defaultAlbum, nil, err)
			return
		}

		if fileInfo.MimeType == "text/html" {
			handled, err = getAlbumWithHTML(env, url, callback)

			if err != nil {
				callback(defaultAlbum, nil, err)
				return
			}
		} else if strings.HasPrefix(fileInfo.MimeType, "image/") {
			// If the URL is an image, use the "singleimage" provider to download it.
			provider := singleimage.Provider()
			provider.FetchAlbum(env, url, callback)
		}
	}

	if !handled {
		callback(defaultAlbum, nil, fmt.Errorf("Could not find a suitable provider to download album"))
	}
}

// downloadAlbum will fetch every image in an album and then download it, using
// the specified downloader.
func downloadAlbum(downloader ImageDownloader, url string, options DownloadOptions, reporter ProgressReporter) {
	started := false
	startPage := -1

	reporter.AlbumFetch(url)
	getAlbum(downloader.getEnv(), url, func(album *AlbumMetadata, image *ImageMetadata, err error) bool {
		if !started {
			reporter.AlbumStart(album)
			started = true

			if image == nil || err != nil {
				// Never got a first image - end right away
				reporter.AlbumEnd(album, err)
				return false
			}
		} else if image == nil {
			// All done!
			reporter.AlbumEnd(album, err)
			return false
		}

		if startPage == -1 {
			startPage = image.Page
		}

		if options.MaxPages > 0 && (image.Page-startPage) >= options.MaxPages {
			// Stop fetching images
			return false
		}

		downloader.DownloadImage(image, options.ToFolder, reporter)

		return true
	})
}
