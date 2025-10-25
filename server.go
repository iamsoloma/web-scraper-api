package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

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
	fmt.Printf("Сервер запущен на порту %s
", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func scrapeHandler(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	if url == "" {
		http.Error(w, "Параметр url обязателен", http.StatusBadRequest)
		return
	}

	// Создаем контекст chromedp с опциями для Docker
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.ExecPath("/usr/bin/chromium-browser"),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	// Создаем контекст браузера
	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Устанавливаем таймаут
	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var title, text string
	var screenshot []byte
	var imageSrcs, linkHrefs []string

	// Выполняем задачи в браузере
	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.Title(&title),
		chromedp.Text("body", &text, chromedp.ByQuery),
		chromedp.Screenshot("body", &screenshot, chromedp.ByQuery),
		chromedp.EvaluateAsDevTools(`Array.from(document.querySelectorAll('img')).map(img => img.src)`, &imageSrcs),
		chromedp.EvaluateAsDevTools(`Array.from(document.querySelectorAll('a[href]')).map(a => a.href)`, &linkHrefs),
	)

	if err != nil {
		http.Error(w, fmt.Sprintf("Ошибка при скрапинге: %v", err), http.StatusInternalServerError)
		return
	}

	response := ScraperResponse{
		Title:      title,
		Screenshot: base64.StdEncoding.EncodeToString(screenshot),
		Text:       text,
		Images:     imageSrcs,
		Links:      linkHrefs,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}