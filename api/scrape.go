package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"web-scraper-api/utils"

	"github.com/chromedp/chromedp"
	md "github.com/firecrawl/html-to-markdown"
	"github.com/firecrawl/html-to-markdown/plugin"
)

type ScraperResponse struct {
	Title      string   `json:"title"`
	Screenshot string   `json:"screenshot"`
	HTML       string   `json:"html"`
	Markdown   string   `json:"markdown"`
	Text       string   `json:"text"`
	Images     []string `json:"images"`
	Links      []string `json:"links"`
}

func (s_ *Server) Scrape(w http.ResponseWriter, r *http.Request) {
	requrl := r.URL.Query().Get("url")
	if requrl == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	//Вытаскиваем доменное из URL
	url, err := url.Parse(requrl)
	if err != nil {
		http.Error(w, fmt.Sprintf("URL parse: %v", err), http.StatusBadRequest)
		return
	}

	// Создаем контекст chromedp с опциями для Docker
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	// Создаем контекст браузера
	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Устанавливаем таймаут
	//ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	//defer cancel()

	var title, html, text string
	var screenshot []byte
	var imageSrcs, linkHrefs []string

	// Выполняем задачи в браузере
	err = chromedp.Run(ctx,
		chromedp.Navigate(requrl),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.Title(&title),
		chromedp.OuterHTML("html", &html, chromedp.ByQueryAll),
		chromedp.Text("body", &text, chromedp.ByQuery),
		//chromedp.FullScreenshot(&screenshot, 100),
		chromedp.EvaluateAsDevTools(`Array.from(document.querySelectorAll('img')).map(img => img.src)`, &imageSrcs),
		chromedp.EvaluateAsDevTools(`Array.from(document.querySelectorAll('a[href]')).map(a => a.href)`, &linkHrefs),
	)

	if err != nil {
		http.Error(w, fmt.Sprintf("Scraping: %v", err), http.StatusInternalServerError)
		return
	}

	// Очищаем HTML от скриптов и потенциального inline-JS
	sanitizedHTML := utils.SanitizeHTML(html)

	conv := md.NewConverter(url.Hostname(), true, nil)
	conv.Use(plugin.GitHubFlavored())
	markdown, err := conv.ConvertString(sanitizedHTML)
	if err != nil {
		http.Error(w, fmt.Sprintf("Markdown conversion: %v", err), http.StatusInternalServerError)
		return
	}

	response := ScraperResponse{
		Title:      title,
		Screenshot: base64.StdEncoding.EncodeToString(screenshot),
		HTML:       sanitizedHTML,
		Markdown:   markdown,
		Text:       text,
		Images:     imageSrcs,
		Links:      linkHrefs,
	}

	/*err = os.WriteFile("md.txt", []byte(markdown), 0666)
	if err != nil {
		fmt.Println(err)
	}
	err = os.WriteFile("screen.png", screenshot, 0666)
	if err != nil {
		fmt.Println(err)
	}*/

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, fmt.Sprintf("JSON: %v", err), http.StatusInternalServerError)
		return
	}
}
