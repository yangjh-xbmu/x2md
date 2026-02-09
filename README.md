# x2md

将 X (Twitter) 推文和文章提取为 Markdown。

基于 [FxTwitter API](https://github.com/FixTweet/FxTwitter)，无需认证，无需 API Key。

## 安装

```bash
go install github.com/yangjh/x2md@latest
```

或从源码编译：

```bash
git clone https://github.com/yangjh-xbmu/x2md.git
cd x2md
go build -o x2md .
```

## 用法

```
x2md <url> [flags]

Flags:
  -o string    输出文件路径（默认 stdout）
  -thread      展开整个线程
  -images      下载图片到本地目录
```

### 提取单条推文

```bash
x2md https://x.com/user/status/123456
```

### 提取推文线程

```bash
x2md -thread https://x.com/user/status/123456
```

### 提取文章

```bash
x2md https://x.com/user/article/123456
```

文章 URL 使用 `/status/` 路径时也能自动识别。

### 保存到文件

```bash
x2md -o output.md https://x.com/user/status/123456
```

### 下载图片到本地

```bash
x2md -images -o output.md https://x.com/user/status/123456
```

图片保存到 `output_images/` 目录，Markdown 中的 URL 自动替换为本地路径。

## 输出格式

输出为带 YAML frontmatter 的 Markdown。元数据以结构化方式存储在 frontmatter 中，正文只保留内容。

### 推文

```markdown
---
type: tweet
author: "@user"
author_name: Display Name
date: "2024-01-15T12:30:00Z"
source: "https://x.com/user/status/123456"
likes: 100
retweets: 50
replies: 20
views: 5000
bookmarks: 30
lang: en
via: Twitter Web App
---

推文正文内容...

![image](https://pbs.twimg.com/media/xxx.jpg)
```

### 文章

```markdown
---
type: article
title: 文章标题
author: "@user"
author_name: Display Name
date: "2024-01-15T12:00:00Z"
source: "https://x.com/user/status/123456"
cover_image: "https://pbs.twimg.com/media/xxx.jpg"
likes: 100
retweets: 50
replies: 20
views: 5000
bookmarks: 30
---

# 文章标题

文章正文...
```

## 支持的内容类型

| 类型 | 说明 |
|------|------|
| 单条推文 | 文本、图片、视频链接 |
| 推文线程 | 通过 `-thread` 按时间正序展开同一作者的回复链 |
| 引用推文 | 渲染为 blockquote |
| 投票 | 渲染为列表 + 百分比进度条 |
| 文章 | X Articles 长文章，含标题、封面图、正文 |

## 支持的 URL 格式

支持以下域名，自动识别推文和文章：

- `x.com`
- `twitter.com`
- `fxtwitter.com`
- `fixupx.com`

## 限制

- 仅能获取公开内容，私密账号返回 404
- 线程追溯方向为向上（通过 `replying_to_status`），无法获取目标推文之后的回复
- 线程最多追溯 50 条
- 零外部依赖，仅使用 Go 标准库

## 许可证

MIT
