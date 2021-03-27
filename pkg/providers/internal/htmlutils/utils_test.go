package htmlutils

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestGetNodeTextContent(t *testing.T) {
	htmlSource := `<html><body><div>Hello <span>world</span></div></body></html>`

	doc, _ := html.Parse(strings.NewReader(htmlSource))
	result := GetNodeTextContent(doc)
	if result != "Hello world" {
		t.Errorf("Expected 'Hello world', got: '%s'", result)
	}
}
