package download

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseContentType(t *testing.T) {
	assert.Equal(t, "image/png", parseContentType("image/png"))
	assert.Equal(t, "image/svg+xml", parseContentType("image/svg+xml; charset=utf-8"))
	assert.Equal(t, "", parseContentType("blat"))
	assert.Equal(t, "", parseContentType(""))
}
