package providers

import (
	"fmt"

	"github.com/jwalton/pixdl/pkg/providers/imgur"
	"github.com/jwalton/pixdl/pkg/providers/types"
	"github.com/jwalton/pixdl/pkg/providers/web"
)

type Provider = types.Provider

var providerRegistry = []Provider{
	imgur.Provider(),
	web.Provider(),
}

func registerProvider(provider Provider) {
	providerRegistry = append(providerRegistry, provider)
}

func GetProviderForURL(url string) (Provider, error) {
	for index := range providerRegistry {
		provider := providerRegistry[index]
		if provider.CanDownload(url) {
			return provider, nil
		}
	}

	return nil, fmt.Errorf("Could not find suitable downloader for url: %s", url)
}
