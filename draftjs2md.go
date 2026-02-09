package main

import (
	"fmt"
	"sort"
	"strings"
)

// DraftJSToMarkdown converts Draft.js article content to Markdown.
func DraftJSToMarkdown(content *ArticleContent, mediaEntities []ArticleMedia) string {
	if content == nil || len(content.Blocks) == 0 {
		return ""
	}

	// Build media lookup: mediaId -> image URL
	mediaLookup := buildMediaLookup(mediaEntities)

	// Build entity map lookup: key -> EntityValue
	entityLookup := make(map[int]EntityValue)
	for _, item := range content.EntityMap {
		entityLookup[int(item.Key)] = item.Value
	}

	var parts []string
	olCounter := 0 // track ordered list numbering

	for _, block := range content.Blocks {
		switch block.Type {
		case "header-one":
			olCounter = 0
			text := applyInlineStyles(block.Text, block.InlineStyleRanges)
			parts = append(parts, "# "+text)

		case "header-two":
			olCounter = 0
			text := applyInlineStyles(block.Text, block.InlineStyleRanges)
			parts = append(parts, "## "+text)

		case "header-three":
			olCounter = 0
			text := applyInlineStyles(block.Text, block.InlineStyleRanges)
			parts = append(parts, "### "+text)

		case "header-four":
			olCounter = 0
			text := applyInlineStyles(block.Text, block.InlineStyleRanges)
			parts = append(parts, "#### "+text)

		case "header-five":
			olCounter = 0
			text := applyInlineStyles(block.Text, block.InlineStyleRanges)
			parts = append(parts, "##### "+text)

		case "header-six":
			olCounter = 0
			text := applyInlineStyles(block.Text, block.InlineStyleRanges)
			parts = append(parts, "###### "+text)

		case "blockquote":
			olCounter = 0
			text := applyInlineStyles(block.Text, block.InlineStyleRanges)
			lines := strings.Split(text, "\n")
			var quoted []string
			for _, line := range lines {
				quoted = append(quoted, "> "+line)
			}
			parts = append(parts, strings.Join(quoted, "\n"))

		case "unordered-list-item":
			olCounter = 0
			text := applyInlineStyles(block.Text, block.InlineStyleRanges)
			parts = append(parts, "- "+text)

		case "ordered-list-item":
			olCounter++
			text := applyInlineStyles(block.Text, block.InlineStyleRanges)
			parts = append(parts, fmt.Sprintf("%d. %s", olCounter, text))

		case "code-block":
			olCounter = 0
			parts = append(parts, "```\n"+block.Text+"\n```")

		case "atomic":
			olCounter = 0
			// Atomic blocks contain media or dividers referenced by entityRanges
			rendered := renderAtomicBlock(block, entityLookup, mediaLookup)
			if rendered != "" {
				parts = append(parts, rendered)
			}

		default: // "unstyled" and others
			olCounter = 0
			if strings.TrimSpace(block.Text) == "" {
				parts = append(parts, "")
			} else {
				text := applyInlineStyles(block.Text, block.InlineStyleRanges)
				parts = append(parts, text)
			}
		}
	}

	return strings.TrimSpace(strings.Join(parts, "\n\n"))
}

// buildMediaLookup creates a map from mediaId to image URL.
func buildMediaLookup(entities []ArticleMedia) map[string]string {
	lookup := make(map[string]string)
	for _, e := range entities {
		if e.MediaInfo != nil && e.MediaInfo.OriginalImgURL != "" {
			lookup[e.MediaID] = e.MediaInfo.OriginalImgURL
		}
	}
	return lookup
}

// renderAtomicBlock renders an atomic block (media, divider).
func renderAtomicBlock(block Block, entityLookup map[int]EntityValue, mediaLookup map[string]string) string {
	for _, er := range block.EntityRanges {
		entity, ok := entityLookup[er.Key]
		if !ok {
			continue
		}

		switch entity.Type {
		case "MEDIA":
			return renderMediaEntity(entity, mediaLookup)
		case "DIVIDER":
			return "---"
		}
	}
	return ""
}

// renderMediaEntity renders a MEDIA entity as Markdown image(s).
func renderMediaEntity(entity EntityValue, mediaLookup map[string]string) string {
	var images []string
	for _, ref := range entity.Data.MediaItems {
		if url, ok := mediaLookup[ref.MediaID]; ok {
			images = append(images, fmt.Sprintf("![image](%s)", url))
		}
	}
	return strings.Join(images, "\n\n")
}

// styleRange represents a style boundary event for inline style processing.
type styleRange struct {
	pos   int
	style string
	start bool
}

// applyInlineStyles applies Bold, Italic, Code, Underline styles to text.
func applyInlineStyles(text string, styles []InlineStyleRange) string {
	if len(styles) == 0 {
		return text
	}

	runes := []rune(text)
	n := len(runes)

	// Collect all style boundaries
	var events []styleRange
	for _, s := range styles {
		end := s.Offset + s.Length
		if end > n {
			end = n
		}
		events = append(events,
			styleRange{pos: s.Offset, style: s.Style, start: true},
			styleRange{pos: end, style: s.Style, start: false},
		)
	}

	// Sort events: process ends before starts at the same position
	sort.Slice(events, func(i, j int) bool {
		if events[i].pos != events[j].pos {
			return events[i].pos < events[j].pos
		}
		// Ends before starts at same position
		if events[i].start != events[j].start {
			return !events[i].start
		}
		return false
	})

	// Build result by inserting markers at boundaries
	var result strings.Builder
	eventIdx := 0

	for pos := 0; pos <= n; pos++ {
		// Process all events at this position (ends first, then starts)
		for eventIdx < len(events) && events[eventIdx].pos == pos {
			marker := styleMarker(events[eventIdx].style)
			result.WriteString(marker)
			eventIdx++
		}
		if pos < n {
			result.WriteRune(runes[pos])
		}
	}

	return result.String()
}

// styleMarker returns the Markdown marker for a style.
func styleMarker(style string) string {
	switch style {
	case "Bold", "BOLD":
		return "**"
	case "Italic", "ITALIC":
		return "*"
	case "Code", "CODE":
		return "`"
	default:
		return ""
	}
}
