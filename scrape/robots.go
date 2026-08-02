package scrape

import (
	"bufio"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type RobotsRule struct {
	UserAgent  string
	Allowed    []string
	Disallowed []string
	CrawlDelay time.Duration
	Sitemap    []string
}

type RobotsParser struct {
	Rules   []RobotsRule
	BaseURL *url.URL
}

func NewRobotsParser(baseUrl string) (*RobotsParser, error) {
	u, err := url.Parse(baseUrl)
	if err != nil {
		return nil, err
	}
	return &RobotsParser{
		Rules:   make([]RobotsRule, 0),
		BaseURL: u,
	}, nil
}

func (rp *RobotsParser) Fetch() error {
	robotsUrl := rp.BaseURL.Scheme + "://" + rp.BaseURL.Host + "/robots.txt"

	resp, err := http.Get(robotsUrl)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil // No robots.txt found, treat as empty
	}

	return rp.Parse(resp.Body)
}

func (rp *RobotsParser) Parse(body io.Reader) error {
	scanner := bufio.NewScanner(body)
	var currentRule *RobotsRule

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		directive := strings.ToLower(strings.TrimSpace(parts[0]))
		value := strings.TrimSpace(parts[1])

		switch directive {
		case "user-agent":
			if currentRule != nil {
				rp.Rules = append(rp.Rules, *currentRule)
			}
			currentRule = &RobotsRule{
				UserAgent:  value,
				Allowed:    make([]string, 0),
				Disallowed: make([]string, 0),
				Sitemap:    make([]string, 0),
			}
		case "disallow":
			if currentRule != nil && value != "" {
				currentRule.Disallowed = append(currentRule.Disallowed, value)
			}
		case "allow":
			if currentRule != nil && value != "" {
				currentRule.Allowed = append(currentRule.Allowed, value)
			}
		case "crawl-delay":
			if currentRule != nil {
				if delay, err := strconv.Atoi(value); err == nil {
					currentRule.CrawlDelay = time.Duration(delay) * time.Second
				}
			}
		case "sitemap":
			if currentRule != nil && value != "" {
				currentRule.Sitemap = append(currentRule.Sitemap, value)
			}
		}
	}

	if currentRule != nil {
		rp.Rules = append(rp.Rules, *currentRule)
	}

	return scanner.Err()
}

func (rp *RobotsParser) IsAllowed(userAgent, urlPath string) bool {
	var applicableRule *RobotsRule

	for _, rule := range rp.Rules {
		if rule.UserAgent == "*" && applicableRule == nil {
			applicableRule = &rule
		} else if strings.EqualFold(rule.UserAgent, userAgent) {
			applicableRule = &rule
			break
		}
	}

	if applicableRule == nil {
		return true // No rules, allow by default
	}

	for _, disallowed := range applicableRule.Disallowed {
		if mathesPattern(urlPath, disallowed) {
			return false
		}
	}

	return true // Not explicitly disallowed
}

func mathesPattern(path, pattern string) bool {
	if pattern == "/" {
		return true
	}

	// Convert robots.txt pattern to regular expression
	pattern = regexp.QuoteMeta(pattern)
	pattern = strings.ReplaceAll(pattern, "\\*", ".*")
	pattern = "^" + pattern

	matched, _ := regexp.MatchString(pattern, path)
	return matched
}

func (rp *RobotsParser) GetCrawlDelay(userAgent string) time.Duration {
	for _, rule := range rp.Rules {
		if strings.EqualFold(rule.UserAgent, userAgent) ||
			rule.UserAgent == "*" {
			return rule.CrawlDelay
		}
	}
	return 0
}