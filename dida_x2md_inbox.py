#!/usr/bin/env python3
"""Process X links from Dida inbox with the x2md skill wrapper."""

from __future__ import annotations

import argparse
import json
import os
import re
import shutil
import subprocess
import sys
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path
from urllib.parse import urlsplit, urlunsplit


DEFAULT_STATE_DIR = Path.home() / ".local" / "state" / "x2md-dida"
DEFAULT_OUTPUT_DIR = Path.home() / "Desktop" / "repos" / "MyNotes" / "00_Inbox"
DEFAULT_X2MD_CLI = Path.home() / ".claude" / "skills" / "x2md" / "x2md-cli"
URL_RE = re.compile(
    r"https?://(?:www\.)?(?:x\.com|twitter\.com|fxtwitter\.com|fixupx\.com)/[^\s<>\]\)\"']+",
    re.IGNORECASE,
)
TEXT_FIELDS = ("title", "content", "desc", "description", "note")


@dataclass
class LinkItem:
    canonical_url: str
    url: str
    task_ids: list[str]
    titles: list[str]


def now_iso() -> str:
    return datetime.now(timezone.utc).replace(microsecond=0).isoformat()


def emit_event(enabled: bool, event: dict) -> None:
    if not enabled:
        return
    event.setdefault("ts", now_iso())
    print(json.dumps(event, ensure_ascii=False), flush=True)


def find_dida_binary(explicit: str | None) -> str:
    if explicit:
        return explicit

    candidates = [
        Path.home() / "Desktop" / "repos" / "dida-cli" / "bin" / "dida.exe",
        Path.home() / "Desktop" / "repos" / "dida-cli" / "bin" / "dida",
    ]
    for candidate in candidates:
        if candidate.exists():
            return str(candidate)

    found = shutil.which("dida") or shutil.which("dida.exe")
    if found:
        return found

    raise FileNotFoundError("dida CLI not found. Pass --dida-bin or install dida in PATH.")


def find_x2md_cli(explicit: str | None) -> Path:
    if explicit:
        path = Path(explicit).expanduser()
    else:
        path = DEFAULT_X2MD_CLI
    if path.exists():
        return path
    raise FileNotFoundError(f"x2md-cli not found: {path}")


def run_json(cmd: list[str]) -> dict:
    result = subprocess.run(cmd, capture_output=True, text=True, encoding="utf-8", errors="replace")
    if result.returncode != 0:
        stderr = (result.stderr or "").strip()
        raise RuntimeError(f"command failed: {' '.join(cmd)}\n{stderr}")
    try:
        return json.loads(result.stdout)
    except json.JSONDecodeError as exc:
        raise RuntimeError(f"command returned invalid JSON: {' '.join(cmd)}") from exc


def resolve_inbox_id(dida_bin: str, explicit: str | None) -> str:
    if explicit:
        return explicit
    data = run_json([dida_bin, "agent", "context", "--outline", "--json"])
    inbox_id = (((data or {}).get("data") or {}).get("inboxId") or "").strip()
    if not inbox_id:
        raise RuntimeError("failed to resolve inboxId from dida agent context")
    return inbox_id


def read_inbox_tasks(dida_bin: str, inbox_id: str, limit: int) -> list[dict]:
    data = run_json([dida_bin, "project", "tasks", inbox_id, "--limit", str(limit), "--compact", "--json"])
    tasks = ((data or {}).get("data") or {}).get("tasks") or []
    if len(tasks) >= limit:
        raise RuntimeError(f"inbox task count reached limit {limit}; rerun with a larger --limit")
    return tasks


def clean_url(url: str) -> str:
    second_http = url.find("https://", 8)
    if second_http > -1:
        url = url[:second_http]
    second_http = url.find("http://", 7)
    if second_http > -1:
        url = url[:second_http]
    return url.rstrip("，。；、,.);]")


def canonicalize_url(url: str) -> str:
    parts = urlsplit(clean_url(url))
    host = (parts.hostname or "").lower()
    if host.startswith("www."):
        host = host[4:]
    if host in {"twitter.com", "fxtwitter.com", "fixupx.com"}:
        host = "x.com"
    netloc = host
    if parts.port:
        netloc = f"{netloc}:{parts.port}"
    path = parts.path.rstrip("/")
    return urlunsplit(("https", netloc, path, "", ""))


def extract_links(tasks: list[dict]) -> list[LinkItem]:
    grouped: dict[str, LinkItem] = {}
    for task in tasks:
        text = "\n".join(str(task.get(field) or "") for field in TEXT_FIELDS)
        raw_links = [clean_url(match.group(0)) for match in URL_RE.finditer(text)]
        task_links = []
        for raw in raw_links:
            canonical = canonicalize_url(raw)
            if canonical not in task_links:
                task_links.append(canonical)
            if canonical not in grouped:
                grouped[canonical] = LinkItem(canonical_url=canonical, url=raw, task_ids=[], titles=[])
            item = grouped[canonical]
            task_id = str(task.get("id") or "")
            title = str(task.get("title") or "")
            if task_id and task_id not in item.task_ids:
                item.task_ids.append(task_id)
            if title and title not in item.titles:
                item.titles.append(title)
    return [grouped[key] for key in sorted(grouped)]


def load_state(path: Path) -> dict:
    if not path.exists():
        return {"version": 1, "processed": {}}
    with path.open("r", encoding="utf-8") as fh:
        data = json.load(fh)
    data.setdefault("version", 1)
    data.setdefault("processed", {})
    return data


def save_state(path: Path, state: dict) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    tmp = path.with_suffix(path.suffix + ".tmp")
    with tmp.open("w", encoding="utf-8") as fh:
        json.dump(state, fh, ensure_ascii=False, indent=2)
        fh.write("\n")
    tmp.replace(path)


def run_x2md(x2md_cli: Path, output_dir: Path, url: str, thread: bool, images: bool) -> tuple[bool, str, list[str]]:
    cmd = [str(x2md_cli), url, "-o", str(output_dir), "--no-commit"]
    if thread:
        cmd.append("--thread")
    if images:
        cmd.append("--images")
    result = subprocess.run(cmd, capture_output=True, text=True, encoding="utf-8", errors="replace")
    combined = "\n".join(part for part in [result.stdout.strip(), result.stderr.strip()] if part)
    saved = []
    for line in combined.splitlines():
        if line.startswith("saved:"):
            saved.append(line.split("saved:", 1)[1].strip())
        elif line.startswith("已保存:"):
            saved.append(line.split("已保存:", 1)[1].strip())
    return result.returncode == 0, combined, saved


def commit_output(output_dir: Path, saved_paths: list[str]) -> tuple[bool, str]:
    if not saved_paths:
        return True, "no saved files to commit"
    repo = output_dir.parent
    try:
        subprocess.run(["git", "-C", str(repo), "add", str(output_dir)], check=True, capture_output=True)
        message = f"Add: X存档 - dida inbox {datetime.now().strftime('%Y-%m-%d')}"
        subprocess.run(["git", "-C", str(repo), "commit", "-m", message], check=True, capture_output=True)
        return True, message
    except subprocess.CalledProcessError as exc:
        stderr = exc.stderr.decode(errors="ignore") if isinstance(exc.stderr, bytes) else str(exc.stderr)
        return False, stderr.strip()


def write_reports(report_dir: Path, summary: dict) -> tuple[Path, Path]:
    report_dir.mkdir(parents=True, exist_ok=True)
    stamp = datetime.now().strftime("%Y%m%d-%H%M%S")
    json_path = report_dir / f"run-{stamp}.json"
    md_path = report_dir / f"run-{stamp}.md"
    with json_path.open("w", encoding="utf-8") as fh:
        json.dump(summary, fh, ensure_ascii=False, indent=2)
        fh.write("\n")

    lines = [
        f"# Dida x2md Run {stamp}",
        "",
        f"Mode: {summary['mode']}",
        f"Inbox tasks: {summary['inboxTaskCount']}",
        f"X links: {summary['linkCount']}",
        f"New links: {summary['newCount']}",
        f"Skipped processed: {summary['skippedProcessedCount']}",
        f"Successes: {summary['successCount']}",
        f"Failures: {summary['failureCount']}",
        "",
        "## New Links",
        "",
    ]
    for item in summary["newLinks"]:
        lines.append(f"- {item['canonicalUrl']}")
        lines.append(f"  - taskIds: {', '.join(item['taskIds'])}")
    if summary["failures"]:
        lines.extend(["", "## Failures", ""])
        for failure in summary["failures"]:
            lines.append(f"- {failure['canonicalUrl']}")
            lines.append(f"  - error: {failure['message']}")
    md_path.write_text("\n".join(lines) + "\n", encoding="utf-8")
    return json_path, md_path


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Process Dida inbox X links with x2md.")
    mode = parser.add_mutually_exclusive_group()
    mode.add_argument("--dry-run", action="store_true", help="List work without writing files. This is the default.")
    mode.add_argument("--run", action="store_true", help="Run x2md-cli and update processed state.")
    parser.add_argument("--dida-bin", help="Path to dida CLI.")
    parser.add_argument("--inbox-id", help="Dida inbox project id. Default resolves through dida agent context.")
    parser.add_argument("--limit", type=int, default=500, help="Inbox task read limit.")
    parser.add_argument("--x2md-cli", help="Path to x2md-cli.")
    parser.add_argument("-o", "--output-dir", default=DEFAULT_OUTPUT_DIR, help="MyNotes inbox output directory.")
    parser.add_argument("--state", default=DEFAULT_STATE_DIR / "processed.json", help="Processed state JSON path.")
    parser.add_argument("--report-dir", default=DEFAULT_STATE_DIR / "runs", help="Run report directory.")
    parser.add_argument("--thread", action="store_true", help="Pass --thread to x2md-cli.")
    parser.add_argument("--images", action="store_true", help="Pass --images to x2md-cli.")
    parser.add_argument("--events", choices=("none", "jsonl"), default="none", help="Emit machine-readable event stream.")
    parser.add_argument("--progress-every", type=int, default=10, help="Emit progress event every N processed links.")
    return parser


def main(argv: list[str] | None = None) -> int:
    args = build_parser().parse_args(argv)
    mode = "run" if args.run else "dry-run"
    event_jsonl = args.events == "jsonl"
    state_path = Path(args.state).expanduser()
    report_dir = Path(args.report_dir).expanduser()
    output_dir = Path(args.output_dir).expanduser()

    try:
        dida_bin = find_dida_binary(args.dida_bin)
        inbox_id = resolve_inbox_id(dida_bin, args.inbox_id)
        tasks = read_inbox_tasks(dida_bin, inbox_id, args.limit)
        links = extract_links(tasks)
        state = load_state(state_path)
        processed = state["processed"]
        new_links = [item for item in links if item.canonical_url not in processed]
        skipped = [item for item in links if item.canonical_url in processed]
        x2md_cli = find_x2md_cli(args.x2md_cli) if args.run else Path(args.x2md_cli or DEFAULT_X2MD_CLI)
    except Exception as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        return 2

    successes = []
    failures = []
    saved_paths: list[str] = []
    commit_result = {"ok": True, "message": "dry-run"}

    emit_event(
        event_jsonl,
        {
            "type": "start",
            "mode": mode,
            "inboxId": inbox_id,
            "inboxTaskCount": len(tasks),
            "linkCount": len(links),
            "newCount": len(new_links),
            "skippedProcessedCount": len(skipped),
        },
    )

    if args.run:
        output_dir.mkdir(parents=True, exist_ok=True)
        for index, item in enumerate(new_links, start=1):
            ok, message, saved = run_x2md(x2md_cli, output_dir, item.url, args.thread, args.images)
            record = {
                "canonicalUrl": item.canonical_url,
                "url": item.url,
                "taskIds": item.task_ids,
                "titles": item.titles,
                "message": message,
                "savedPaths": saved,
                "processedAt": now_iso(),
            }
            if ok:
                successes.append(record)
                saved_paths.extend(saved)
                emit_event(
                    event_jsonl,
                    {
                        "type": "success",
                        "index": index,
                        "total": len(new_links),
                        "canonicalUrl": item.canonical_url,
                        "url": item.url,
                        "taskIds": item.task_ids,
                        "savedPaths": saved,
                    },
                )
            else:
                failures.append(record)
                emit_event(
                    event_jsonl,
                    {
                        "type": "failure",
                        "index": index,
                        "total": len(new_links),
                        "canonicalUrl": item.canonical_url,
                        "url": item.url,
                        "taskIds": item.task_ids,
                        "message": message,
                    },
                )
            if args.progress_every > 0 and index % args.progress_every == 0:
                emit_event(
                    event_jsonl,
                    {
                        "type": "progress",
                        "done": index,
                        "total": len(new_links),
                        "successCount": len(successes),
                        "failureCount": len(failures),
                    },
                )
        commit_ok, commit_message = commit_output(output_dir, saved_paths)
        commit_result = {"ok": commit_ok, "message": commit_message}
        if successes and commit_ok:
            for record in successes:
                processed[record["canonicalUrl"]] = record
            save_state(state_path, state)

    summary = {
        "mode": mode,
        "inboxId": inbox_id,
        "inboxTaskCount": len(tasks),
        "linkCount": len(links),
        "newCount": len(new_links),
        "skippedProcessedCount": len(skipped),
        "successCount": len(successes),
        "failureCount": len(failures),
        "statePath": str(state_path),
        "outputDir": str(output_dir),
        "commit": commit_result,
        "newLinks": [
            {
                "canonicalUrl": item.canonical_url,
                "url": item.url,
                "taskIds": item.task_ids,
                "titles": item.titles,
            }
            for item in new_links
        ],
        "skippedProcessed": [
            {
                "canonicalUrl": item.canonical_url,
                "taskIds": item.task_ids,
            }
            for item in skipped
        ],
        "successes": successes,
        "failures": failures,
    }
    json_report, md_report = write_reports(report_dir, summary)
    summary["jsonReport"] = str(json_report)
    summary["markdownReport"] = str(md_report)

    emit_event(
        event_jsonl,
        {
            "type": "summary",
            "mode": mode,
            "inboxTaskCount": summary["inboxTaskCount"],
            "linkCount": summary["linkCount"],
            "newCount": summary["newCount"],
            "skippedProcessedCount": summary["skippedProcessedCount"],
            "successCount": summary["successCount"],
            "failureCount": summary["failureCount"],
            "commit": summary["commit"],
            "jsonReport": summary["jsonReport"],
            "markdownReport": summary["markdownReport"],
        },
    )
    if not event_jsonl:
        print(json.dumps(summary, ensure_ascii=False, indent=2))
    if failures or not commit_result["ok"]:
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
