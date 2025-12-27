package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"math"
	"net/http"
	"net/url"
	"time"
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
		// небольшой сон, чтобы дать странице подгрузиться JS/ресурсы и ленивую загрузку
		//chromedp.Sleep(500*time.Millisecond),
		chromedp.EvaluateAsDevTools(`Array.from(document.querySelectorAll('img')).map(img => img.src)`, &imageSrcs),
		chromedp.EvaluateAsDevTools(`Array.from(document.querySelectorAll('a[href]')).map(a => a.href)`, &linkHrefs),
	)

	if err != nil {
		http.Error(w, fmt.Sprintf("Scraping: %v", err), http.StatusInternalServerError)
		return
	}

	// Попытка корректно сделать скриншот длинной страницы: делаем серию скриншотов видимой области и склеиваем
	screenshot, err = captureFullPageScreenshot(ctx)
	if err != nil {
		http.Error(w, fmt.Sprintf("Screenshot Error: %v", err), http.StatusInternalServerError)
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

// captureFullPageScreenshot делает серию снимков видимой области страницы, прокручивая её,
// и склеивает полученные PNG-изображения в один большой скриншот полной высоты.
func captureFullPageScreenshot(ctx context.Context) ([]byte, error) {
	var pageHeightF, viewportHeightF float64
	if err := chromedp.Run(ctx,
		chromedp.Evaluate(`Math.max(document.body.scrollHeight, document.documentElement.scrollHeight)`, &pageHeightF),
		chromedp.Evaluate(`window.innerHeight`, &viewportHeightF),
	); err != nil {
		return nil, err
	}

	pageHeight := int(pageHeightF)
	viewportHeight := int(viewportHeightF)

	if pageHeight <= 0 || viewportHeight <= 0 {
		// fallback: попробуем стандартный full screenshot
		var buf []byte
		if err := chromedp.Run(ctx, chromedp.FullScreenshot(&buf, 100)); err != nil {
			return nil, err
		}
		return buf, nil
	}

	steps := int(math.Ceil(float64(pageHeight) / float64(viewportHeight)))
	imgs := make([][]byte, 0, steps)

	for i := 0; i < steps; i++ {
		y := i * viewportHeight
		if err := chromedp.Run(ctx,
			chromedp.Evaluate(fmt.Sprintf(`window.scrollTo(0, %d)`, y), nil),
			chromedp.Sleep(200*time.Millisecond),
		); err != nil {
			return nil, err
		}

		var buf []byte
		if err := chromedp.Run(ctx, chromedp.CaptureScreenshot(&buf)); err != nil {
			return nil, err
		}
		imgs = append(imgs, buf)
	}

	// Декодируем и склеиваем изображения
	decoded := make([]image.Image, 0, len(imgs))
	totalHeight := 0
	width := 0
	for _, b := range imgs {
		img, err := png.Decode(bytes.NewReader(b))
		if err != nil {
			return nil, err
		}
		decoded = append(decoded, img)
		if width == 0 {
			width = img.Bounds().Dx()
		}
		totalHeight += img.Bounds().Dy()
	}

	targetHeight := pageHeight
	if targetHeight > totalHeight {
		targetHeight = totalHeight
	}

	dst := image.NewRGBA(image.Rect(0, 0, width, targetHeight))
	curY := 0
	for _, img := range decoded {
		h := img.Bounds().Dy()
		drawH := h
		if curY+drawH > targetHeight {
			drawH = targetHeight - curY
		}
		draw.Draw(dst, image.Rect(0, curY, width, curY+drawH), img, image.Point{0, 0}, draw.Src)
		curY += drawH
		if curY >= targetHeight {
			break
		}
	}

	var out bytes.Buffer
	if err := png.Encode(&out, dst); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
