package providers

import (
	"strings"
	"testing"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/jwalton/pixdl/pkg/pixdl/meta"
	"github.com/stretchr/testify/assert"
)

func TestCyberdropRegex(t *testing.T) {
	match := cyberdropRegex.FindStringSubmatch("https://cyberdrop.me/a/abcdef")
	if match == nil {
		t.Error("Expected URL to match")
		return
	}

	albumID := match[2]
	if albumID != "abcdef" {
		t.Errorf("Expected albumID to be %v but got %v", "abcdef", albumID)
	}
}

func TestCyberdropParseAlbum(t *testing.T) {
	sampleAlbum := heredoc.Doc(`
		<!DOCTYPE html>
		<html lang="en">
		<body>
		<section class="section hero is-fullheight">
			<div class="container">

				<div class="divider is-info">
				<h1 id="title" class="title has-text-centered" title="My Album">
					My Album
				</h1>
				</div>
				<h1 id="count" class="subtitle is-hidden-desktop has-text-centered">
					2 files (30 MB)<br>
				</h1>

				<div class="subtitle description has-text-centered" title="by Jason" style="margin-bottom:0px;">
					<!--<span>by Jason</span>-->
					<span id="description-box">by Jason</span>
				</div>

				<div id="table" class="columns is-multiline is-mobile is-centered has-text-centered">
					<div class="image-container column">
						<a class="image" href="https://f.cyberdrop.cc/image1-abcdef.jpg" target="_blank" title="image1.jpg" rel="noopener noreferrer" data-html="#1616045815" data-timestamp="1616045815" data-src="https://f.cyberdrop.cc/s/image1-abcdef.jpg" >
							<img alt="image1-abcdef.jpg" data-src="https://i0.wp.com/cyberdrop.me/thumbs/image1-abcdef.png" src="https://i0.wp.com/cyberdrop.me/thumbs/image1-abcdef.png" loading="lazy" />
						</a>
						<div class="details">
							<p><span class="name"><a id="file" href="https://f.cyberdrop.cc/image1-abcdef.jpg" target="_blank" title="image1.jpg" rel="noopener noreferrer">image1.jpg</a></span></p>
							<p class="is-hidden file-size">10000000 B</p>
							<p>10.00 MB</p>
						</div>
					</div>

					<div class="image-container column">
						<a class="image" href="https://f.cyberdrop.cc/image2-efg hij.jpg" target="_blank" title="image2.jpg" rel="noopener noreferrer" data-html="#1616045814" data-timestamp="1616045814" data-src="https://f.cyberdrop.cc/s/image2-efg hij.jpg" >
							<img alt="image2-efg hij.jpg" data-src="https://i0.wp.com/cyberdrop.me/thumbs/image2-efg hij.png" src="https://i0.wp.com/cyberdrop.me/thumbs/image2-efg hij.png" loading="lazy" />
						</a>
						<div class="details">
							<p><span class="name"><a id="file" href="https://f.cyberdrop.cc/image2-efg hij.jpg" target="_blank" title="image2.jpg" rel="noopener noreferrer">image2.jpg</a></span></p>
							<p class="is-hidden file-size">20000000 B</p>
							<p>20.00 MB</p>
						</div>
					</div>
				</div>
			</div>
		</body>
		</html>
	`)

	url := "https://cyberdrop.me/a/test"

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

	provider := cyberdropProvider{}
	provider.parseAlbum(url, "test", strings.NewReader(sampleAlbum), callback)
	assert.Nil(t, err)

	assert.Equal(t, url, album.URL)
	assert.Equal(t, "test", album.AlbumID)
	assert.Equal(t, "Jason", album.Author)
	assert.Equal(t, "My Album", album.Name)
	assert.Equal(t, 2, album.TotalImageCount)

	t1 := time.Unix(1616045815, 0)
	t2 := time.Unix(1616045814, 0)
	expectedImages := []*meta.ImageMetadata{
		{
			Album:     album,
			URL:       "https://f.cyberdrop.cc/image1-abcdef.jpg",
			Filename:  "image1.jpg",
			Title:     "image1.jpg",
			Size:      10000000,
			Timestamp: &t1,
			Index:     0,
			Page:      1,
		},
		{
			Album:     album,
			URL:       "https://f.cyberdrop.cc/image2-efg hij.jpg",
			Filename:  "image2.jpg",
			Title:     "image2.jpg",
			Size:      20000000,
			Timestamp: &t2,
			Index:     1,
			Page:      1,
		},
	}

	assert.Equal(t, expectedImages, images)
}
