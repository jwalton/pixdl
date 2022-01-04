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
	"data": {
	  "image": {
		"id": 270373373,
		"hash": "88wOh",
		"account_id": "1240532",
		"account_url": "jwalton",
		"title": "La Machine, Ottawa, 2017 #LaMachine",
		"score": 37,
		"starting_score": 0,
		"virality": 4618.459513,
		"size": 0,
		"views": "1530",
		"is_hot": false,
		"is_album": true,
		"album_cover": "wWwA1k6",
		"album_cover_width": 4683,
		"album_cover_height": 3746,
		"mimetype": null,
		"ext": ".jpg",
		"width": 0,
		"height": 0,
		"animated": false,
		"looping": false,
		"ups": 40,
		"downs": 3,
		"points": 37,
		"reddit": null,
		"description": "",
		"bandwidth": null,
		"timestamp": "2017-08-01 15:08:29",
		"hot_datetime": null,
		"gallery_datetime": "2017-07-31 12:11:05",
		"in_gallery": true,
		"section": "",
		"tags": ["0", "0"],
		"subtype": null,
		"spam": 0,
		"pending": "0",
		"comment_count": 3,
		"nsfw": false,
		"topic": null,
		"topic_id": 0,
		"meme_name": null,
		"meme_top": null,
		"meme_bottom": null,
		"prefer_video": false,
		"video_source": null,
		"video_host": null,
		"num_images": 9,
		"platform": null,
		"readonly": false,
		"ad_type": 0,
		"ad_url": "",
		"weight": -1,
		"favorite_count": 11,
		"has_sound": false,
		"album_privacy": "0",
		"album_layout": "b",
		"album_images": {
		  "count": 2,
		  "images": [
			{
			  "hash": "wWwA1k6",
			  "title": "",
			  "description": "Kumo waking up on Sunday.",
			  "has_sound": false,
			  "width": 4683,
			  "height": 3746,
			  "size": 2081928,
			  "ext": ".jpg",
			  "animated": false,
			  "prefer_video": false,
			  "looping": false,
			  "datetime": "2017-07-31 12:25:20",
			  "name": "IMG_1364",
			  "edited": "0"
			},
			{
			  "hash": "7IoXzlA",
			  "title": "",
			  "description": "Long Ma waking up on Sunday.",
			  "has_sound": false,
			  "width": 5121,
			  "height": 3414,
			  "size": 2632628,
			  "ext": ".jpg",
			  "animated": false,
			  "prefer_video": false,
			  "looping": false,
			  "datetime": "2017-07-31 12:25:29",
			  "name": "IMG_1873",
			  "edited": "0"
			}
		  ]
		},
		"favorited": false,
		"adConfig": {
		  "safeFlags": ["page_load", "in_gallery", "album"],
		  "highRiskFlags": [],
		  "unsafeFlags": ["sixth_mod_unsafe"],
		  "wallUnsafeFlags": [],
		  "showsAds": false
		},
		"vote": null
	  }
	},
	"success": true,
	"status": 200
  }
`)

func TextImgurRegex(t *testing.T) {
	match := imgurRegex.FindStringSubmatch("https://imgur.com/gallery/88wOh")
	if match == nil {
		t.Error("Expected URL to match")
		return
	}

	albumID := match[2]
	if albumID != "88wOh" {
		t.Errorf("expected albumID to be %v but got %v", "88wOh", albumID)
	}
}

func TestImgurParseAlbum(t *testing.T) {
	url := "https://imgur.com/gallery/88wOh"

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
			URL:       "https://i.imgur.com/wWwA1k6.jpg",
			Filename:  "IMG_1364.jpg",
			Title:     "",
			Size:      2081928,
			Timestamp: &t1,
			Index:     0,
			Page:      1,
		},
		{
			Album:     album,
			URL:       "https://i.imgur.com/7IoXzlA.jpg",
			Filename:  "IMG_1873.jpg",
			Title:     "",
			Size:      2632628,
			Timestamp: &t2,
			Index:     1,
			Page:      1,
		},
	}

	assert.Equal(t, url, album.URL)
	assert.Equal(t, "88wOh", album.AlbumID)
	assert.Equal(t, "jwalton", album.Author)
	assert.Equal(t, "La Machine, Ottawa, 2017 #LaMachine", album.Name)
	assert.Equal(t, 2, album.TotalImageCount)
	assert.Equal(t, expectedImages, images)
}
