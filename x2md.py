#!/usr/bin/env python3
"""x2md skill wrapper: save X content to MyNotes and commit it."""

import argparse
import os
import re
import shutil
import subprocess
import sys
import tempfile
from datetime import datetime
from pathlib import Path


DEFAULT_OUT = Path.home() / "Desktop" / "repos" / "MyNotes" / "00_Inbox"
URL_RE = re.compile(r"https?://(?:www\.)?(?:x\.com|twitter\.com|fxtwitter\.com|fixupx\.com)/\S+")


def find_binary() -> Path:
    env_bin = os.environ.get("X2MD_BIN")
    if env_bin:
        path = Path(env_bin).expanduser()
        if path.exists():
            return path
        raise FileNotFoundError(f"X2MD_BIN does not exist: {path}")

    for name in ("x2md", "x2md.exe"):
        found = shutil.which(name)
        if found:
            return Path(found)

    for candidate in (Path.home() / "bin" / "x2md", Path.home() / "bin" / "x2md.exe"):
        if candidate.exists():
            return candidate

    raise FileNotFoundError("x2md binary not found in PATH or ~/bin")


def parse_field(content: str, field: str) -> str | None:
    m = re.search(rf'^{field}:\s*["\']?([^"\'\n]+)["\']?', content, re.MULTILINE)
    return m.group(1).strip() if m else None


def sanitize_author(raw: str) -> str:
    raw = raw.lstrip("@").replace(".", "-").replace("_", "-")
    raw = re.sub(r"[^\w\-]", "", raw)
    return raw or "unknown"


def normalize_date(raw: str) -> str:
    if not raw:
        return datetime.now().strftime("%Y%m%d")
    if "T" in raw:
        return raw.split("T")[0].replace("-", "")
    return raw.replace("-", "")


def unique_dest(output_dir: Path, author: str, date: str) -> Path:
    dest = output_dir / f"x-{author}-{date}.md"
    if not dest.exists():
        return dest

    i = 2
    while True:
        candidate = output_dir / f"x-{author}-{date}-{i}.md"
        if not candidate.exists():
            return candidate
        i += 1


def commit_saved_files(output_dir: Path, saved_paths: list[Path]) -> None:
    repo = output_dir.parent
    subprocess.run(["git", "-C", str(repo), "add", str(output_dir)], check=True, capture_output=True)
    authors = []
    for path in saved_paths:
        match = re.match(r"^x-(.+)-\d{8}(?:-\d+)?$", path.stem)
        authors.append(match.group(1) if match else path.stem)
    message = "Add: X存档 - " + ", ".join(sorted(set(authors)))
    subprocess.run(["git", "-C", str(repo), "commit", "-m", message], check=True, capture_output=True)


def main(argv: list[str] | None = None) -> int:
    ap = argparse.ArgumentParser(description="x2md: X (Twitter) URL to Markdown")
    ap.add_argument("urls", nargs="+", help="X/Twitter URL(s)")
    ap.add_argument("-o", "--output-dir", default=DEFAULT_OUT, help="Output dir")
    ap.add_argument("--thread", action="store_true", help="Expand thread")
    ap.add_argument("--images", action="store_true", help="Download images")
    ap.add_argument("--no-commit", action="store_true", help="Do not git commit")
    args = ap.parse_args(argv)

    invalid_urls = [url for url in args.urls if not URL_RE.fullmatch(url)]
    if invalid_urls:
        for url in invalid_urls:
            print(f"ERROR: unsupported X URL: {url}", file=sys.stderr)
        return 2

    try:
        binary = find_binary()
    except FileNotFoundError as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        return 127

    output_dir = Path(args.output_dir).expanduser()
    output_dir.mkdir(parents=True, exist_ok=True)

    saved_paths: list[Path] = []
    for url in args.urls:
        fd, tmp_name = tempfile.mkstemp(suffix=".md")
        os.close(fd)
        tmp_path = Path(tmp_name)

        cmd = [str(binary), "-o", str(tmp_path)]
        if args.thread:
            cmd.append("-thread")
        if args.images:
            cmd.append("-images")
        cmd.append(url)

        result = subprocess.run(cmd, capture_output=True, text=True, encoding="utf-8", errors="replace")
        if result.returncode != 0:
            err = (result.stderr or "").strip()
            print(f"ERROR [{url}]: {err or 'unknown failure'}", file=sys.stderr)
            tmp_path.unlink(missing_ok=True)
            continue

        content = tmp_path.read_text(encoding="utf-8")
        author = sanitize_author(parse_field(content, "author") or "unknown")
        date = normalize_date(parse_field(content, "date") or "")

        dest = unique_dest(output_dir, author, date)
        dest.write_text(content, encoding="utf-8")
        tmp_path.unlink(missing_ok=True)
        print(f"saved: {dest}")
        saved_paths.append(dest)

    if saved_paths and not args.no_commit:
        try:
            commit_saved_files(output_dir, saved_paths)
            print(f"git commit created: {len(saved_paths)} file(s)")
        except subprocess.CalledProcessError as exc:
            stderr = exc.stderr.decode(errors="ignore") if isinstance(exc.stderr, bytes) else str(exc.stderr)
            print(f"git commit skipped: {stderr.strip()}", file=sys.stderr)

    return 0 if saved_paths else 1


if __name__ == "__main__":
    sys.exit(main())
