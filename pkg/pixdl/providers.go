package pixdl

import (
	"github.com/jwalton/pixdl/pkg/providers"
	"github.com/jwalton/pixdl/pkg/providers/env"
	"github.com/jwalton/pixdl/pkg/providers/types"
	"golang.org/x/net/html"
)

type URLProvider = types.URLProvider
type HTMLProvider = types.HTMLProvider

// getAlbumByURL tries to download the album using a URLProvider.  If a suitable
// provider is found, this will return true.
func getAlbumByURL(env *env.Env, url string, callback ImageCallback) bool {
	defaultAlbum := &AlbumMetadata{URL: url}

	for _, provider := range providers.UrlProviderRegistry {
		if provider.CanDownload(url) {
			provider.FetchAlbum(
				env,
				url,
				func(album *AlbumMetadata, image *ImageMetadata, err error) bool {
					// If the provider doesn't give us an album, use the default one we created.
					if album == nil {
						album = defaultAlbum
					}
					return callback(album, image, err)
				},
			)
			return true
		}
	}

	return false
}

// getAlbumWithHTML will download the HTML for an album and parse it, then pass
// it to each HTMLProvider.  If an HTMLProvider claims to be able to download
// the album, this will return true, false otherwise.
func getAlbumWithHTML(env *env.Env, url string, callback ImageCallback) (bool, error) {
	resp, err := env.Get(url)
	if err != nil {
		return false, err
	}

	defer resp.Body.Close()

	node, err := html.Parse(resp.Body)
	if err != nil {
		return false, err
	}

	for _, provider := range providers.HtmlProviderRegistry {
		if provider.FetchAlbumFromHTML(env, url, node, callback) {
			return true, nil
		}
	}

	return false, nil
}
