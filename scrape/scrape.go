package scrape

import (
	"context"
	"encoding/base64"
	"errors"
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
		var screenshot []byte
		if err := chromedp.Run(ctx, chromedp.FullScreenshot(&screenshot, 100)); err != nil {
			return resp, errors.New("Screenshot: " + err.Error())
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