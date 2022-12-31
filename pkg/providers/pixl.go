package providers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"

	"github.com/jwalton/pixdl/internal/htmlutils"
	"github.com/jwalton/pixdl/pkg/pixdl/meta"
	"golang.org/x/net/html"
)

/*
* The HTML for a pixl page consists of a set of "div.list-item" for each image.
* Each of these has a "data-object" attribute which contains a JSON object
* with all the information we need.
*
* At the bottom of the page an `a[data-pagination="next"]` which links to the
* next page in the album.  If there is no next page, this anchor tag will still
* be present, but will be missing an href.
*
* On album and use pages, there will be an element with data-text="image-count"
* which contains the total number of images in the album.  This is a "b" tag
* for users, and a "span" for albums.
 */

type pixlUser struct {
	URL           string `json:"url"`
	Username      string `json:"username"`
	NameShortHTML string `json:"name_short_html"`
}

type pixlDataObject struct {
	// IDEncoded string `json:"id_encoded"`
	Image struct {
		Filename  string `json:"filename"`
		Name      string `json:"name"`
		MimeType  string `json:"mime"`
		Extension string `json:"extension"`
		URL       string `json:"url"`
		Size      string `json:"size"`
	} `json:"image"`
	Filename string `json:"filename"`
	Name     string `json:"name"`
	Title    string `json:"title"`
	URL      string `json:"url"`
	Width    string `json:"width"`
	Height   string `json:"height"`
}

type pixlProvider struct{}

func (pixlProvider) Name() string {
	return "pixl.li"
}

func (pixlProvider) parseUrl(pixlURL string) (pageType string, albumName string, page int) {
	u, err := url.Parse(pixlURL)
	hostname := u.Hostname()

	supportedHost := hostname == "pixl.li" || hostname == "jpg.church"

	if err != nil || !supportedHost || u.Path == "/login" {
		return "", "", 0
	}

	if strings.HasPrefix(u.Path, "/image/") {
		return "image", strings.TrimPrefix(u.Path, "/image/"), 1
	}

	if strings.HasPrefix(u.Path, "/img/") {
		return "image", strings.TrimPrefix(u.Path, "/img/"), 1
	}

	// Assume everything else is an album.
	// Fetch the album page number from the query, if present.
	page = 1
	query, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		pageQuery, ok := query["page"]
		if ok && len(pageQuery) == 1 {
			p, err := strconv.Atoi(pageQuery[0])
			if err == nil {
				page = p
			}
		}
	}

	return "album", u.Path, page
}

// CanDownload returns true if this downloader can download an album from the given URL.
func (provider pixlProvider) CanDownload(url string) bool {
	pageType, _, _ := provider.parseUrl(url)
	return pageType == "album"
}

func (provider pixlProvider) FetchAlbum(env *Env, params map[string]string, url string, callback ImageCallback) {
	pageType, albumID, pageNumber := provider.parseUrl(url)

	if pageType != "album" {
		callback(nil, nil, fmt.Errorf("invalid pixl.li URL"))
	}

	album := meta.AlbumMetadata{
		URL:             url,
		AlbumID:         albumID,
		Name:            "",
		Author:          "",
		TotalImageCount: 0,
	}

	nextImageIndex := pageNumber * 104
	nextPageURL := url
	wantMore := true

	for nextPageURL != "" && wantMore {
		nextPageURL = provider.parseAlbumPage(
			env,
			&album,
			nextPageURL,
			&nextImageIndex,
			&wantMore,
			callback,
		)
	}
}

// Parses a single page of an album.  Returns the next page in the album.
func (provider pixlProvider) parseAlbumPage(
	env *Env,
	album *meta.AlbumMetadata,
	pageURL string,
	nextImageIndex *int,
	wantMore *bool,
	callback ImageCallback,
) (nextPageURL string) {
	_, _, pageNumber := provider.parseUrl(pageURL)
	nextPageURL = ""

	resp, err := env.Get(pageURL)
	if err != nil {
		callback(nil, nil, fmt.Errorf("unable to fetch album: %s: %v", pageURL, err))
		return
	}

	defer resp.Body.Close()

	tokenizer := html.NewTokenizer(resp.Body)

	for *wantMore {
		tokenType := tokenizer.Next()
		token := tokenizer.Token()

		err := tokenizer.Err()
		if err == io.EOF {
			// All done!
			break
		} else if err != nil {
			callback(album, nil, err)
			nextPageURL = ""
			break
		}

		switch tokenType {
		case html.StartTagToken:
			attrs := htmlutils.GetAttrMap(token.Attr)

			// Handle total image count
			if dataText, ok := attrs["data-text"]; ok {
				if dataText == "image-count" {
					imageCountStr := htmlutils.GetTokenTextContent(tokenizer)
					if imageCountStr != "" {
						imageCount, err := strconv.Atoi(imageCountStr)
						if err == nil {
							album.TotalImageCount = imageCount
						}
					}
				}
			}

			// Handle an image
			if dataObjectStr, ok := attrs["data-object"]; ok && htmlutils.HasClass(token.Attr, "list-item") {
				image, err := provider.parseDataObject(album, nextImageIndex, pageNumber, dataObjectStr)
				if err != nil {
					fmt.Println("Warning: Error parsing data object: ", err)
				} else {
					err := provider.parseImage(tokenizer, album, image)
					if err != nil {
						fmt.Println("Warning: Error parsing details: ", err)
					} else {
						*wantMore = callback(album, image, nil)
					}
				}
			}

			// Handle next page
			if pagination, ok := attrs["data-pagination"]; ok && token.Data == "a" && pagination == "next" {
				href, ok := attrs["href"]
				if ok && href != "" {
					nextPageURL = href
				}
			}
		}
	}

	return nextPageURL
}

func (pixlProvider) parseImage(
	tokenizer *html.Tokenizer,
	album *meta.AlbumMetadata,
	image *meta.ImageMetadata,
) error {
	depth := 1

	for depth > 0 {
		tokenType := tokenizer.Next()
		token := tokenizer.Token()

		err := tokenizer.Err()
		if err == io.EOF {
			return io.ErrUnexpectedEOF
		} else if err != nil {
			return err
		}

		switch tokenType {
		case html.StartTagToken:
			if token.Data != "img" {
				depth = depth + 1
			}

			if token.Data == "a" {
				attrs := htmlutils.GetAttrMap(token.Attr)
				dataText, ok := attrs["data-text"]
				if ok && dataText == "album-name" {
					image.SubAlbum = htmlutils.GetTokenTextContent(tokenizer)
				}
			}

		case html.EndTagToken:
			if token.Data != "img" {
				depth = depth - 1
			}
		}
	}

	return nil
}

func (pixlProvider) parseDataObject(
	album *meta.AlbumMetadata,
	nextImageIndex *int,
	pageNumber int,
	dataObjectStr string,
) (*meta.ImageMetadata, error) {
	decoded, err := url.QueryUnescape(dataObjectStr)
	if err != nil {
		return nil, err
	}

	dataObject := pixlDataObject{}
	err = json.Unmarshal([]byte(decoded), &dataObject)
	if err != nil {
		return nil, err
	}

	image := meta.NewImageMetadata(album, *nextImageIndex)
	*nextImageIndex++

	imageSize, err := strconv.ParseInt(dataObject.Image.Size, 10, 64)
	if err != nil {
		imageSize = -1
	}

	image.URL = dataObject.URL
	image.Filename = dataObject.Filename
	image.Title = dataObject.Title
	image.Page = pageNumber
	image.Size = imageSize

	return image, nil
}
