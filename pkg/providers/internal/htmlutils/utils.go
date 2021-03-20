package htmlutils

import (
	"fmt"
	"io"
	"strconv"

	"golang.org/x/net/html"
)

// GetTokenAttr returns the value for an attribute in a token, or "" if no such
// attribute is present.
func GetTokenAttr(t html.Token, attrName string) string {
	for index := range t.Attr {
		if t.Attr[index].Key == attrName {
			return t.Attr[index].Val
		}
	}

	return ""
}

func GetAttrMap(t html.Token) map[string]string {
	result := make(map[string]string, len(t.Attr))
	for index := range t.Attr {
		key := t.Attr[index].Key
		val := t.Attr[index].Val
		result[key] = val
	}
	return result
}

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
			fmt.Printf("%d: Skipping %s\n", len(stack), token.Data)
			if token.Data != "br" && token.Data != "img" {
				stack = append(stack, token.Data)
			}
		case html.SelfClosingTagToken:
			fmt.Printf("%d: Self %s\n", len(stack), token.Data)
		case html.EndTagToken:
			fmt.Printf("%d: End %s\n", len(stack), token.Data)
			stackLen := len(stack)
			if stack[stackLen-1] == token.Data {
				stack = stack[0 : stackLen-1]
			} else {
				// This will happen if there's a spurious close tag in the
				// source we're reading.
				fmt.Printf("Unexpected %s\n", token.Data)
			}
		}
	}

	return nil
}
