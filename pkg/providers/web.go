package providers

import (
	"strings"
	"time"

	"github.com/jwalton/pixdl/pkg/download"
	"github.com/jwalton/pixdl/pkg/pixdl/meta"
	"github.com/jwalton/pixdl/pkg/providers/internal/htmlutils"
	"golang.org/x/net/html"
)

// Skip tiny images.
const minImageSize = 5000

// TODO: Add options for min size, CSS selector, min dimensions.

type webProvider struct{}

func (webProvider) Name() string {
	return "web"
}

func (webProvider) FetchAlbumFromHTML(env *Env, params map[string]string, url string, node *html.Node, callback ImageCallback) bool {
	album := &meta.AlbumMetadata{
		Provider:        "web",
		URL:             url,
		AlbumID:         url,
		Name:            "", // TODO: Use page title
		Author:          "",
		TotalImageCount: -1,
	}

	index := 0

	seenURLs := map[string]bool{}

	linkHandler := func(
		url string,
		elType string,
		title string,
		width int64,
		height int64,
		done bool,
		err error,
	) (wantMore bool, isImage bool) {
		if done {
			return callback(album, nil, err), true
		}

		// Don't visit the same URL twice.
		if _, seen := seenURLs[url]; seen {
			return true, false
		}
		seenURLs[url] = true

		image := convertLinkToImage(env, album, url, title, index)
		if image != nil {
			// Skip small `img` tags.
			if image.Size != -1 && elType != "a" && image.Size < minImageSize {
				// Skip this image.
				return true, true
			}

			index++
			return callback(album, image, nil), true
		}

		// Not an image... keep going.
		return true, false
	}

	findPossibleImageLinks(url, node, linkHandler)

	return index != 0
}

func convertLinkToImage(env *Env, album *meta.AlbumMetadata, url string, title string, nextImageIndex int) *meta.ImageMetadata {
	// Check to make sure this really is an image, and get info about the file.
	isImage, remoteInfo := checkIsImage(env, url)
	if !isImage {
		return nil
	}

	size := int64(-1)
	filename := ""
	var timestamp *time.Time
	if remoteInfo != nil {
		size = remoteInfo.Size
		filename = remoteInfo.Filename
		timestamp = remoteInfo.LastModified
	}

	if filename == "" {
		filename, _ = getFilenameFromURL(url)
	}

	image := &meta.ImageMetadata{
		Album:      album,
		URL:        url,
		Filename:   filename,
		Title:      title,
		Size:       size,
		Timestamp:  timestamp,
		Index:      nextImageIndex,
		RemoteInfo: remoteInfo,
		Page:       1,
	}

	return image
}

// This will call the callback with each possible image URL found, with the
// width and height if available, or -1 for each if unavailable.  `elType`
// will be either "img" or "src" depending on where this came from.
func findPossibleImageLinks(
	url string,
	node *html.Node,
	callback func(url string, elType string, title string, width int64, height int64, done bool, err error) (wantMore bool, isImage bool),
) {
	running := true

	htmlutils.WalkNodesPreOrder(node, func(node *html.Node) bool {
		if !running {
			return false
		}

		switch node.Type {
		case html.ElementNode:
			switch node.Data {
			case "nav":
				// Skip everything in the nav.
				return false
			case "a":
				href := htmlutils.GetAttr(node.Attr, "href")
				if href != "" && !strings.HasPrefix(href, "#") {
					title := htmlutils.GetNodeTextContent(node)
					wantMore, isImage := callback(href, "a", title, -1, -1, false, nil)
					if !wantMore {
						running = false
						return false
					}
					if isImage {
						// The "a" linked to an image - skip over any child elements,
						// because if there's an `img` in there, it's probably
						// going to be a thumbnail.
						return false
					}
				}
			case "img":
				attrs := htmlutils.GetAttrMap(node.Attr)
				width := htmlutils.GetNumericAttrFromMapWithDefault(attrs, "width", -1)
				height := htmlutils.GetNumericAttrFromMapWithDefault(attrs, "height", -1)
				src := attrs["src"]
				title := attrs["alt"]
				if title == "" {
					title = attrs["title"]
				}
				if src != "" {
					wantMore, _ := callback(src, "img", title, width, height, false, nil)
					if !wantMore {
						running = false
						return false
					}
				}
			}
		}
		return true
	})
}

func checkIsImage(env *Env, url string) (bool, *download.RemoteFileInfo) {
	if IsImageByExtension(url) {
		return true, nil
	}

	fileInfo, err := env.GetFileInfo(url)
	if err != nil {
		// Could be the server just doesn't support HEAD... But let's drop it.
		return false, nil
	}

	if strings.HasPrefix(fileInfo.MimeType, "image/") || strings.HasPrefix(fileInfo.MimeType, "video/") {
		return true, nil
	}

	return false, nil
}
