package providers

import (
	"github.com/jwalton/pixdl/pkg/pixdl/meta"
	"golang.org/x/net/html"
)

// URLProvider represents a back-end which can read album and image metadata
// from a server.  URLProvider differs from HTMLProvider in that it can
// decide whether or not it can fetch an album given only a URL.
type URLProvider interface {
	// Name is the name of this provider.
	Name() string
	// CanDownload returns true if this Provider can fetch the specified URL.
	CanDownload(url string) bool
	// FetchAlbum will fetch all images in an album, and pass them to the ImageCallback.
	FetchAlbum(env *Env, params map[string]string, url string, callback ImageCallback)
}

// HTMLProvider represents a back-end which can figure out if a given HTML document
// represents an image album, and find images in that album.
type HTMLProvider interface {
	// Name is the name of this provider.
	Name() string
	// FetchAlbum will fetch all images in an album, and pass them to the ImageCallback.
	// If this provider cannot download images from this album, returns `false`
	// immediately.  If any images were successfully fetched, returns true.
	FetchAlbumFromHTML(env *Env, params map[string]string, url string, node *html.Node, callback ImageCallback) bool
}

// URLImageProvider downloads a single image. This is used when a given website
// (say a Xenforo forum) has an album containing links to images from external
// image hosts.
type URLImageProvider interface {
	// Name is the name of this provider.
	Name() string
	// CanFetchImage returns true if this Provider can fetch the specified URL.
	CanFetchImage(url string) bool
	// FetchImage will get information about the image at the specified URL.
	FetchImage(
		env *Env,
		params map[string]string,
		album *meta.AlbumMetadata,
		url string,
	) (image *meta.ImageMetadata, err error)
}

// ImageCallback is a function called by a Provider for each image in an album.
// This will be called once for each image, and then with `album, nil, nil` when
// there are no more images.
//
// If an error occurs fetching images, this will be called with err set.
//
// Implemnetations can return false to stop the Provider from providing any
// further images.
type ImageCallback func(
	album *meta.AlbumMetadata,
	image *meta.ImageMetadata,
	err error,
) (wantMore bool)
