#!/usr/bin/env python3
"""Run dida_x2md_inbox.py and forward progress events through Hermes send."""

from __future__ import annotations

import argparse
import json
import shutil
import subprocess
import sys
from pathlib import Path


DEFAULT_HERMES = Path.home() / ".hermes" / "hermes-agent" / "venv" / "bin" / "hermes"


def find_hermes(explicit: str | None) -> str:
    if explicit:
        return explicit
    if DEFAULT_HERMES.exists():
        return str(DEFAULT_HERMES)
    found = shutil.which("hermes")
    if found:
        return found
    raise FileNotFoundError("Hermes CLI not found. Pass --hermes-bin.")


def send_message(hermes_bin: str, target: str, message: str, dry_notify: bool) -> bool:
    if target == "none":
        print(f"[notify disabled]\n{message}", file=sys.stderr)
        return True
    if dry_notify:
        print(f"[notify dry-run to {target}]\n{message}", file=sys.stderr)
        return True
    result = subprocess.run(
        [hermes_bin, "send", "--to", target, "--quiet", message],
        capture_output=True,
        text=True,
        encoding="utf-8",
        errors="replace",
    )
    if result.returncode != 0:
        err = (result.stderr or result.stdout or "").strip()
        print(f"WARNING: failed to send Hermes notification: {err}", file=sys.stderr)
        return False
    return True


def shorten(text: str, limit: int = 120) -> str:
    text = " ".join(text.split())
    if len(text) <= limit:
        return text
    return text[: limit - 1] + "…"


def format_event_message(event: dict, success_policy: str) -> str | None:
    event_type = event.get("type")
    if event_type == "start":
        return (
            "x2md 滴答收集箱开始处理\n"
            f"收集箱任务：{event.get('inboxTaskCount')}\n"
            f"X 链接：{event.get('linkCount')}\n"
            f"本次新链接：{event.get('newCount')}\n"
            f"已处理跳过：{event.get('skippedProcessedCount')}"
        )
    if event_type == "success" and success_policy == "each":
        saved = event.get("savedPaths") or []
        saved_text = saved[0] if saved else "(未解析到保存路径)"
        return (
            f"x2md 保存成功 {event.get('index')}/{event.get('total')}\n"
            f"{event.get('canonicalUrl')}\n"
            f"{saved_text}"
        )
    if event_type == "failure":
        return (
            f"x2md 保存失败 {event.get('index')}/{event.get('total')}\n"
            f"{event.get('canonicalUrl')}\n"
            f"{shorten(str(event.get('message') or 'unknown error'))}"
        )
    if event_type == "progress":
        return (
            f"x2md 进度 {event.get('done')}/{event.get('total')}\n"
            f"成功：{event.get('successCount')}，失败：{event.get('failureCount')}"
        )
    if event_type == "summary":
        commit = event.get("commit") or {}
        return (
            "x2md 滴答收集箱处理完成\n"
            f"X 链接：{event.get('linkCount')}\n"
            f"本次新链接：{event.get('newCount')}\n"
            f"成功：{event.get('successCount')}，失败：{event.get('failureCount')}\n"
            f"已处理跳过：{event.get('skippedProcessedCount')}\n"
            f"提交：{'成功' if commit.get('ok') else '失败'}\n"
            f"报告：{event.get('markdownReport')}"
        )
    return None


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Run Dida x2md automation with Hermes notifications.")
    mode = parser.add_mutually_exclusive_group()
    mode.add_argument("--dry-run", action="store_true", help="Preview links without processing. Default.")
    mode.add_argument("--run", action="store_true", help="Process links and update state.")
    parser.add_argument("--target", default="feishu", help="Hermes send target, e.g. feishu or feishu:<chat_id>. Use none to disable.")
    parser.add_argument("--hermes-bin", help="Path to Hermes CLI.")
    parser.add_argument("--script", default=Path(__file__).with_name("dida_x2md_inbox.py"), help="Path to dida_x2md_inbox.py.")
    parser.add_argument("--success-policy", choices=("batch", "each", "none"), default="batch", help="Success notification policy.")
    parser.add_argument("--progress-every", type=int, default=10, help="Batch progress interval.")
    parser.add_argument("--dry-notify", action="store_true", help="Print notifications instead of sending.")
    parser.add_argument("script_args", nargs=argparse.REMAINDER, help="Extra args passed after -- to dida_x2md_inbox.py.")
    return parser


def main(argv: list[str] | None = None) -> int:
    args = build_parser().parse_args(argv)
    mode_arg = "--run" if args.run else "--dry-run"
    try:
        hermes_bin = find_hermes(args.hermes_bin) if args.target != "none" and not args.dry_notify else ""
    except FileNotFoundError as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        return 2

    script = Path(args.script).expanduser()
    if not script.exists():
        print(f"ERROR: dida_x2md_inbox.py not found: {script}", file=sys.stderr)
        return 2

    extra = list(args.script_args)
    if extra and extra[0] == "--":
        extra = extra[1:]
    cmd = [
        sys.executable,
        str(script),
        mode_arg,
        "--events",
        "jsonl",
        "--progress-every",
        str(args.progress_every),
        *extra,
    ]
    proc = subprocess.Popen(
        cmd,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        encoding="utf-8",
        errors="replace",
        bufsize=1,
    )

    assert proc.stdout is not None
    for line in proc.stdout:
        line = line.strip()
        if not line:
            continue
        try:
            event = json.loads(line)
        except json.JSONDecodeError:
            print(line)
            continue
        message = format_event_message(event, args.success_policy)
        if message:
            send_message(hermes_bin, args.target, message, args.dry_notify)
        print(json.dumps(event, ensure_ascii=False), flush=True)

    stderr = proc.stderr.read() if proc.stderr else ""
    rc = proc.wait()
    if stderr.strip():
        print(stderr.strip(), file=sys.stderr)
    return rc


if __name__ == "__main__":
    sys.exit(main())
