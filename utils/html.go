package utils 

import (
	"bytes"
	"strings"

	"golang.org/x/net/html"
)

// sanitizeHTML удаляет теги <script> и <style>, а также атрибуты, начинающиеся с "on" (inline event handlers),
// и значения href/src с javascript: чтобы избежать вставки JS в результат API.
func SanitizeHTML(input string) string {
	doc, err := html.Parse(strings.NewReader(input))
	if err != nil {
		return input
	}

	var clone func(*html.Node) *html.Node
	clone = func(n *html.Node) *html.Node {
		// пропускаем script и style
		if n.Type == html.ElementNode {
			switch strings.ToLower(n.Data) {
			case "script", "style":
				return nil
			}
		}

		out := &html.Node{Type: n.Type, DataAtom: n.DataAtom, Data: n.Data}
		// копируем и фильтруем атрибуты
		if len(n.Attr) > 0 {
			for _, a := range n.Attr {
				key := strings.ToLower(a.Key)
				if strings.HasPrefix(key, "on") {
					continue
				}
				if (key == "href" || key == "src") && strings.HasPrefix(strings.ToLower(a.Val), "javascript:") {
					continue
				}
				out.Attr = append(out.Attr, a)
			}
		}

		// рекурсивно копируем детей
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if child := clone(c); child != nil {
				child.Parent = out
				if out.FirstChild == nil {
					out.FirstChild = child
					out.LastChild = child
				} else {
					out.LastChild.NextSibling = child
					child.PrevSibling = out.LastChild
					out.LastChild = child
				}
			}
		}
		return out
	}

	cleaned := clone(doc)
	var b bytes.Buffer
	if cleaned == nil {
		return ""
	}
	html.Render(&b, cleaned)
	return b.String()
}
