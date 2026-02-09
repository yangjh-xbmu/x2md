package main

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

// HTMLToMarkdown converts simple HTML (as returned by FxTwitter for articles)
// to Markdown. This is a lightweight implementation handling common tags.
func HTMLToMarkdown(htmlStr string) string {
	s := htmlStr

	// Normalize line endings
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")

	// Process pre/code blocks first to protect their content
	s = processPreBlocks(s)

	// Handle block-level elements
	s = processHeadings(s)
	s = processBlockquotes(s)
	s = processLists(s)
	s = processParagraphs(s)
	s = processHorizontalRules(s)

	// Handle inline elements
	s = processLinks(s)
	s = processImages(s)
	s = processBold(s)
	s = processItalic(s)
	s = processInlineCode(s)
	s = processLineBreaks(s)

	// Strip remaining HTML tags
	s = stripTags(s)

	// Decode HTML entities
	s = html.UnescapeString(s)

	// Clean up excessive whitespace
	s = cleanWhitespace(s)

	return strings.TrimSpace(s)
}

var preBlockRe = regexp.MustCompile(`(?is)<pre[^>]*>(.*?)</pre>`)
var codeInPreRe = regexp.MustCompile(`(?is)<code(?:\s+class="language-([^"]*)")?[^>]*>(.*?)</code>`)

func processPreBlocks(s string) string {
	return preBlockRe.ReplaceAllStringFunc(s, func(match string) string {
		inner := preBlockRe.FindStringSubmatch(match)
		if inner == nil {
			return match
		}
		content := inner[1]

		// Check if there's a <code> tag with optional language class
		lang := ""
		if cm := codeInPreRe.FindStringSubmatch(content); cm != nil {
			lang = cm[1]
			content = cm[2]
		}

		// Decode entities inside code blocks
		content = html.UnescapeString(content)
		content = stripTags(content)
		content = strings.TrimSpace(content)

		return "\n\n```" + lang + "\n" + content + "\n```\n\n"
	})
}

var (
	h1Re = regexp.MustCompile(`(?is)<h1[^>]*>(.*?)</h1>`)
	h2Re = regexp.MustCompile(`(?is)<h2[^>]*>(.*?)</h2>`)
	h3Re = regexp.MustCompile(`(?is)<h3[^>]*>(.*?)</h3>`)
	h4Re = regexp.MustCompile(`(?is)<h4[^>]*>(.*?)</h4>`)
	h5Re = regexp.MustCompile(`(?is)<h5[^>]*>(.*?)</h5>`)
	h6Re = regexp.MustCompile(`(?is)<h6[^>]*>(.*?)</h6>`)
)

func processHeadings(s string) string {
	s = h1Re.ReplaceAllStringFunc(s, func(m string) string {
		inner := stripTags(h1Re.FindStringSubmatch(m)[1])
		return "\n\n# " + strings.TrimSpace(inner) + "\n\n"
	})
	s = h2Re.ReplaceAllStringFunc(s, func(m string) string {
		inner := stripTags(h2Re.FindStringSubmatch(m)[1])
		return "\n\n## " + strings.TrimSpace(inner) + "\n\n"
	})
	s = h3Re.ReplaceAllStringFunc(s, func(m string) string {
		inner := stripTags(h3Re.FindStringSubmatch(m)[1])
		return "\n\n### " + strings.TrimSpace(inner) + "\n\n"
	})
	s = h4Re.ReplaceAllStringFunc(s, func(m string) string {
		inner := stripTags(h4Re.FindStringSubmatch(m)[1])
		return "\n\n#### " + strings.TrimSpace(inner) + "\n\n"
	})
	s = h5Re.ReplaceAllStringFunc(s, func(m string) string {
		inner := stripTags(h5Re.FindStringSubmatch(m)[1])
		return "\n\n##### " + strings.TrimSpace(inner) + "\n\n"
	})
	s = h6Re.ReplaceAllStringFunc(s, func(m string) string {
		inner := stripTags(h6Re.FindStringSubmatch(m)[1])
		return "\n\n###### " + strings.TrimSpace(inner) + "\n\n"
	})
	return s
}

var blockquoteRe = regexp.MustCompile(`(?is)<blockquote[^>]*>(.*?)</blockquote>`)

func processBlockquotes(s string) string {
	return blockquoteRe.ReplaceAllStringFunc(s, func(match string) string {
		inner := blockquoteRe.FindStringSubmatch(match)[1]
		inner = stripTags(inner)
		inner = strings.TrimSpace(inner)
		lines := strings.Split(inner, "\n")
		var quoted []string
		for _, line := range lines {
			quoted = append(quoted, "> "+strings.TrimSpace(line))
		}
		return "\n\n" + strings.Join(quoted, "\n") + "\n\n"
	})
}

var (
	ulRe = regexp.MustCompile(`(?is)<ul[^>]*>(.*?)</ul>`)
	olRe = regexp.MustCompile(`(?is)<ol[^>]*>(.*?)</ol>`)
	liRe = regexp.MustCompile(`(?is)<li[^>]*>(.*?)</li>`)
)

func processLists(s string) string {
	// Unordered lists
	s = ulRe.ReplaceAllStringFunc(s, func(match string) string {
		inner := ulRe.FindStringSubmatch(match)[1]
		items := liRe.FindAllStringSubmatch(inner, -1)
		var lines []string
		for _, item := range items {
			text := strings.TrimSpace(stripTags(item[1]))
			lines = append(lines, "- "+text)
		}
		return "\n\n" + strings.Join(lines, "\n") + "\n\n"
	})

	// Ordered lists
	s = olRe.ReplaceAllStringFunc(s, func(match string) string {
		inner := olRe.FindStringSubmatch(match)[1]
		items := liRe.FindAllStringSubmatch(inner, -1)
		var lines []string
		for i, item := range items {
			text := strings.TrimSpace(stripTags(item[1]))
			lines = append(lines, fmt.Sprintf("%d. %s", i+1, text))
		}
		return "\n\n" + strings.Join(lines, "\n") + "\n\n"
	})

	return s
}

var pRe = regexp.MustCompile(`(?is)<p[^>]*>(.*?)</p>`)

func processParagraphs(s string) string {
	return pRe.ReplaceAllString(s, "\n\n$1\n\n")
}

var hrRe = regexp.MustCompile(`(?i)<hr\s*/?>`)

func processHorizontalRules(s string) string {
	return hrRe.ReplaceAllString(s, "\n\n---\n\n")
}

var linkRe = regexp.MustCompile(`(?is)<a\s[^>]*href="([^"]*)"[^>]*>(.*?)</a>`)

func processLinks(s string) string {
	return linkRe.ReplaceAllStringFunc(s, func(match string) string {
		parts := linkRe.FindStringSubmatch(match)
		href := parts[1]
		text := strings.TrimSpace(stripTags(parts[2]))
		if text == "" {
			text = href
		}
		return "[" + text + "](" + href + ")"
	})
}

var imgRe = regexp.MustCompile(`(?i)<img\s[^>]*src="([^"]*)"[^>]*(?:alt="([^"]*)")?[^>]*/?>`)

func processImages(s string) string {
	return imgRe.ReplaceAllStringFunc(s, func(match string) string {
		parts := imgRe.FindStringSubmatch(match)
		src := parts[1]
		alt := ""
		if len(parts) > 2 {
			alt = parts[2]
		}
		return "![" + alt + "](" + src + ")"
	})
}

var (
	boldRe  = regexp.MustCompile(`(?is)<(?:strong|b)>(.*?)</(?:strong|b)>`)
)

func processBold(s string) string {
	return boldRe.ReplaceAllString(s, "**$1**")
}

var (
	italicRe = regexp.MustCompile(`(?is)<(?:em|i)>(.*?)</(?:em|i)>`)
)

func processItalic(s string) string {
	return italicRe.ReplaceAllString(s, "*$1*")
}

var inlineCodeRe = regexp.MustCompile(`(?is)<code>(.*?)</code>`)

func processInlineCode(s string) string {
	return inlineCodeRe.ReplaceAllString(s, "`$1`")
}

var brRe = regexp.MustCompile(`(?i)<br\s*/?>`)

func processLineBreaks(s string) string {
	return brRe.ReplaceAllString(s, "\n")
}

var tagRe = regexp.MustCompile(`<[^>]*>`)

func stripTags(s string) string {
	return tagRe.ReplaceAllString(s, "")
}

var multiNewlineRe = regexp.MustCompile(`\n{3,}`)
var trailingSpaceRe = regexp.MustCompile(`[ \t]+\n`)

func cleanWhitespace(s string) string {
	s = trailingSpaceRe.ReplaceAllString(s, "\n")
	s = multiNewlineRe.ReplaceAllString(s, "\n\n")
	return s
}
