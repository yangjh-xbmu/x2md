package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const (
	fxTwitterBase = "https://api.fxtwitter.com"
	userAgent     = "x2md/1.0"
	httpTimeout   = 30 * time.Second
)

var (
	// Matches: x.com/{user}/status/{id}, twitter.com/{user}/status/{id},
	// fxtwitter.com/{user}/status/{id}, fixupx.com/{user}/status/{id}
	tweetURLPattern = regexp.MustCompile(
		`(?:https?://)?(?:www\.)?(?:x\.com|twitter\.com|fxtwitter\.com|fixupx\.com)/([^/]+)/status/(\d+)`,
	)
	// Matches: x.com/{user}/article/{id} or x.com/i/article/{id}
	articleURLPattern = regexp.MustCompile(
		`(?:https?://)?(?:www\.)?(?:x\.com|twitter\.com|fxtwitter\.com|fixupx\.com)/([^/]+)/article/(\d+)`,
	)
)

// ParseURL parses a tweet or article URL and returns structured info.
func ParseURL(rawURL string) (URLInfo, error) {
	rawURL = strings.TrimSpace(rawURL)

	if m := articleURLPattern.FindStringSubmatch(rawURL); m != nil {
		return URLInfo{
			Type:        URLTypeArticle,
			ScreenName:  m[1],
			ID:          m[2],
			OriginalURL: normalizeOriginalURL(m[1], "article", m[2]),
		}, nil
	}

	if m := tweetURLPattern.FindStringSubmatch(rawURL); m != nil {
		return URLInfo{
			Type:        URLTypeTweet,
			ScreenName:  m[1],
			ID:          m[2],
			OriginalURL: normalizeOriginalURL(m[1], "status", m[2]),
		}, nil
	}

	return URLInfo{}, fmt.Errorf("unsupported URL format: %s", rawURL)
}

// normalizeOriginalURL converts any variant URL to a canonical x.com URL.
func normalizeOriginalURL(screenName, pathType, id string) string {
	return fmt.Sprintf("https://x.com/%s/%s/%s", screenName, pathType, id)
}

// FetchTweet fetches a single tweet from FxTwitter API.
func FetchTweet(screenName, id string) (*Tweet, error) {
	url := fmt.Sprintf("%s/%s/status/%s", fxTwitterBase, screenName, id)
	return fetchAndParse(url)
}

// FetchArticle fetches an article from FxTwitter API.
func FetchArticle(screenName, id string) (*Tweet, error) {
	// Try with screen name first
	url := fmt.Sprintf("%s/%s/article/%s", fxTwitterBase, screenName, id)
	tweet, err := fetchAndParse(url)
	if err != nil {
		// Fallback: try with /i/ path
		url = fmt.Sprintf("%s/i/article/%s", fxTwitterBase, id)
		tweet, err = fetchAndParse(url)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch article %s: %w", id, err)
		}
	}
	return tweet, nil
}

// fetchAndParse makes an HTTP GET request and parses the JSON response.
func fetchAndParse(url string) (*Tweet, error) {
	client := &http.Client{Timeout: httpTimeout}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("parsing JSON response: %w", err)
	}

	if apiResp.Code != 200 {
		return nil, fmt.Errorf("API error (code %d): %s", apiResp.Code, apiResp.Message)
	}

	if apiResp.Tweet == nil {
		return nil, fmt.Errorf("no tweet data in response")
	}

	return apiResp.Tweet, nil
}
