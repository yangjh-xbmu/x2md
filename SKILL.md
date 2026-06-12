---
name: x2md
description: 把 X (Twitter) 推文或长文转成 Markdown,落盘到 MyNotes/00_Inbox。底层调用本机 `x2md` Go 二进制(从 github.com/yangjh-xbmu/x2md 编译安装,使用 fxtwitter API 无需鉴权)。输出文件名 `x-{author}-{YYYYMMDD}.md`,完事自动 git commit。触发词:「存 X」「存档 X 链接」「x2md 一下这条推」「把这条推文保存到笔记」。也接受 `/x2md <url>` 直接调用。
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

底层调用 `./x2md-cli <url>`。`x2md-cli` 是 skill 入口薄壳,自动选能跑的 Python 调同目录的 `x2md.py`。`x2md.py` 再调用 PATH 或 `~/bin/` 里的 Go 二进制 `x2md` / `x2md.exe`,负责落盘和 commit。

## 流水线

1. 校验 URL 必须命中 `x.com` / `twitter.com` / `fxtwitter.com` / `fixupx.com`
2. 调 Go 二进制 `x2md -o <tmpfile> [flags] <url>`
3. 解析输出 frontmatter 拿 `author` / `date` 字段
4. 文件名:`x-{author-sanitized}-{YYYYMMDD}.md`
   - author: 去 `@`,`.` / `_` 转 `-`,去掉非 `\w-` 字符
   - 已存在则追加 `-2` / `-3` ...
5. 落盘到 `00_Inbox/`
6. `git add 00_Inbox/ && git commit -m "Add: X存档 - {author}"`

## Script / Binary Location

- **事实源仓库**:`~/Desktop/repos/x2md/`
  - Go 源码 — 编译出 `x2md` / `x2md.exe`,负责提取 X 内容
  - `SKILL.md` / `x2md-cli` / `x2md.py` — skill wrapper,负责落盘和 commit
  - `dida_x2md_inbox.py` — 滴答收集箱 X 链接自动处理脚本
  - `dida_x2md_notify.py` — 调用 Hermes send 发送飞书进度通知的包装脚本
- **skill 安装目录**:`~/.claude/skills/x2md/`
  - `SKILL.md` — Claude 读这个
  - `x2md-cli` — 跨平台薄壳,自动选 `python3` / `python` 调下面的 `x2md.py`
  - `x2md.py` — Python 包装,做落盘 + git commit
  - `dida_x2md_inbox.py` — Hermes 定时任务可调用的确定性脚本
  - `dida_x2md_notify.py` — Hermes/飞书通知包装脚本
- **二进制**:`~/bin/x2md.exe`(Windows) / `~/bin/x2md`(macOS/Linux),在 PATH

## 跨机器安装

```bash
# 1. clone 源码(任何机器都一样)
git clone https://github.com/yangjh-xbmu/x2md.git ~/Desktop/repos/x2md

# 2. 编译并装到 PATH
# macOS / Linux:
cd ~/Desktop/repos/x2md && go build -o ~/bin/x2md .
# Windows (Git Bash):
cd ~/Desktop/repos/x2md && GOOS=windows GOARCH=amd64 go build -o ~/bin/x2md.exe .

# 3. 安装 skill wrapper
mkdir -p ~/.claude/skills/x2md
cp SKILL.md x2md-cli x2md.py dida_x2md_inbox.py dida_x2md_notify.py ~/.claude/skills/x2md/
```

> 想要纯 markdown 流(不落盘)直接 `x2md <url>`,原始 markdown 输出到 stdout。

## 滴答收集箱自动处理

首版自动化脚本只做采集和去重,不自动完成滴答任务。

Dry-run:

```bash
python ~/Desktop/repos/x2md/dida_x2md_inbox.py --dry-run
```

实际运行:

```bash
python ~/Desktop/repos/x2md/dida_x2md_inbox.py --run
```

Hermes 定时任务建议调用 Mac 上的绝对路径。默认通知策略是开始一次、失败逐条、成功每 10 条进度、结束一次:

```bash
python /Users/yangjh/Desktop/repos/x2md/dida_x2md_notify.py --run --target feishu
```

测试飞书通知但不实际处理:

```bash
python /Users/yangjh/Desktop/repos/x2md/dida_x2md_notify.py --dry-run --target feishu
```

运行状态写入 `~/.local/state/x2md-dida/processed.json`,报告写入 `~/.local/state/x2md-dida/runs/`。

## 常见错误

- `error: 请提供 X (Twitter) URL` — 没传 URL
- `HTTP 404` — 推文不存在或账号私密
- `git commit 跳过` — 00_Inbox 不在 git repo 里(刚 clone MyNotes 还没初始化)
- `No working Python found` — 极少见,系统既没 `python3` 也没能跑的 `python`

## 不做的事

- 不下载推文里的视频(只下载图片,且需 `--images`)
- 不爬取评论区(只追作者自己发的线程,最多 50 条)
- 不改原始 x2md 行为,只在外面套一层落盘+提交
