package htmlutils

import (
	"io"
	"net/url"
	"path"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

// TODO: Write test cases for this file.

// GetNumericAttrFromMapWithDefault parses an attribute to a number, and returns
// the value, or returns `def` if the value is not present or cannot be parsed.
func GetNumericAttrFromMapWithDefault(attrMap map[string]string, attrName string, def int64) int64 {
	valStr, ok := attrMap[attrName]
	if !ok {
		return def
	}
	val, err := strconv.ParseInt(valStr, 10, 64)
	if err != nil {
		return def
	}
	return val
}

// SkipTokenContents can be used when encountering a `html.StartTagToken` to
// read up until the matching `html.EndTagToken`, discarding everything in
// the middle.
func SkipTokenContents(tokenizer *html.Tokenizer, tokenType string) error {
	stack := []string{tokenType}
	for len(stack) > 0 {
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
			if token.Data != "br" && token.Data != "img" {
				stack = append(stack, token.Data)
			}
		case html.EndTagToken:
			stackLen := len(stack)
			if stack[stackLen-1] == token.Data {
				stack = stack[0 : stackLen-1]
			} else {
				// This will happen if there's a spurious close tag in the
				// source we're reading.
			}
		}
	}

	return nil
}

// GetAttr returns the value for an attribute, or "" if no such attribute is present.
func GetAttr(attributes []html.Attribute, attrName string) string {
	for index := range attributes {
		if attributes[index].Key == attrName {
			return attributes[index].Val
		}
	}

	return ""
}

// GetAttrMap returns a map of attributes by name.
func GetAttrMap(attributes []html.Attribute) map[string]string {
	result := make(map[string]string, len(attributes))
	for index := range attributes {
		key := attributes[index].Key
		val := attributes[index].Val
		result[key] = val
	}
	return result
}

// HasClass returns true if the given element has the specified class.
func HasClass(attributes []html.Attribute, className string) bool {
	classAttr := GetAttr(attributes, "class")
	return classAttr != "" && (classAttr == className ||
		strings.HasPrefix(classAttr, className+" ") ||
		strings.HasSuffix(classAttr, " "+className) ||
		strings.Contains(classAttr, " "+className+" "))
}

// FindNodeByID searches the tree rooted at "node" for a node with the "id"
// attribute with the specified value.  Returns the node, or nil if no
// such node is found.
func FindNodeByID(node *html.Node, id string, maxDepth int) *html.Node {
	if GetAttr(node.Attr, "id") == id {
		return node
	}
	if maxDepth == 1 {
		return nil
	}
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		cResult := FindNodeByID(c, id, maxDepth-1)
		if cResult != nil {
			return cResult
		}
	}
	return nil
}

// WalkNodesPreOrder calls `walker` on each node in pre-order.  If `walker` returns
// false, the the given node's children will be skipped.
func WalkNodesPreOrder(node *html.Node, walker func(*html.Node) bool) {
	var f func(*html.Node)
	f = func(node *html.Node) {
		traverseChildren := walker(node)
		if traverseChildren {
			for c := node.FirstChild; c != nil; c = c.NextSibling {
				f(c)
			}
		}
	}
	f(node)
}

// GetNodeTextContent returns the text content of a node.
func GetNodeTextContent(node *html.Node) string {
	result := strings.Builder{}

	WalkNodesPreOrder(node, func(node *html.Node) bool {
		if node.Type == html.TextNode {
			result.WriteString(node.Data)
		}
		return true
	})

	return result.String()
}

// ResolveURL resolves a URL relative to a parsed URL.
// For example, calling `ResolveURL("http://foo.com/bar", "../baz")` would return
// "http://foo.com/baz".
func ResolveURL(baseURL *url.URL, relativeURL string) string {
	if strings.HasPrefix(relativeURL, "https://") || strings.HasPrefix(relativeURL, "http://") {
		return relativeURL
	}

	resolvedURL := *baseURL
	if path.IsAbs(relativeURL) {
		resolvedURL.Path = relativeURL
	} else {
		resolvedURL.Path = path.Join(path.Dir(baseURL.Path), relativeURL)
	}
	return resolvedURL.String()
}
