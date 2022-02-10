package providers

import (
	"encoding/json"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/jwalton/pixdl/pkg/pixdl/meta"
	"github.com/jwalton/pixdl/pkg/providers/internal/htmlutils"
	"golang.org/x/net/html"
)

type putmegaProvider struct{}

type putmegaImage struct {
	// Filename is the name of the file.
	Filename string `json:"filename"`
	// Name is the name of the image.
	Name string `json:"name"`
	// URL is the URL to the image.
	URL string `json:"url"`
	// Size is the size of the image in bytes.
	Size string `json:"size"`
}

type putmegaImageData struct {
	// Image is data about the full-sized image.
	Image putmegaImage `json:"image"`
}

var putmegapRegex = regexp.MustCompile(`^(https://)?(?:putme.ga|putmega.com)/album/(\w*)/?.*$`)

func (putmegaProvider) Name() string {
	return "putmega"
}

// CanDownload returns true if this downloader can download an album from the given URL.
func (putmegaProvider) CanDownload(url string) bool {
	return putmegapRegex.MatchString(url)
}

func (provider putmegaProvider) getPageFromURL(urlString string) int {
	page := 1
	// Figure out what page we're on.
	parsedURL, err := url.Parse(urlString)
	if err == nil {
		pageStr := parsedURL.Query().Get("page")
		if pageStr != "" {
			page, err = strconv.Atoi(pageStr)
			if err != nil {
				page = 1
			}
		}
	}
	return page
}

func (provider putmegaProvider) FetchAlbumFromHTML(env *Env, params map[string]string, urlString string, node *html.Node, callback ImageCallback) bool {
	match := putmegapRegex.FindStringSubmatch(urlString)
	if match == nil {
		// This is not a putme.ga URL.
		return false
	}

	albumID := match[2]

	album := &meta.AlbumMetadata{
		Provider:        provider.Name(),
		URL:             urlString,
		AlbumID:         albumID,
		Name:            urlString,
		Author:          "",
		TotalImageCount: -1,
	}

	index := 0
	page := provider.getPageFromURL(urlString)
	running := true

	sendImage := func(image *meta.ImageMetadata) {
		if running && image != nil {
			running = callback(album, image, nil)
			index++
		}
	}

	var albumErr error
	var handleNextPage func(nextLink string)
	var walkDocument func(node *html.Node, getAlbum bool)

	// Go fetch the next page, and read all the images from it.
	handleNextPage = func(nextLink string) {
		if !running {
			return
		}

		node, err := env.GetHTML(nextLink)
		if err != nil {
			albumErr = err
			return
		}

		page++
		walkDocument(node, false)
	}

	parseImages := func(node *html.Node) {
		htmlutils.WalkNodesPreOrder(node, func(node *html.Node) bool {
			if node.Type == html.ElementNode && node.Data == "div" && htmlutils.HasClass(node.Attr, "list-item") {
				dataAttr := htmlutils.GetAttr(node.Attr, "data-object")
				if dataAttr != "" {
					dataStr, err := url.QueryUnescape(dataAttr)
					if err != nil {
						return false
					}
					data := putmegaImageData{}
					err = json.Unmarshal([]byte(dataStr), &data)
					if err != nil {
						return false
					}
					var size int64
					if data.Image.Size != "" {
						size, _ = strconv.ParseInt(data.Image.Size, 10, 64)
					}
					if data.Image.URL == "" {
						return false
					}

					if err == nil {
						sendImage(&meta.ImageMetadata{
							Album:    album,
							URL:      data.Image.URL,
							Filename: data.Image.Filename,
							Title:    data.Image.Name,
							Size:     size,
							Index:    index,
							Page:     page,
						})
					}

				}
				return false
			}
			return true
		})
	}

	// Find all the images in a given page.
	walkDocument = func(node *html.Node, getAlbum bool) {
		htmlutils.WalkNodesPreOrder(node, func(node *html.Node) bool {
			if !running {
				return false
			}

			if node.Type == html.ElementNode {
				if getAlbum {
					if node.Data == "h1" {
						album.Name = strings.TrimSpace(htmlutils.GetNodeTextContent(node))
						return false
					} else if node.Data == "a" && htmlutils.HasClass(node.Attr, "user-link") {
						album.Author = strings.TrimSpace(htmlutils.GetNodeTextContent(node))
						return false
					} else if node.Data == "span" && htmlutils.GetAttr(node.Attr, "data-text") == "image-count" {
						imageCount := strings.TrimSpace(htmlutils.GetNodeTextContent(node))
						album.TotalImageCount, _ = strconv.Atoi(imageCount)
						return false
					}
				}

				if htmlutils.HasClass(node.Attr, "pad-content-listing") {
					parseImages(node)
					return false
				} else if node.Data == "a" && htmlutils.GetAttr(node.Attr, "data-pagination") == "next" {
					nextLink := htmlutils.GetAttr(node.Attr, "href")
					if nextLink != "" {
						handleNextPage(nextLink)
					}
					return false
				}
			}

			return true
		})
	}

	// Start walking the current page.
	walkDocument(node, true)

	// All done
	if running {
		callback(album, nil, albumErr)
	}

	return true
}
