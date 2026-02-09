package main

import (
	"fmt"
	"strings"
	"time"
)

// yamlEscape escapes a string for use as a YAML value.
// Wraps in quotes if the string contains special characters.
func yamlEscape(s string) string {
	if s == "" {
		return `""`
	}
	if strings.ContainsAny(s, ":#{}[]|>&*!,?\\\"'\n") {
		escaped := strings.ReplaceAll(s, `\`, `\\`)
		escaped = strings.ReplaceAll(escaped, `"`, `\"`)
		return `"` + escaped + `"`
	}
	return s
}

// writeFrontmatter writes YAML frontmatter from key-value pairs.
// Only writes non-empty string values and non-zero int values.
func writeFrontmatter(sb *strings.Builder, fields []frontmatterField) {
	sb.WriteString("---\n")
	for _, f := range fields {
		switch v := f.value.(type) {
		case string:
			if v != "" {
				sb.WriteString(fmt.Sprintf("%s: %s\n", f.key, yamlEscape(v)))
			}
		case int:
			sb.WriteString(fmt.Sprintf("%s: %d\n", f.key, v))
		case int64:
			sb.WriteString(fmt.Sprintf("%s: %d\n", f.key, v))
		}
	}
	sb.WriteString("---\n\n")
}

type frontmatterField struct {
	key   string
	value interface{}
}

// RenderTweet renders a single tweet as Markdown with frontmatter.
func RenderTweet(tweet *Tweet) string {
	var sb strings.Builder

	writeTweetFrontmatter(&sb, tweet)
	writeText(&sb, tweet.Text)
	writeMedia(&sb, tweet.Media)
	writePoll(&sb, tweet.Poll)
	writeQuote(&sb, tweet.Quote)

	return sb.String()
}

// RenderThread renders a thread (multiple tweets) as Markdown with frontmatter.
func RenderThread(tweets []*Tweet) string {
	if len(tweets) == 0 {
		return ""
	}

	var sb strings.Builder

	// Use the first tweet for author/date, last tweet for stats and source URL
	first := tweets[0]
	last := tweets[len(tweets)-1]

	fields := []frontmatterField{
		{"type", "thread"},
		{"tweet_count", len(tweets)},
	}
	if first.Author != nil {
		fields = append(fields,
			frontmatterField{"author", "@" + first.Author.ScreenName},
			frontmatterField{"author_name", first.Author.Name},
		)
	}
	fields = append(fields, frontmatterField{"date", formatDate(first.CreatedAt)})
	if last.Author != nil {
		fields = append(fields, frontmatterField{"source", fmt.Sprintf("https://x.com/%s/status/%s", last.Author.ScreenName, last.ID)})
	}
	fields = append(fields,
		frontmatterField{"likes", last.Likes},
		frontmatterField{"retweets", last.Retweets},
		frontmatterField{"replies", last.Replies},
		frontmatterField{"views", last.Views},
	)
	writeFrontmatter(&sb, fields)

	for i, tweet := range tweets {
		if i > 0 {
			sb.WriteString("\n---\n\n")
		}
		writeText(&sb, tweet.Text)
		writeMedia(&sb, tweet.Media)
		writePoll(&sb, tweet.Poll)
		writeQuote(&sb, tweet.Quote)
	}

	return sb.String()
}

// RenderArticle renders an X Article as Markdown with frontmatter.
func RenderArticle(tweet *Tweet, info URLInfo) string {
	var sb strings.Builder

	article := tweet.Article
	if article == nil {
		return RenderTweet(tweet)
	}

	// Frontmatter
	fields := []frontmatterField{
		{"type", "article"},
		{"title", article.Title},
	}
	if tweet.Author != nil {
		fields = append(fields,
			frontmatterField{"author", "@" + tweet.Author.ScreenName},
			frontmatterField{"author_name", tweet.Author.Name},
		)
	}
	dateStr := formatDate(tweet.CreatedAt)
	if article.CreatedAt != "" {
		dateStr = formatDate(article.CreatedAt)
	}
	fields = append(fields, frontmatterField{"date", dateStr})
	if article.ModifiedAt != "" {
		fields = append(fields, frontmatterField{"modified", formatDate(article.ModifiedAt)})
	}
	fields = append(fields, frontmatterField{"source", info.OriginalURL})
	if article.CoverMedia != nil && article.CoverMedia.MediaInfo != nil {
		fields = append(fields, frontmatterField{"cover_image", article.CoverMedia.MediaInfo.OriginalImgURL})
	}
	fields = append(fields,
		frontmatterField{"likes", tweet.Likes},
		frontmatterField{"retweets", tweet.Retweets},
		frontmatterField{"replies", tweet.Replies},
		frontmatterField{"views", tweet.Views},
		frontmatterField{"bookmarks", tweet.Bookmarks},
	)
	writeFrontmatter(&sb, fields)

	// Title as H1
	if article.Title != "" {
		sb.WriteString("# " + article.Title + "\n\n")
	}

	// Cover image
	if article.CoverMedia != nil && article.CoverMedia.MediaInfo != nil &&
		article.CoverMedia.MediaInfo.OriginalImgURL != "" {
		sb.WriteString(fmt.Sprintf("![cover](%s)\n\n", article.CoverMedia.MediaInfo.OriginalImgURL))
	}

	// Article content from Draft.js blocks
	if article.Content != nil {
		md := DraftJSToMarkdown(article.Content, article.MediaEntities)
		if md != "" {
			sb.WriteString(md)
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func writeTweetFrontmatter(sb *strings.Builder, tweet *Tweet) {
	fields := []frontmatterField{
		{"type", "tweet"},
	}
	if tweet.Author != nil {
		fields = append(fields,
			frontmatterField{"author", "@" + tweet.Author.ScreenName},
			frontmatterField{"author_name", tweet.Author.Name},
		)
	}
	fields = append(fields, frontmatterField{"date", formatDate(tweet.CreatedAt)})
	if tweet.Author != nil {
		fields = append(fields, frontmatterField{"source", fmt.Sprintf("https://x.com/%s/status/%s", tweet.Author.ScreenName, tweet.ID)})
	}
	fields = append(fields,
		frontmatterField{"likes", tweet.Likes},
		frontmatterField{"retweets", tweet.Retweets},
		frontmatterField{"replies", tweet.Replies},
		frontmatterField{"views", tweet.Views},
		frontmatterField{"bookmarks", tweet.Bookmarks},
	)
	if tweet.Lang != "" {
		fields = append(fields, frontmatterField{"lang", tweet.Lang})
	}
	if tweet.Source != "" {
		fields = append(fields, frontmatterField{"via", tweet.Source})
	}
	writeFrontmatter(sb, fields)
}

func writeText(sb *strings.Builder, text string) {
	if text == "" {
		return
	}
	sb.WriteString(text + "\n")
}

func writeMedia(sb *strings.Builder, media *Media) {
	if media == nil {
		return
	}

	for _, photo := range media.Photos {
		alt := photo.AltText
		if alt == "" {
			alt = "image"
		}
		sb.WriteString(fmt.Sprintf("\n![%s](%s)\n", alt, photo.URL))
	}

	for _, video := range media.Videos {
		if video.URL != "" {
			sb.WriteString(fmt.Sprintf("\n[▶ Video](%s)\n", video.URL))
		} else if video.ThumbnailURL != "" {
			sb.WriteString(fmt.Sprintf("\n![video thumbnail](%s)\n", video.ThumbnailURL))
		}
	}
}

func writePoll(sb *strings.Builder, poll *Poll) {
	if poll == nil {
		return
	}

	sb.WriteString("\n**投票**")
	if poll.Ended {
		sb.WriteString(" (已结束)")
	}
	sb.WriteString("\n\n")

	for _, choice := range poll.Choices {
		bar := renderPollBar(choice.Percentage)
		sb.WriteString(fmt.Sprintf("- %s %s (%.1f%%)\n", choice.Label, bar, choice.Percentage))
	}
	sb.WriteString(fmt.Sprintf("\n共 %d 票\n", poll.TotalVotes))
}

func renderPollBar(percentage float64) string {
	filled := int(percentage / 5)
	if filled > 20 {
		filled = 20
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", 20-filled)
}

func writeQuote(sb *strings.Builder, quote *Tweet) {
	if quote == nil {
		return
	}

	sb.WriteString("\n")
	lines := strings.Split(quote.Text, "\n")
	for _, line := range lines {
		sb.WriteString("> " + line + "\n")
	}

	if quote.Author != nil {
		sb.WriteString(fmt.Sprintf("> — @%s\n", quote.Author.ScreenName))
	}
}

// formatDate formats a date string to a more readable format.
func formatDate(dateStr string) string {
	if dateStr == "" {
		return ""
	}

	// Try parsing Twitter's date format: "Wed Jan 15 12:30:00 +0000 2024"
	t, err := time.Parse(time.RubyDate, dateStr)
	if err != nil {
		// Try RFC1123 format
		t, err = time.Parse(time.RFC1123, dateStr)
		if err != nil {
			// Try ISO 8601
			t, err = time.Parse(time.RFC3339, dateStr)
			if err != nil {
				return dateStr
			}
		}
	}
	return t.UTC().Format("2006-01-02T15:04:05Z")
}
