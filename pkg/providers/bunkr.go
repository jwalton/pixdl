package providers

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"time"

	"github.com/jwalton/pixdl/internal/htmlutils"
	"github.com/jwalton/pixdl/pkg/download"
	"github.com/jwalton/pixdl/pkg/pixdl/meta"
	"golang.org/x/net/html"
)

const bunkrReferer = "https://stream.bunkr.is/"

type bunkrProvider struct{}

type bunkrPage struct {
	// Props is the top level JSON element
	Props *bunkrProps `json:"props"`
	// BuildID is the Bunkr build ID
	BuildID string `json:"buildId"`
	// IsFallback is true if this is a fallback data structure, and we have to load the real one.
	IsFallback bool `json:"isFallback"`
	// Page is either "/a/[albumId]" or "/v/[videoId]"
	Page string `json:"page"`
}

type bunkrProps struct {
	// PageProps is properties for this album.
	PageProps struct {
		// Album is present when this page is an album.
		Album *bunkrAlbum `json:"album"`
		// File is present when this page is for a single file.
		File *bunkrVideo `json:"file"`
	} `json:"pageProps"`
}

type bunkrAlbum struct {
	// ID is the album ID.
	ID string `json:"identifier"`
	// Name of the album
	Name string `json:"name"`
	// Description of the album
	Description string `json:"description"`
	// Files in this album
	Files []bunkrImage `json:"files"`
}

type bunkrImage struct {
	// Name is the file name of the image.
	Name string `json:"name"`
	// Size of the image.
	Size string `json:"size"`
	// Timestamp is the unix timestamp this file was created at (e.g. 1618941563)
	Timestamp int64 `json:"timestamp"`
	// Status is the status of the request - should be "ok".
	Status string `json:"status"`
	// CDNUrl is the base URL to fetch the image from (e.g. "https://cdn.bunkr.is")
	CDNUrl string `json:"cdn"`
	// RemoteInfo is info about the file, if fetched.
	RemoteInfo *download.RemoteFileInfo
}

type bunkrVideo struct {
	// Name is the file name of the video.
	Name string `json:"name"`
	// Size of the image.
	Size string `json:"size"`
	// MediaFiles is the base URL to fetch the video from (e.g. "https://media-files4.bunkr.is")
	MediaFiles string `json:"mediafiles"`
	// ImageFiles is ??? (e.g. "https://i4.bunkr.is")
	ImageFiles string `json:"imagefiles"`
}

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

	page, err := provider.fetchPage(env, url, albumID, true)
	if err != nil {
		callback(nil, nil, fmt.Errorf("unable to fetch album: %s: %v", url, err))
		return
	}

	provider.parseAlbum(env, url, albumID, page, callback)
}

func (provider bunkrProvider) fetchPage(env *Env, url string, entityID string, findRedirects bool) (*bunkrPage, error) {
	var page *bunkrPage = nil

	resp, err := env.GetWithReferer(url, bunkrReferer)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	tokenizer := html.NewTokenizer(resp.Body)
	for {
		tokenType := tokenizer.Next()
		token := tokenizer.Token()

		err := tokenizer.Err()
		if err == io.EOF {
			return page, nil
		} else if err != nil {
			return nil, err
		}

		switch tokenType {
		case html.StartTagToken:
			switch token.Data {
			case "script":
				scriptID := htmlutils.GetAttr(token.Attr, "id")
				if scriptID == "__NEXT_DATA__" {
					scriptContent, err := provider.getTextContent(tokenizer)
					if err != nil {
						return nil, err
					} else {
						page = &bunkrPage{}
						err := json.Unmarshal([]byte(scriptContent), &page)

						if err != nil {
							return nil, err
						}

						if page.Props.PageProps.Album == nil && page.Props.PageProps.File == nil && page.IsFallback {
							entity := "/a/" + entityID
							if page.Page == "/v/[videoId]" {
								entity = "/v/" + entityID
							} else if page.Page == "/d/[name]" {
								entity = "/d/" + entityID
							}
							props, err := provider.getFallback(env, page.BuildID, entity)
							page.Props = props
							if err != nil {
								return nil, err
							}
						}

						if page.Props.PageProps.Album != nil && findRedirects {
							bunkrAlbum := page.Props.PageProps.Album

							// HEAD each URL. Images sometimes redirect.  Videos will always redirect
							// to an HTML file, and we need to parse that file.
							for index := range bunkrAlbum.Files {
								bunkrImage := &bunkrAlbum.Files[index]
								imageURL := bunkrImage.CDNUrl + "/" + bunkrImage.Name

								// Bunkr media files are actually redirects to a streaming URL.
								fileInfo, err := env.GetFileInfoWithReferer(imageURL, bunkrReferer)
								if err == nil {
									if fileInfo.MimeType == "text/html" {
										// We've been redirected to the video player.
										videoPage, err := provider.fetchPage(env, fileInfo.URL, bunkrImage.Name, false)
										if err == nil && videoPage.Props.PageProps.File != nil {
											video := videoPage.Props.PageProps.File
											bunkrImage.CDNUrl = video.MediaFiles
											bunkrImage.Name = video.Name
										}
									} else {
										bunkrImage.RemoteInfo = fileInfo
									}
								}
							}
						}
					}

				}
			}
		}
	}
}

func (provider bunkrProvider) getFallback(
	env *Env,
	buildID string,
	entity string,
) (*bunkrProps, error) {
	fallbackUrl := "https://bunkr.is/_next/data/" + buildID + entity + ".json"
	resp, err := env.GetWithReferer(fallbackUrl, bunkrReferer)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Unexpected status code %d", resp.StatusCode)
	}

	props := bunkrProps{}
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&props)
	if err != nil {
		return nil, err
	}

	return &props, nil
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
	env *Env,
	url string,
	albumID string,
	page *bunkrPage,
	callback ImageCallback,
) {
	// TODO: Do something useful with warnings.
	warnings := []string{}

	album := meta.AlbumMetadata{
		URL:             url,
		AlbumID:         albumID,
		Name:            "",
		Author:          "",
		TotalImageCount: 0,
	}

	bunkrAlbum := page.Props.PageProps.Album
	album.Name = bunkrAlbum.Name
	album.AlbumID = bunkrAlbum.ID
	album.TotalImageCount = len(bunkrAlbum.Files)

	for index, bunkrImage := range bunkrAlbum.Files {
		t := time.Unix(bunkrImage.Timestamp, 0)
		size, err := strconv.Atoi(bunkrImage.Size)
		if err != nil {
			size = 0
			warnings = append(warnings, "Could not parse image size: "+bunkrImage.Size)
		}

		image := meta.NewImageMetadata(&album, index)
		image.Filename = bunkrImage.Name
		image.Size = int64(size)
		image.Timestamp = &t
		image.URL = bunkrImage.CDNUrl + "/" + bunkrImage.Name
		image.RemoteInfo = bunkrImage.RemoteInfo
		image.Referer = bunkrReferer

		callback(&album, image, nil)
	}

	if len(warnings) != 0 {
		fmt.Println("Warnings:")
		for _, warning := range warnings {
			fmt.Println("  " + warning)
		}
	}
}
