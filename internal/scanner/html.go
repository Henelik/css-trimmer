package scanner

import (
	"io"
	"strings"

	"golang.org/x/net/html"
)

// ExtractHTMLClasses scans an HTML file and returns all found class names.
func ExtractHTMLClasses(content io.Reader) ([]string, error) {
	var classes []string
	classSet := make(map[string]bool)

	tokenizer := html.NewTokenizer(content)

	for {
		tt := tokenizer.Next()
		switch tt {
		case html.ErrorToken:
			err := tokenizer.Err()
			if err == io.EOF {
				return classes, nil
			}
			return nil, err

		case html.StartTagToken, html.SelfClosingTagToken:
			t := tokenizer.Token()

			// Look for the class attribute
			for _, a := range t.Attr {
				if a.Key == "class" {
					// Split on whitespace and add each class
					parts := strings.Fields(a.Val)
					for _, part := range parts {
						if part != "" && !classSet[part] {
							classes = append(classes, part)
							classSet[part] = true
						}
					}
					break
				}
			}
		}
	}
}
