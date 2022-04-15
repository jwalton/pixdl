package providers

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/jwalton/pixdl/internal/htmlutils"
	"github.com/jwalton/pixdl/pkg/pixdl/meta"
	"golang.org/x/net/html"
)

type bunkrProvider struct{}

var bunkrRegex = regexp.MustCompile(`^(https://)?bunkr.is/a/(\w*)/?$`)

func (bunkrProvider) Name() string {
	return "bunkr.is"
}

// CanDownload returns true if this downloader can download an album from the given URL.
func (bunkrProvider) CanDownload(url string) bool {
	return bunkrRegex.MatchString(url)
}

func (provider bunkrProvider) FetchAlbum(env *Env, params map[string]string, url string, callback ImageCallback) {
	match := bunkrRegex.FindStringSubmatch(url)
	if match == nil {
		callback(nil, nil, fmt.Errorf("invalid bunkr.is URL: %s", url))
		return
	}

	albumID := match[2]

	resp, err := env.Get(url)
	if err != nil {
		callback(nil, nil, fmt.Errorf("unable to fetch album: %s: %v", url, err))
		return
	}

	defer resp.Body.Close()

	provider.parseAlbum(url, albumID, resp.Body, callback)
}

func (provider bunkrProvider) parseBunkrImage(album *meta.AlbumMetadata, tokenizer *html.Tokenizer, index int) (*meta.ImageMetadata, error) {
	image := meta.NewImageMetadata(album, index)
	image.Page = 1

	sizeRegex := regexp.MustCompile(`^\s*(\d*) B\s*$`)

	depth := 1
	for depth > 0 {
		tokenType := tokenizer.Next()
		token := tokenizer.Token()

		err := tokenizer.Err()
		if err == io.EOF {
			return nil, io.ErrUnexpectedEOF
		} else if err != nil {
			return nil, err
		}

		switch tokenType {
		case html.StartTagToken:
			depth = depth + 1

			if token.Data == "a" {
				attrs := htmlutils.GetAttrMap(token.Attr)
				className := attrs["class"]

				if className == "image" {
					href, ok := attrs["href"]
					if !ok {
						return nil, fmt.Errorf("image has no href")
					}
					image.URL = href
				}
			} else if token.Data == "p" {
				attrs := htmlutils.GetAttrMap(token.Attr)
				className := attrs["class"]
				style := attrs["style"]

				if className == "name" && style == "" {
					text, err := provider.getTextContent(tokenizer)
					if err == nil {
						image.Title = text
						image.Filename = text
					}
				} else if className == "file-size" {
					text, err := provider.getTextContent(tokenizer)
					if err == nil {
						match := sizeRegex.FindStringSubmatch(text)
						if match != nil {
							size, err := strconv.ParseInt(match[1], 10, 64)
							if err == nil {
								image.Size = size
							}
						}
					}
				}
			} else if token.Data == "img" {
				// Bunkr doesn't close their img tags.
				depth = depth - 1
			}

		case html.EndTagToken:
			depth = depth - 1
		}
	}

	if image.URL == "" {
		return nil, fmt.Errorf("image has no URL")
	}
	return image, nil
}

func (provider bunkrProvider) getTextContent(tokenizer *html.Tokenizer) (string, error) {
	tokenType := tokenizer.Next()
	token := tokenizer.Token()
	if tokenType == html.TextToken {
		return token.Data, nil
	}
	return "", fmt.Errorf("unexpected token type: %d", tokenType)
}

func (provider bunkrProvider) parseAlbum(
	url string,
	albumID string,
	input io.Reader,
	callback ImageCallback,
) {
	bunkrCountRegex := regexp.MustCompile(`^\s*(\d*) files`)

	// TODO: Do something useful with warnings.
	warnings := []string{}

	album := meta.AlbumMetadata{
		URL:             url,
		AlbumID:         albumID,
		Name:            "",
		Author:          "",
		TotalImageCount: 0,
	}

	imageIndex := 0

	tokenizer := html.NewTokenizer(input)

	for {
		tokenType := tokenizer.Next()
		token := tokenizer.Token()

		err := tokenizer.Err()
		if err == io.EOF {
			// All done!
			callback(&album, nil, nil)
			break
		} else if err != nil {
			callback(&album, nil, err)
			break
		}

		switch tokenType {
		case html.StartTagToken:
			switch token.Data {
			case "h1":
				attrs := htmlutils.GetAttrMap(token.Attr)
				id := attrs["id"]
				if id == "title" {
					// Parse the album title
					album.Name = attrs["title"]
				}
			case "p":
				id := htmlutils.GetAttr(token.Attr, "id")
				if id == "count" {
					// Parse the total file count
					countText, err := provider.getTextContent(tokenizer)
					if err == nil {
						match := bunkrCountRegex.FindStringSubmatch(countText)
						if match == nil {
							warnings = append(warnings, fmt.Sprintf("Could not parse file count: %s", strings.TrimSpace(countText)))
						} else {
							totalCount, err := strconv.Atoi(match[1])
							if err != nil {
								warnings = append(warnings, fmt.Sprintf("Album has invalid file count: %s", strings.TrimSpace(countText)))
							} else {
								album.TotalImageCount = totalCount
							}
						}
					}
				}

			case "div":
				attrs := htmlutils.GetAttrMap(token.Attr)
				className := attrs["class"]
				if strings.Contains(className, "image-container") {
					// Parse image
					image, err := provider.parseBunkrImage(&album, tokenizer, imageIndex)
					if err != nil {
						warnings = append(warnings, "Could not parse image: "+err.Error())
					} else {
						imageIndex++
						cont := callback(&album, image, err)
						if !cont {
							return
						}
					}
				}
			}
		}
	}

	if len(warnings) != 0 {
		fmt.Println("Warnings:")
		for _, warning := range warnings {
			fmt.Println("  " + warning)
		}
	}
}
