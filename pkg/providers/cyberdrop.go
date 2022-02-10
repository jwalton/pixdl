package providers

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jwalton/pixdl/pkg/pixdl/meta"
	"github.com/jwalton/pixdl/pkg/providers/internal/htmlutils"
	"golang.org/x/net/html"
)

type cyberdropProvider struct{}

var cyberdropRegex = regexp.MustCompile(`^(https://)?cyberdrop.me/a/(\w*)/?$`)

func (cyberdropProvider) Name() string {
	return "cyberdrop.me"
}

// CanDownload returns true if this downloader can download an album from the given URL.
func (cyberdropProvider) CanDownload(url string) bool {
	return cyberdropRegex.MatchString(url)
}

func (provider cyberdropProvider) FetchAlbum(env *Env, params map[string]string, url string, callback ImageCallback) {
	match := cyberdropRegex.FindStringSubmatch(url)
	if match == nil {
		callback(nil, nil, fmt.Errorf("Invalid cyberdrop.me URL: %s", url))
		return
	}

	albumID := match[2]

	resp, err := env.Get(url)
	if err != nil {
		callback(nil, nil, fmt.Errorf("Unable to fetch album: %s: %v", url, err))
		return
	}

	defer resp.Body.Close()

	provider.parseAlbum(url, albumID, resp.Body, callback)
}

func (provider cyberdropProvider) parseCyberdropImage(album *meta.AlbumMetadata, tokenizer *html.Tokenizer, index int) (*meta.ImageMetadata, error) {
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
					title := attrs["title"]

					href, ok := attrs["href"]
					if !ok {
						return nil, fmt.Errorf("Image has no href")
					}

					image.URL = href
					image.Filename = title
					image.Title = title

					timestampStr, ok := attrs["data-timestamp"]
					if ok {
						timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
						if err == nil {
							t := time.Unix(timestamp, 0)
							image.Timestamp = &t
						}
					}
				}
			} else if token.Data == "p" && htmlutils.GetAttr(token.Attr, "class") == "is-hidden file-size" {
				tokenType := tokenizer.Next()
				token := tokenizer.Token()
				if tokenType == html.TextToken {
					match := sizeRegex.FindStringSubmatch(token.Data)
					if match != nil {
						size, err := strconv.ParseInt(match[1], 10, 64)
						if err == nil {
							image.Size = size
						}
					}
				}

			}

		case html.EndTagToken:
			depth = depth - 1
		}
	}

	if image.URL == "" {
		return nil, fmt.Errorf("Image has no URL")
	}
	return image, nil
}

func (provider cyberdropProvider) parseAlbum(
	url string,
	albumID string,
	input io.Reader,
	callback ImageCallback,
) {
	cyberdropCountRegex := regexp.MustCompile(`^\s*(\d*) files`)

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
				} else if id == "count" {
					// Parse the total file count
					tokenType := tokenizer.Next()
					token := tokenizer.Token()
					if tokenType == html.TextToken {
						match := cyberdropCountRegex.FindStringSubmatch(token.Data)
						if match == nil {
							warnings = append(warnings, fmt.Sprintf("Could not parse file count: %s", strings.TrimSpace(token.Data)))
						} else {
							totalCount, err := strconv.Atoi(match[1])
							if err != nil {
								warnings = append(warnings, fmt.Sprintf("Album has invalid file count: %s", strings.TrimSpace(token.Data)))
							} else {
								album.TotalImageCount = totalCount
							}
						}
					}
				}

			case "div":
				attrs := htmlutils.GetAttrMap(token.Attr)

				className := attrs["class"]

				if strings.Contains(className, "subtitle") {
					title, ok := attrs["title"]
					if ok && len(title) > 3 {
						album.Author = title[3:]
					}
				} else if strings.Contains(className, "image-container") {
					// Parse image
					image, err := provider.parseCyberdropImage(&album, tokenizer, imageIndex)
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
