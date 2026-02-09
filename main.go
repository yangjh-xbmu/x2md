package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func main() {
	outputFile := flag.String("o", "", "输出文件路径（默认 stdout）")
	thread := flag.Bool("thread", false, "展开整个线程（默认只提取单条）")
	images := flag.Bool("images", false, "下载图片到本地目录")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "x2md — 将 X (Twitter) 内容提取为 Markdown\n\n")
		fmt.Fprintf(os.Stderr, "用法:\n  x2md <url> [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n示例:\n")
		fmt.Fprintf(os.Stderr, "  x2md https://x.com/elonmusk/status/123456\n")
		fmt.Fprintf(os.Stderr, "  x2md -thread https://x.com/user/status/123456\n")
		fmt.Fprintf(os.Stderr, "  x2md -o output.md https://x.com/user/status/123456\n")
		fmt.Fprintf(os.Stderr, "  x2md https://x.com/user/article/123456\n")
	}

	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "错误: 请提供 X (Twitter) URL")
		flag.Usage()
		os.Exit(1)
	}

	rawURL := flag.Arg(0)

	info, err := ParseURL(rawURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}

	var markdown string

	switch info.Type {
	case URLTypeArticle:
		tweet, err := FetchArticle(info.ScreenName, info.ID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "错误: 获取文章失败: %v\n", err)
			os.Exit(1)
		}
		markdown = RenderArticle(tweet, info)

	case URLTypeTweet:
		if *thread {
			tweets, err := FetchThread(info.ScreenName, info.ID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "错误: 获取线程失败: %v\n", err)
				os.Exit(1)
			}
			markdown = RenderThread(tweets)
		} else {
			tweet, err := FetchTweet(info.ScreenName, info.ID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "错误: 获取推文失败: %v\n", err)
				os.Exit(1)
			}
			// Auto-detect: if tweet contains an article, render as article
			if tweet.Article != nil && tweet.Article.Content != nil {
				markdown = RenderArticle(tweet, info)
			} else {
				markdown = RenderTweet(tweet)
			}
		}
	}

	// Download images if requested
	if *images && markdown != "" {
		imgDir := "images"
		if *outputFile != "" {
			imgDir = strings.TrimSuffix(*outputFile, filepath.Ext(*outputFile)) + "_images"
		}
		markdown = downloadAndReplaceImages(markdown, imgDir)
	}

	// Output
	if *outputFile != "" {
		if err := os.WriteFile(*outputFile, []byte(markdown), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "错误: 写入文件失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "已保存到 %s\n", *outputFile)
	} else {
		fmt.Print(markdown)
	}
}

var mdImageRe = regexp.MustCompile(`!\[([^\]]*)\]\((https?://[^)]+)\)`)

// downloadAndReplaceImages downloads images found in Markdown and replaces URLs with local paths.
func downloadAndReplaceImages(markdown, imgDir string) string {
	matches := mdImageRe.FindAllStringSubmatch(markdown, -1)
	if len(matches) == 0 {
		return markdown
	}

	if err := os.MkdirAll(imgDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "警告: 无法创建图片目录 %s: %v\n", imgDir, err)
		return markdown
	}

	for i, match := range matches {
		fullMatch := match[0]
		alt := match[1]
		imgURL := match[2]

		ext := filepath.Ext(imgURL)
		if ext == "" || len(ext) > 5 {
			ext = ".jpg"
		}
		// Clean extension (remove query params)
		if idx := strings.Index(ext, "?"); idx != -1 {
			ext = ext[:idx]
		}

		filename := fmt.Sprintf("img_%d%s", i+1, ext)
		localPath := filepath.Join(imgDir, filename)

		if err := downloadFile(imgURL, localPath); err != nil {
			fmt.Fprintf(os.Stderr, "警告: 下载图片失败 %s: %v\n", imgURL, err)
			continue
		}

		newRef := fmt.Sprintf("![%s](%s)", alt, localPath)
		markdown = strings.Replace(markdown, fullMatch, newRef, 1)
		fmt.Fprintf(os.Stderr, "已下载: %s\n", localPath)
	}

	return markdown
}

func downloadFile(url, destPath string) error {
	client := &http.Client{Timeout: 30 * time.Second}

	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}
