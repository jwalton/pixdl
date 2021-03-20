package pixdl

import (
	"github.com/jwalton/pixdl/pkg/providers"
	"github.com/jwalton/pixdl/pkg/providers/types"
)

type ImageCallback = types.ImageCallback

func GetAlbum(url string, callback ImageCallback) {
	defaultAlbum := &AlbumMetadata{URL: url}

	provider, err := providers.GetProviderForURL(url)
	if err != nil {
		callback(defaultAlbum, nil, err)
		return
	}

	provider.FetchAlbum(
		url,
		func(album *AlbumMetadata, image *ImageMetadata, err error) bool {
			// If the provider doesn't give us an album, use the default one we created.
			if album == nil {
				album = defaultAlbum
			}
			return callback(album, image, err)
		},
	)
}

func downloadAlbum(downloader ImageDownloader, url string, toFolder string, reporter ProgressReporter) {
	started := false

	reporter.AlbumFetch(url)
	GetAlbum(url, func(album *AlbumMetadata, image *ImageMetadata, err error) bool {
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

		downloader.DownloadImage(image, toFolder, reporter)

		return true
	})
}
