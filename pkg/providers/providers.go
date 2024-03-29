package providers

import (
	"fmt"

	"github.com/jwalton/pixdl/pkg/pixdl/meta"
)

// URLProviderRegistry is a list of all URLProviders.
var URLProviderRegistry = []URLProvider{
	imgurProvider{},
	gofileProvider{},
	singleimageProvider{},
}

// HTMLProviderRegistry is a list of all HTMLProviders.
var HTMLProviderRegistry = []HTMLProvider{
	xenforoProvider{},
	// Web will download just about anything, so it should always be last in this list.
	webProvider{},
}

var imageProviderRegistry = []URLImageProvider{}

func fetchImage(
	env *Env,
	params map[string]string,
	album *meta.AlbumMetadata,
	url string,
) (image *meta.ImageMetadata, err error) {
	for _, provider := range imageProviderRegistry {
		if provider.CanFetchImage(url) {
			image, err = provider.FetchImage(env, params, album, url)
			if err == nil && image != nil {
				return image, nil
			}
		}
	}

	return nil, fmt.Errorf("Unable to fetch image: %s", url)
}
