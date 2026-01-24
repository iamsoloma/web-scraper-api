package scrape

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"math"
	"net/url"
	"time"
	"web-scraper-api/utils"

	"github.com/chromedp/chromedp"
	md "github.com/firecrawl/html-to-markdown"
	"github.com/firecrawl/html-to-markdown/plugin"
)

func Scrape(req ScrapeRequest, userAgent string) (resp ScraperResponse, err error) {
	requrl, err := url.Parse(req.URL)
	if err != nil {
		return resp, errors.New("URL parse: " + err.Error())
	}

	// Создаем контекст chromedp с опциями для Docker
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.UserAgent(userAgent),
		//chromedp.UserDataDir("./data"),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	// Создаем контекст браузера
	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	var Actions []chromedp.Action
	Actions = append(Actions,
		chromedp.Navigate(req.URL),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.Title(&resp.Title),
		chromedp.OuterHTML("html", &resp.HTML, chromedp.ByQueryAll),
	)
	if req.Images {
		Actions = append(Actions, chromedp.EvaluateAsDevTools(`Array.from(document.querySelectorAll('img')).reduce((m, img) => {
			try {
				const src = img.src || '';
				const name = (img.alt || img.title || (new URL(src, location.href).pathname.split('/').pop()) || src).toString().trim();
				const key = name || src;
				m[key] = src;
			} catch (e) {
				// ignore
			}
			return m;
		}, {})`, &resp.Images))
	}
	if req.Links {
		Actions = append(Actions, chromedp.EvaluateAsDevTools(`(function(){
			const m = {};
			const curr = new URL(location.href);
			Array.from(document.querySelectorAll('a[href]')).forEach(a => {
				try {
					const href = a.href || '';
					const resolved = new URL(href, location.href);
					// исключаем ссылки, ведущие на ту же страницу (игнорируем фрагмент)
					if (resolved.href.split('#')[0] === curr.href.split('#')[0]) return;
					const name = (a.textContent || a.getAttribute('title') || href).toString().trim();
					const key = name || href;
					if (!(key in m)) m[key] = resolved.href;
				} catch (e) {
					// ignore malformed URLs
				}
			});
			return m;
		})()`, &resp.Links))
	}
	if req.Text {
		Actions = append(Actions, chromedp.EvaluateAsDevTools("(function() { return document.body.innerText; })()", &resp.Text))
	}

	// Выполняем задачи в браузере
	err = chromedp.Run(ctx, Actions...)
	if err != nil {
		return resp, errors.New("Scraping: " + err.Error())
	}

	if req.Screenshot == true {
		// Попытка корректно сделать скриншот длинной страницы: делаем серию скриншотов видимой области и склеиваем
		screenshot, err := captureFullPageScreenshot(ctx)
		if err != nil {
			return resp, errors.New("Screenshot Error: " + err.Error())
		}
		resp.Screenshot = base64.StdEncoding.EncodeToString(screenshot)
	}

	if req.Markdown {
		// Очищаем HTML от скриптов и потенциального inline-JS
		sanitizedHTML := utils.SanitizeHTML(resp.HTML)

		conv := md.NewConverter(requrl.Hostname(), true, nil)
		conv.Use(plugin.GitHubFlavored())
		markdown, err := conv.ConvertString(sanitizedHTML)
		if err != nil {
			return resp, errors.New("Markdown conversion: " + err.Error())
		}
		resp.Markdown = markdown
	}

	if !req.HTML {
		resp.HTML = ""
	}

	resp.Date = time.Now().UTC()
	return resp, nil

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

	if pageHeight <= viewportHeight {
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
