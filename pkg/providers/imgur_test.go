package providers

import (
	"strings"
	"testing"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/jwalton/pixdl/pkg/pixdl/meta"
	"github.com/stretchr/testify/assert"
)

var sample = heredoc.Doc(`
{
	"id": "88wOh",
	"delete_id": "aq4rUDVNqmg18Un",
	"account_id": 1240532,
	"title": "La Machine, Ottawa, 2017 #LaMachine",
	"description": "",
	"view_count": 1534,
	"upvote_count": 0,
	"downvote_count": 0,
	"point_count": 0,
	"image_count": 2,
	"comment_count": 3,
	"favorite_count": 0,
	"virality": 0,
	"score": 0,
	"in_most_viral": false,
	"is_album": true,
	"is_mature": false,
	"cover_id": "wWwA1k6",
	"created_at": "2017-07-31T12:11:05Z",
	"updated_at": null,
	"url": "https://imgur.com/a/88wOh",
	"privacy": "public",
	"vote": null,
	"favorite": false,
	"is_ad": false,
	"ad_type": 0,
	"ad_url": "",
	"include_album_ads": false,
	"shared_with_community": true,
	"is_pending": false,
	"platform": "api",
	"media": [
	  {
		"id": "wWwA1k6",
		"account_id": 1240532,
		"mime_type": "image/jpeg",
		"type": "image",
		"name": "IMG_1364",
		"basename": "",
		"url": "https://i.imgur.com/wWwA1k6.jpeg",
		"ext": "jpeg",
		"width": 4683,
		"height": 3746,
		"size": 2081928,
		"metadata": {
		  "title": "",
		  "description": "Kumo waking up on Sunday.",
		  "is_animated": false,
		  "is_looping": false,
		  "duration": 0,
		  "has_sound": false
		},
		"created_at": "2017-07-31T12:25:20Z",
		"updated_at": null
	  },
	  {
		"id": "7IoXzlA",
		"account_id": 1240532,
		"mime_type": "image/jpeg",
		"type": "image",
		"name": "IMG_1873",
		"basename": "",
		"url": "https://i.imgur.com/7IoXzlA.jpeg",
		"ext": "jpeg",
		"width": 5121,
		"height": 3414,
		"size": 2632628,
		"metadata": {
		  "title": "",
		  "description": "Long Ma waking up on Sunday.",
		  "is_animated": false,
		  "is_looping": false,
		  "duration": 0,
		  "has_sound": false
		},
		"created_at": "2017-07-31T12:25:29Z",
		"updated_at": null
	  }
	],
	"display": []
}`)

func TestImgurRegex(t *testing.T) {
	match := imgurRegex.FindStringSubmatch("https://imgur.com/gallery/88wOh")
	assert.NotNilf(t, match, "Expected https://imgur.com/gallery/88wOh to match")
	albumID := match[2]
	assert.Equal(t, "88wOh", albumID)

	match = imgurRegex.FindStringSubmatch("https://imgur.com/a/88wOh")
	assert.NotNilf(t, match, "Expected https://imgur.com/a/88wOh to match")
	assert.Equal(t, "88wOh", albumID)
}

func TestImgurParseAlbum(t *testing.T) {
	url := "https://imgur.com/a/88wOh"

	images := []*meta.ImageMetadata{}
	var album *meta.AlbumMetadata
	var err error

	callback := func(a *meta.AlbumMetadata, i *meta.ImageMetadata, e error) bool {
		album = a
		if e != nil {
			err = e
		} else if i != nil {
			images = append(images, i)
		}
		return true
	}

	provider := imgurProvider{}
	provider.parseAlbum(url, "88wOh", strings.NewReader(sample), callback)
	assert.Nil(t, err)

	t1, _ := time.Parse("2006-01-02 15:04:05", "2017-07-31 12:25:20")
	t2, _ := time.Parse("2006-01-02 15:04:05", "2017-07-31 12:25:29")

	expectedImages := []*meta.ImageMetadata{
		{
			Album:     album,
			URL:       "https://i.imgur.com/wWwA1k6.jpeg",
			Filename:  "IMG_1364.jpeg",
			Title:     "IMG_1364",
			Size:      2081928,
			Timestamp: &t1,
			Index:     0,
			Page:      1,
		},
		{
			Album:     album,
			URL:       "https://i.imgur.com/7IoXzlA.jpeg",
			Filename:  "IMG_1873.jpeg",
			Title:     "IMG_1873",
			Size:      2632628,
			Timestamp: &t2,
			Index:     1,
			Page:      1,
		},
	}

	assert.Equal(t, url, album.URL)
	assert.Equal(t, "88wOh", album.AlbumID)
	assert.Equal(t, "", album.Author)
	assert.Equal(t, "La Machine, Ottawa, 2017 #LaMachine", album.Name)
	assert.Equal(t, 2, album.TotalImageCount)
	assert.Equal(t, expectedImages, images)
}
