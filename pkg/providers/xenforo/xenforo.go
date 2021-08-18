package xenforo

import (
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

func getPageFromURL(url string) (string, int) {
	match := threadRegex.FindStringSubmatch(url)
	albumID := ""
	page := 1
	if match != nil {
		albumID = match[2]
		if match[3] == "" {
			page = 1
		} else {
			page64, err := strconv.ParseInt(match[3][5:], 10, 64)
			if err == nil {
				page = int(page64)
			}
		}
	}

	return albumID, page
}

func (xenforoProvider) FetchAlbumFromHTML(env *env.Env, urlStr string, node *html.Node, callback types.ImageCallback) bool {
	// Look for `<div id="top" class="p-pageWrapper">`.
	topNode := htmlutils.FindNodeByID(node, "top", 5)
	if !strings.Contains(htmlutils.GetAttr(topNode.Attr, "class"), "p-pageWrapper") {
		return false
	}

	albumID, page := getPageFromURL(urlStr)

	album := &meta.AlbumMetadata{
		Provider:        "xenforo",
		URL:             urlStr,
		AlbumID:         albumID,
		Name:            urlStr, // TODO: Use page title
		Author:          "",
		TotalImageCount: -1,
	}

	parsedURL, err := url.Parse(urlStr)
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

		_, page = getPageFromURL(nextLink)

		nextPage := htmlutils.ResolveURL(parsedURL, nextLink)
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

	parsePost := func(
		parsedURL *url.URL,
		node *html.Node,
		album *meta.AlbumMetadata,
		page int,
	) {
		subAlbum := ""
		htmlutils.WalkNodesPreOrder(node, func(node *html.Node) bool {
			// Grab the post number from the upper right corner.
			if node.Type == html.ElementNode && node.Data == "a" && strings.HasPrefix(htmlutils.GetAttr(node.Attr, "href"), "/threads") {
				post := htmlutils.GetNodeTextContent(node)
				post = strings.TrimSpace(post)
				post = strings.TrimPrefix(post, "#")
				subAlbum = post
				return false
			}

			// 'link--external' is a link to an image on an external site.
			// We don't handle these yet.
			if node.Type == html.ElementNode && node.Data == "a" && htmlutils.HasClass(node.Attr, "link--external") {
				return false
			}

			if node.Type == html.ElementNode && node.Data == "li" && htmlutils.HasClass(node.Attr, "attachment") {
				image := parseAttachment(parsedURL, node, album, subAlbum, page, index)
				sendImage(image)
				return false
			}
			if node.Type == html.ElementNode && node.Data == "img" && htmlutils.HasClass(node.Attr, "bbImage") {
				image := parseInlineImage(parsedURL, node, album, subAlbum, page, index)
				sendImage(image)
				return false
			}
			if node.Type == html.ElementNode && node.Data == "a" && htmlutils.HasClass(node.Attr, "js-lbImage") {
				// js-lbImage can show up in an attachment, but also in a `bbWrapper` div, where there's just
				// a whole bunch of js-lbImage with no other metadata.
				image := parseLBImage(parsedURL, node, album, subAlbum, page, index)
				sendImage(image)
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
			if node.Type == html.ElementNode && node.Data == "a" && htmlutils.HasClass(node.Attr, "p-body-header") {
				if getAlbum {
					parseAlbumInfo(node, album)
				}
				return false
			}
			if node.Type == html.ElementNode && node.Data == "article" {
				parsePost(parsedURL, node, album, page)
				return false
			}
			if node.Type == html.ElementNode && node.Data == "div" && htmlutils.HasClass(node.Attr, "block-outer--after") {
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
		if node.Type == html.ElementNode && node.Data == "a" && htmlutils.HasClass(node.Attr, "p-description") {
			parseDescriptionBlock(node, album)
			return false
		}
		if node.Type == html.ElementNode && node.Data == "h1" && htmlutils.HasClass(node.Attr, "p-title-value") {
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
		if node.Type == html.ElementNode && node.Data == "a" && htmlutils.HasClass(node.Attr, "username") {
			album.Author = htmlutils.GetNodeTextContent(node)
			return false
		}
		// if node.Type == html.ElementNode && node.Data == "time" {
		// 	timestampStr := htmlutils.GetAttr(node.Attr, "data-time")
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
	parsedURL *url.URL,
	node *html.Node,
	album *meta.AlbumMetadata,
	subAlbum string,
	page int,
	index int,
) *meta.ImageMetadata {
	imageURLPath := ""
	imageName := ""

	htmlutils.WalkNodesPreOrder(node, func(node *html.Node) bool {
		if node.Type == html.ElementNode && node.Data == "div" && htmlutils.HasClass(node.Attr, "attachment-name") {
			imageName = strings.TrimSpace(htmlutils.GetNodeTextContent(node))
			return false
		}
		if node.Type == html.ElementNode && node.Data == "a" && htmlutils.HasClass(node.Attr, "js-lbImage") {
			imageURLPath = htmlutils.GetAttr(node.Attr, "href")
			return false
		}
		return true
	})

	if imageURLPath != "" {
		image := meta.NewImageMetadata(album, index)
		image.SubAlbum = subAlbum
		image.Filename = imageName
		image.Page = page
		image.URL = htmlutils.ResolveURL(parsedURL, imageURLPath)
		return image
	}

	return nil
}

func parseLBImage(
	parsedURL *url.URL,
	node *html.Node,
	album *meta.AlbumMetadata,
	subAlbum string,
	page int,
	index int,
) *meta.ImageMetadata {
	imageURLPath := ""
	imageName := ""

	htmlutils.WalkNodesPreOrder(node, func(node *html.Node) bool {
		if node.Type == html.ElementNode && node.Data == "img" {
			imageName = strings.TrimSpace(htmlutils.GetAttr(node.Attr, "alt"))
			return false
		}
		if node.Type == html.ElementNode && node.Data == "a" && htmlutils.HasClass(node.Attr, "js-lbImage") {
			imageURLPath = htmlutils.GetAttr(node.Attr, "href")
		}
		return true
	})

	if imageURLPath != "" {
		image := meta.NewImageMetadata(album, index)
		image.SubAlbum = subAlbum
		image.Filename = imageName
		image.Page = page
		image.URL = htmlutils.ResolveURL(parsedURL, imageURLPath)
		return image
	}

	return nil
}

func parseInlineImage(
	parsedURL *url.URL,
	node *html.Node,
	album *meta.AlbumMetadata,
	subAlbum string,
	page int,
	index int,
) *meta.ImageMetadata {
	src := htmlutils.GetAttr(node.Attr, "src")
	alt := htmlutils.GetAttr(node.Attr, "alt")

	if src != "" && src != "#" {
		image := meta.NewImageMetadata(album, index)
		image.SubAlbum = subAlbum
		image.Filename = alt
		image.Page = page
		image.URL = htmlutils.ResolveURL(parsedURL, src)
		return image
	}

	return nil
}

func findNextLink(node *html.Node) string {
	nextLink := ""

	htmlutils.WalkNodesPreOrder(node, func(node *html.Node) bool {
		if node.Type == html.ElementNode && node.Data == "a" && htmlutils.HasClass(node.Attr, "pageNav-jump--next") {
			nextLink = htmlutils.GetAttr(node.Attr, "href")
			return false
		}
		if nextLink != "" {
			return false
		}
		return true
	})

	return nextLink
}
