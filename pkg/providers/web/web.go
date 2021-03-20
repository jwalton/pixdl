package web

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/jwalton/pixdl/pkg/download"
	"github.com/jwalton/pixdl/pkg/pixdl/meta"
	"github.com/jwalton/pixdl/pkg/providers/internal/htmlutils"
	"github.com/jwalton/pixdl/pkg/providers/types"
	"golang.org/x/net/html"
)

// TODO: Pass this in, so we're all using the same client.
// Also makes unit tests easier.
var client = download.NewDownloadClient()

// Skip tiny images.
const minImageSize = 5000

// TODO: Add options for min size, CSS selector, min dimensions.

// Provider returns the generic "web" provider.
func Provider() types.Provider {
	return webProvider{}
}

type webProvider struct{}

func (webProvider) Name() string {
	return "web"
}

// CanDownload returns true if this downloader can download an album from the given URL.
func (webProvider) CanDownload(url string) bool {
	return true
}

func (webProvider) FetchAlbum(url string, callback types.ImageCallback) {
	// Figure out what we're dealing with
	fileInfo, _ := client.GetFileInfo(url)

	if strings.HasPrefix(fileInfo.MimeType, "image/") {
		// This is an image - return a dummy album with just the image in it.
		singleImageAlbum(url, fileInfo, callback)

	} else if fileInfo.MimeType == "text/html" || fileInfo.MimeType == "" {
		resp, err := http.Get(url)
		if err != nil {
			callback(nil, nil, fmt.Errorf("Unable to fetch album: %s: %v", url, err))
			return
		}

		defer resp.Body.Close()

		album := &meta.AlbumMetadata{
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

			image := converLinkToImage(album, url, title, index)
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

		findPossibleImageLinks(url, resp.Body, linkHandler)
	} else {
		callback(nil, nil, fmt.Errorf("Don't know what to do with mime type: %s", fileInfo.MimeType))
	}
}

func converLinkToImage(album *meta.AlbumMetadata, url string, title string, nextImageIndex int) *meta.ImageMetadata {
	// Check to make sure this really is an image, and get info about the file.
	isImage, remoteInfo := checkIsImage(url)
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
	}

	return image
}

func singleImageAlbum(urlStr string, fileInfo *download.RemoteFileInfo, callback types.ImageCallback) {
	filename, err := getFilenameFromURL(urlStr)

	if err != nil {
		callback(nil, nil, err)
	}

	album := meta.AlbumMetadata{
		URL:             urlStr,
		AlbumID:         "",
		TotalImageCount: 1,
		Name:            filename,
	}

	image := meta.ImageMetadata{
		URL:        urlStr,
		Album:      &album,
		Filename:   filename,
		Title:      filename,
		Size:       fileInfo.Size,
		RemoteInfo: fileInfo,
		Timestamp:  fileInfo.LastModified,
		Index:      0,
	}

	cont := callback(&album, &image, nil)
	if cont {
		callback(&album, nil, nil)
	}
}

// This will call the callback with each possible image URL found, with the
// width and height if available, or -1 for each if unavailable.  `elType`
// will be either "img" or "src" depending on where this came from.
func findPossibleImageLinks(
	url string,
	input io.Reader,
	callback func(url string, elType string, title string, width int64, height int64, done bool, err error) (wantMore bool, isImage bool),
) {
	tokenizer := html.NewTokenizer(input)

	for {
		tokenType := tokenizer.Next()
		token := tokenizer.Token()

		err := tokenizer.Err()
		if err == io.EOF {
			// All done!
			callback("", "", "", -1, -1, true, nil)
			break
		} else if err != nil {
			callback("", "", "", -1, -1, true, err)
			break
		}

		switch tokenType {
		case html.StartTagToken:
			switch token.Data {
			case "nav":
				// Skip everything in the nav.
				htmlutils.SkipTokenContents(tokenizer, token.Data)
			case "a":
				href := htmlutils.GetTokenAttr(token, "href")
				if href != "" && !strings.HasPrefix(href, "#") {
					// TODO: Pass text content as title?
					wantMore, isImage := callback(href, "a", "", -1, -1, false, nil)
					if !wantMore {
						return
					}
					if isImage {
						// The "a" linked to an image - skip over any child elements,
						// because if there's an `img` in there, it's probably
						// going to be a thumbnail.
						htmlutils.SkipTokenContents(tokenizer, token.Data)
					}
				}
			case "img":
				attrs := htmlutils.GetAttrMap(token)
				width := htmlutils.GetNumericAttrFromMapWithDefault(attrs, "width", -1)
				height := htmlutils.GetNumericAttrFromMapWithDefault(attrs, "height", -1)
				src, _ := attrs["src"]
				title, _ := attrs["alt"]
				if title == "" {
					title, _ = attrs["title"]
				}
				if src != "" {
					wantMore, _ := callback(src, "img", title, width, height, false, nil)
					if !wantMore {
						return
					}
				}
			}
		}
	}
}

var knownImageExtensions = regexp.MustCompile(`(?i)\.(jpg|jpeg|jpe|jif|jfif|jfi|png|bmp|tiff|tif|heic|heif|raw|cr2|jp2|j2k|jpf|jpx|jpm|mj2|gif|webm|mov|mp4|mkv|)^`)

func checkIsImage(url string) (bool, *download.RemoteFileInfo) {
	if knownImageExtensions.MatchString(url) {
		return true, nil
	}

	fileInfo, err := client.GetFileInfo(url)
	if err != nil {
		// Could be the server just doesn't support HEAD... But let's drop it.
		return false, nil
	}

	if strings.HasPrefix(fileInfo.MimeType, "image/") || strings.HasPrefix(fileInfo.MimeType, "video/") {
		return true, nil
	}

	return false, nil
}

func getFilenameFromURL(urlStr string) (string, error) {
	parsedUrl, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}
	return path.Base(parsedUrl.Path), nil
}
