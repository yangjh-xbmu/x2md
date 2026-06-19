---
purpose: 将 X (Twitter) 推文和文章提取为 Markdown 格式
status: active
next_steps: []
capabilities:
  - x-to-markdown
  - thread-extraction
  - image-download
---
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

## Skill wrapper

仓库同时包含 Claude skill wrapper:

- `SKILL.md`: skill 说明和调用约定
- `x2md-cli`: skill 入口薄壳,负责选择 Python 并调用同目录的 `x2md.py`
- `x2md.py`: 调用 Go 二进制 `x2md` / `x2md.exe`,把结果落盘到 `MyNotes/00_Inbox` 并自动 git commit
- `dida_x2md_inbox.py`: 从滴答清单收集箱读取 X 链接,调用 `x2md-cli` 处理,并记录状态防重复
- `dida_x2md_notify.py`: 包装 `dida_x2md_inbox.py` 的 JSONL 事件流,通过 `hermes send` 发送飞书进度通知

也就是说,Go 二进制负责内容提取,skill wrapper 负责笔记落盘和提交。安装 skill 时复制这三个文件到 `~/.claude/skills/x2md/`:

```bash
mkdir -p ~/.claude/skills/x2md
cp SKILL.md x2md-cli x2md.py dida_x2md_inbox.py dida_x2md_notify.py ~/.claude/skills/x2md/
```

## Dida inbox automation

每天自动处理滴答清单收集箱里的 X 链接时,使用 `dida_x2md_inbox.py`。默认是 dry-run,只列出将处理的链接,不会调用 `x2md-cli`:

```bash
python ~/Desktop/repos/x2md/dida_x2md_inbox.py --dry-run
```

确认无误后用 `--run` 执行。脚本会:

1. 通过 `dida agent context --outline --json` 获取 inbox project id
2. 通过 `dida project tasks "$INBOX_ID" --limit 500 --compact --json` 读取收集箱,避免 `task list` 默认 limit 截断
3. 抽取并规范化 X 链接,按去 query 后的 canonical URL 去重
4. 跳过 `~/.local/state/x2md-dida/processed.json` 中已成功处理的链接
5. 对新链接调用 `~/.claude/skills/x2md/x2md-cli --no-commit`
6. 成功后统一提交 MyNotes 的 `00_Inbox`
7. 输出 JSON 和 Markdown 运行报告到 `~/.local/state/x2md-dida/runs/`

```bash
python ~/Desktop/repos/x2md/dida_x2md_inbox.py --run
```

如果需要让 Hermes/飞书实时看到进度,使用通知包装脚本。默认通知策略是:开始通知一次、失败逐条通知、成功每 10 条汇总一次、结束通知一次。

```bash
python /Users/yangjh/Desktop/repos/x2md/dida_x2md_notify.py --run --target feishu
```

测试飞书通知但不实际处理:

```bash
python /Users/yangjh/Desktop/repos/x2md/dida_x2md_notify.py --dry-run --target feishu
```

需要每条成功都发飞书时:

```bash
python /Users/yangjh/Desktop/repos/x2md/dida_x2md_notify.py --run --target feishu --success-policy each
```

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
