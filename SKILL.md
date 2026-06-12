---
name: x2md
description: 把 X (Twitter) 推文或长文转成 Markdown,落盘到 MyNotes/00_Inbox。底层调本机 `~/bin/x2md.exe`(用 GOPROXY=goproxy.cn 在 Windows 交叉编译自 github.com/yangjh-xbmu/x2md,使用 fxtwitter API 无需鉴权)。输出文件名 `x-{author}-{YYYYMMDD}.md`,完事自动 git commit。触发词:「存 X」「存档 X 链接」「x2md 一下这条推」「把这条推文保存到笔记」。也接受 `/x2md <url>` 直接调用。
user-invocable: true
context: fork
allowed-tools: Bash, Read, Glob
effort: low
---

# /x2md — X (Twitter) URL → Markdown

把 X 链接(推文、线程、长文)变成可在 Obsidian 读的 Markdown,落 `~/Desktop/repos/MyNotes/00_Inbox/`,然后自动 git commit。

## Usage

```
/x2md <url> [<url2> ...] [-o <output_dir>] [--thread] [--images] [--no-commit]
```

默认输出目录:`~/Desktop/repos/MyNotes/00_Inbox`(沿用 mac 上 x-content-saver 的约定)。

Examples:

```
/x2md https://x.com/elonmusk/status/123456
/x2md https://x.com/user/status/123 -thread
/x2md https://x.com/user/article/123 -o ~/Downloads
```

## 流水线

1. 校验 URL 必须命中 `x.com` / `twitter.com` / `fxtwitter.com` / `fixupx.com`
2. 调 `~/bin/x2md.exe -o <tmpfile> [flags] <url>`
3. 解析输出 frontmatter 拿 `author` / `date` 字段
4. 文件名:`x-{author-sanitized}-{YYYYMMDD}.md`
   - author: 去 `@`,`.` / `_` 转 `-`,去掉非 `\w-` 字符
   - 已存在则追加 `-2` / `-3` ...
5. 落盘到 `00_Inbox/`
6. `git add 00_Inbox/ && git commit -m "Add: X存档 - {author}"`

## 二进制

- 路径:`~/bin/x2md.exe`(已在 PATH)
- 源码:`~/Desktop/repos/x2md/`(GitHub: yangjh-xbmu/x2md)
- Go 1.24+,无外部依赖,纯 stdlib
- 跨机器同步:二进制不随 chezmoi 走,每台机器 clone 源码 + `GOOS=windows GOARCH=amd64 go build` 一次

## 常见错误

- `error: 请提供 X (Twitter) URL` — 没传 URL
- `HTTP 404` — 推文不存在或账号私密
- `git commit 跳过` — 00_Inbox 不在 git repo 里(刚 clone MyNotes 还没初始化)

## 不做的事

- 不下载推文里的视频(只下载图片,且需 `--images`)
- 不爬取评论区(只追作者自己发的线程,最多 50 条)
- 不改原始 x2md 行为,只在外面套一层落盘+提交
