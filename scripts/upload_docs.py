#!/usr/bin/env python3
"""
Upload markdown docs to the autopilot docs API.

Walks the docs directory, parses YAML frontmatter from each markdown file,
and uploads files that declare a `slug` field as their document id.

Usage:
    DOCS_UPLOAD_TOKEN=dput_... DOCS_REPOSITORY_ID=repo_xxx \
        python scripts/upload_docs.py

    # dry run (no network call):
    python scripts/upload_docs.py --dry-run
"""

import argparse
import json
import os
import re
import sys
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path

DEFAULT_ENDPOINT = "https://autopilot.rxlab.app"
DEFAULT_DOCS_DIR = "docs"
DEFAULT_BATCH_SIZE = 50


def parse_frontmatter(text: str) -> tuple[dict, str]:
    """Parse YAML frontmatter at the top of `text`. Returns (fields, body)."""
    match = re.match(r"^---\s*\n(.*?)\n---\s*\n?(.*)$", text, re.DOTALL)
    if not match:
        return {}, text

    fields: dict[str, str] = {}
    for line in match.group(1).splitlines():
        kv = re.match(r"^([A-Za-z_][\w-]*)\s*:\s*(.*?)\s*$", line)
        if kv:
            value = kv.group(2).strip()
            if (value.startswith('"') and value.endswith('"')) or (
                value.startswith("'") and value.endswith("'")
            ):
                value = value[1:-1]
            fields[kv.group(1)] = value
    return fields, match.group(2)


def collect_docs(docs_dir: Path) -> list[dict]:
    documents: list[dict] = []
    seen: dict[str, Path] = {}

    for md_file in sorted(docs_dir.rglob("*.md")):
        text = md_file.read_text(encoding="utf-8")
        fields, body = parse_frontmatter(text)
        slug = fields.get("slug")
        rel = md_file.relative_to(docs_dir)

        if not slug:
            print(f"skip {rel}: no `slug` in frontmatter", file=sys.stderr)
            continue

        if slug in seen:
            print(
                f"error: duplicate slug `{slug}` in {rel} and {seen[slug]}",
                file=sys.stderr,
            )
            sys.exit(1)
        seen[slug] = rel

        documents.append({"docId": slug, "content": body.lstrip("\n")})

    return documents


def upload_batch(url: str, token: str, batch: list[dict]) -> None:
    payload = json.dumps({"documents": batch}).encode("utf-8")
    req = urllib.request.Request(
        url,
        data=payload,
        headers={
            "Authorization": f"Bearer {token}",
            "Content-Type": "application/json",
        },
        method="POST",
    )
    try:
        with urllib.request.urlopen(req) as resp:
            body = resp.read().decode("utf-8", errors="replace")
            print(f"  -> {resp.status} {body[:200]}")
    except urllib.error.HTTPError as e:
        err_body = e.read().decode("utf-8", errors="replace")
        print(f"error: HTTP {e.code} {e.reason} — {err_body}", file=sys.stderr)
        sys.exit(1)
    except urllib.error.URLError as e:
        print(f"error: request failed — {e.reason}", file=sys.stderr)
        sys.exit(1)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--docs-dir",
        default=os.environ.get("DOCS_DIR", DEFAULT_DOCS_DIR),
        help=f"directory containing markdown files (default: {DEFAULT_DOCS_DIR})",
    )
    parser.add_argument(
        "--endpoint",
        default=os.environ.get("DOCS_ENDPOINT", DEFAULT_ENDPOINT),
        help=f"API base URL (default: {DEFAULT_ENDPOINT})",
    )
    parser.add_argument(
        "--repo-id",
        default=os.environ.get("DOCS_REPOSITORY_ID"),
        help="docs repository id (env: DOCS_REPOSITORY_ID)",
    )
    parser.add_argument(
        "--token",
        default=os.environ.get("DOCS_UPLOAD_TOKEN"),
        help="bearer token, dput_... or dpat_... (env: DOCS_UPLOAD_TOKEN)",
    )
    parser.add_argument(
        "--batch-size",
        type=int,
        default=DEFAULT_BATCH_SIZE,
        help=f"docs per request (default: {DEFAULT_BATCH_SIZE})",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="parse and report, but do not upload",
    )
    args = parser.parse_args()

    docs_dir = Path(args.docs_dir)
    if not docs_dir.is_dir():
        print(f"error: docs dir not found: {docs_dir}", file=sys.stderr)
        return 2

    documents = collect_docs(docs_dir)
    if not documents:
        print("no documents with `slug` frontmatter found")
        return 0

    print(f"found {len(documents)} document(s):")
    for d in documents:
        print(f"  - {d['docId']} ({len(d['content'])} bytes)")

    if args.dry_run:
        return 0

    if not args.repo_id:
        print(
            "error: --repo-id or DOCS_REPOSITORY_ID is required",
            file=sys.stderr,
        )
        return 2
    if not args.token:
        print(
            "error: --token or DOCS_UPLOAD_TOKEN is required",
            file=sys.stderr,
        )
        return 2

    repo_id_encoded = urllib.parse.quote(args.repo_id, safe="")
    url = f"{args.endpoint.rstrip('/')}/api/v1/docs/repositories/{repo_id_encoded}/documents"
    print(f"uploading to {url}")

    for i in range(0, len(documents), args.batch_size):
        batch = documents[i : i + args.batch_size]
        print(f"batch {i // args.batch_size + 1}: {len(batch)} doc(s)")
        upload_batch(url, args.token, batch)

    print("done")
    return 0


if __name__ == "__main__":
    sys.exit(main())
