package providers

import (
	"github.com/jwalton/pixdl/pkg/providers/imgur"
	"github.com/jwalton/pixdl/pkg/providers/singleimage"
	"github.com/jwalton/pixdl/pkg/providers/types"
	"github.com/jwalton/pixdl/pkg/providers/web"
	"github.com/jwalton/pixdl/pkg/providers/xenforo"
)

type URLProvider = types.URLProvider
type HTMLProvider = types.HTMLProvider

var UrlProviderRegistry = []URLProvider{
	imgur.Provider(),
	singleimage.Provider(),
}

var HtmlProviderRegistry = []HTMLProvider{
	xenforo.Provider(),
	// Web will download just about anything, so it should always be last in this list.
	web.Provider(),
}
