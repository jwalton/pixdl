package xenforo

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/jwalton/pixdl/pkg/pixdl/meta"
	"github.com/jwalton/pixdl/pkg/providers/env"
	"github.com/jwalton/pixdl/pkg/providers/internal/htmlutils"
	"github.com/jwalton/pixdl/pkg/providers/types"
	"golang.org/x/net/html"
)

// Provider returns the generic "web" provider.
func Provider() types.HTMLProvider {
	return xenforoProvider{}
}

type xenforoProvider struct{}

func (xenforoProvider) Name() string {
	return "xenforo"
}

var threadRegex = regexp.MustCompile(`(.*)/threads/(.*\.\d*)/(page-\d*)?`)

func getPageFromUrl(url string) (string, int) {
	match := threadRegex.FindStringSubmatch(url)
	albumId := ""
	page := 1
	if match != nil {
		albumId = match[2]
		if match[3] == "" {
			page = 1
		} else {
			page64, err := strconv.ParseInt(match[3][5:], 10, 64)
			if err == nil {
				page = int(page64)
			}
		}
	}

	return albumId, page
}

func (xenforoProvider) FetchAlbumFromHTML(env *env.Env, urlStr string, node *html.Node, callback types.ImageCallback) bool {
	// Look for `<div id="top" class="p-pageWrapper">`.
	topNode := htmlutils.FindNodeById(node, "top", 5)
	if !strings.Contains(htmlutils.GetNodeAttr(topNode, "class"), "p-pageWrapper") {
		return false
	}

	albumId, page := getPageFromUrl(urlStr)

	album := &meta.AlbumMetadata{
		URL:             urlStr,
		AlbumID:         albumId,
		Name:            "", // TODO: Use page title
		Author:          "",
		TotalImageCount: -1,
	}

	parsedUrl, err := url.Parse(urlStr)
	if err != nil {
		return false
	}

	index := 0
	seenURLs := map[string]struct{}{}

	running := true

	sendImage := func(image *meta.ImageMetadata) {
		if running && image != nil {
			_, seen := seenURLs[image.URL]
			if !seen {
				seenURLs[image.URL] = struct{}{}
				fmt.Printf("Image index: %d page: %d %s\n", index, image.Page, image.URL)
				running = callback(album, image, nil)
				index++
			}
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

		_, page = getPageFromUrl(nextLink)

		nextPage := htmlutils.ResolveURL(parsedUrl, nextLink)
		// TODO: Pass in HTTP client?  Want to set user-agent on these requests.
		resp, err := env.Get(nextPage)
		if err != nil {
			albumErr = err
			return
		}

		defer resp.Body.Close()

		node, err := html.Parse(resp.Body)
		if err != nil {
			albumErr = err
			return
		}

		walkDocument(node, false)
	}

	// Find all the images in a given page.
	walkDocument = func(node *html.Node, getAlbum bool) {
		htmlutils.WalkNodesPreOrder(node, func(node *html.Node) bool {
			if !running {
				return false
			}
			if node.Type == html.ElementNode && node.Data == "a" && htmlutils.NodeHasClass(node, "p-body-header") {
				parseAlbumInfo(node, album)
				return false
			}
			if node.Type == html.ElementNode && node.Data == "li" && htmlutils.NodeHasClass(node, "attachment") {
				image := parseAttachment(parsedUrl, node, album, page, index)
				sendImage(image)
				return false
			}
			if node.Type == html.ElementNode && node.Data == "img" && htmlutils.NodeHasClass(node, "bbImage") {
				image := parseInlineImage(parsedUrl, node, album, page, index)
				sendImage(image)
				return false
			}
			if node.Type == html.ElementNode && node.Data == "div" && htmlutils.NodeHasClass(node, "block-outer--after") {
				nextLink := findNextLink(node)
				if nextLink != "" {
					handleNextPage(nextLink)
				}
				return false
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

func parseAlbumInfo(node *html.Node, album *meta.AlbumMetadata) {
	htmlutils.WalkNodesPreOrder(node, func(node *html.Node) bool {
		if node.Type == html.ElementNode && node.Data == "a" && htmlutils.NodeHasClass(node, "p-description") {
			parseDescriptionBlock(node, album)
			return false
		}
		if node.Type == html.ElementNode && node.Data == "h1" && htmlutils.NodeHasClass(node, "p-title-value") {
			album.Name = htmlutils.GetNodeTextContent(node)
			return false
		}
		// If we found everything, stop.
		if album.Name != "" && album.Author != "" {
			return false
		}

		return true
	})
}

func parseDescriptionBlock(node *html.Node, album *meta.AlbumMetadata) {
	htmlutils.WalkNodesPreOrder(node, func(node *html.Node) bool {
		if node.Type == html.ElementNode && node.Data == "a" && htmlutils.NodeHasClass(node, "username") {
			album.Author = htmlutils.GetNodeTextContent(node)
			return false
		}
		// if node.Type == html.ElementNode && node.Data == "time" {
		// 	timestampStr := htmlutils.GetNodeAttr(node, "data-time")
		// 	unixTimestamp, err := strconv.ParseInt(timestampStr, 10, 64)
		// 	if err != nil {
		// 		timestamp := time.Unix(unixTimestamp, 0)
		// 	}
		// 	return false
		// }

		return true
	})
}

func parseAttachment(
	parsedUrl *url.URL,
	node *html.Node,
	album *meta.AlbumMetadata,
	page int,
	index int,
) *meta.ImageMetadata {
	imageUrlPath := ""
	imageName := ""

	htmlutils.WalkNodesPreOrder(node, func(node *html.Node) bool {
		if node.Type == html.ElementNode && node.Data == "div" && htmlutils.NodeHasClass(node, "attachment-name") {
			imageName = strings.TrimSpace(htmlutils.GetNodeTextContent(node))
			return false
		}
		if node.Type == html.ElementNode && node.Data == "a" && htmlutils.NodeHasClass(node, "js-lbImage") {
			imageUrlPath = htmlutils.GetNodeAttr(node, "href")
			return false
		}
		return true
	})

	if imageUrlPath != "" {
		image := meta.NewImageMetadata(album, index)
		image.Filename = imageName
		image.Page = page
		image.URL = htmlutils.ResolveURL(parsedUrl, imageUrlPath)
		return image
	}

	return nil
}

func parseInlineImage(
	parsedUrl *url.URL,
	node *html.Node,
	album *meta.AlbumMetadata,
	page int,
	index int,
) *meta.ImageMetadata {
	src := htmlutils.GetNodeAttr(node, "src")
	alt := htmlutils.GetNodeAttr(node, "alt")

	if src != "" && src != "#" {
		image := meta.NewImageMetadata(album, index)
		image.Filename = alt
		image.Page = page
		image.URL = htmlutils.ResolveURL(parsedUrl, src)
		return image
	}

	return nil
}

func findNextLink(node *html.Node) string {
	nextLink := ""

	htmlutils.WalkNodesPreOrder(node, func(node *html.Node) bool {
		if node.Type == html.ElementNode && node.Data == "a" && htmlutils.NodeHasClass(node, "pageNav-jump--next") {
			nextLink = htmlutils.GetNodeAttr(node, "href")
			return false
		}
		if nextLink != "" {
			return false
		}
		return true
	})

	return nextLink
}
