package providers

import (
	"fmt"
	"strings"

	"github.com/jwalton/pixdl/internal/htmlutils"
	"github.com/jwalton/pixdl/pkg/pixdl/meta"
)

type pixhostToProvider struct{}

func (provider pixhostToProvider) Name() string {
	return "pixhost.to"
}

func (provider pixhostToProvider) CanFetchImage(url string) bool {
	return strings.HasPrefix(url, "https://pixhost.to/show/")
}

func (provider pixhostToProvider) FetchImage(
	env *Env,
	params map[string]string,
	album *meta.AlbumMetadata,
	url string,
) (image *meta.ImageMetadata, err error) {
	node, err := env.GetHTML(url)
	if err != nil {
		return nil, err
	}

	imageNode := htmlutils.FindNodeByID(node, "image", 100)
	if imageNode == nil {
		return nil, fmt.Errorf("image not found")
	}

	image = meta.NewImageMetadata(album, 0)
	attrs := htmlutils.GetAttrMap(imageNode.Attr)

	image.Filename = attrs["alt"]
	image.URL = attrs["src"]

	return image, nil
}
