package providers

import (
	"github.com/jwalton/pixdl/pkg/providers/imgur"
	"github.com/jwalton/pixdl/pkg/providers/singleimage"
	"github.com/jwalton/pixdl/pkg/providers/types"
	"github.com/jwalton/pixdl/pkg/providers/web"
	"github.com/jwalton/pixdl/pkg/providers/xenforo"
)

// URLProviderRegistry is a list of all URLProviders.
var URLProviderRegistry = []types.URLProvider{
	imgur.Provider(),
	singleimage.Provider(),
}

// HTMLProviderRegistry is a list of all HTMLProviders.
var HTMLProviderRegistry = []types.HTMLProvider{
	xenforo.Provider(),
	// Web will download just about anything, so it should always be last in this list.
	web.Provider(),
}
