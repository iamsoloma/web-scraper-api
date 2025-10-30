package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/chromedp/chromedp"
)

// ScraperResponse структура для ответа API
type ScraperResponse struct {
	Title      string   `json:"title"`
	Screenshot string   `json:"screenshot"`
	Text       string   `json:"text"`
	Images     []string `json:"images"`
	Links      []string `json:"links"`
}

func main() {
	http.HandleFunc("/scrape", scrapeHandler)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Printf("Server is started on %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func scrapeHandler(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	if url == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
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

	var title, text string
	var screenshot []byte
	var imageSrcs, linkHrefs []string

	// Выполняем задачи в браузере
	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.Title(&title),
		chromedp.Text("body", &text, chromedp.ByQuery),
		chromedp.FullScreenshot(&screenshot, 100),
		chromedp.EvaluateAsDevTools(`Array.from(document.querySelectorAll('img')).map(img => img.src)`, &imageSrcs),
		chromedp.EvaluateAsDevTools(`Array.from(document.querySelectorAll('a[href]')).map(a => a.href)`, &linkHrefs),
	)

	if err != nil {
		http.Error(w, fmt.Sprintf("Scraping: %v", err), http.StatusInternalServerError)
		return
	}

	response := ScraperResponse{
		Title:      title,
		Screenshot: base64.StdEncoding.EncodeToString(screenshot),
		Text:       text,
		Images:     imageSrcs,
		Links:      linkHrefs,
	}

	fmt.Println(text)
	err = os.WriteFile("screen.png", screenshot, 0666)
	if err != nil {
		fmt.Println(err)
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, fmt.Sprintf("JSON: %v", err), http.StatusInternalServerError)
		return
	}
}
