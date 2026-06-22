package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"web-scraper-api/utils"

	"github.com/PuerkitoBio/goquery"
)

func main() {

	resp, err := http.Get("https://lenta.ru/articles/2026/06/22/brothers/")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	html := utils.SanitizeHTML(string(body))

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	text := doc.Text()
	fmt.Println(text)
}
