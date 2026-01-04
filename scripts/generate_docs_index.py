#!/usr/bin/env python3
"""
Generate docs/index.md by scanning all markdown files and extracting
their title and description from YAML frontmatter.
"""

import os
import re
from pathlib import Path


def extract_frontmatter(file_path: Path) -> dict:
    """Extract title and description from YAML frontmatter."""
    with open(file_path, "r", encoding="utf-8") as f:
        content = f.read()

    # Match YAML frontmatter between --- delimiters
    match = re.match(r"^---\s*\n(.*?)\n---", content, re.DOTALL)
    if not match:
        return {}

    frontmatter = match.group(1)
    result = {}

    # Extract title
    title_match = re.search(r"^title:\s*(.+)$", frontmatter, re.MULTILINE)
    if title_match:
        result["title"] = title_match.group(1).strip().strip('"\'')

    # Extract description
    desc_match = re.search(r"^description:\s*(.+)$", frontmatter, re.MULTILINE)
    if desc_match:
        result["description"] = desc_match.group(1).strip().strip('"\'')

    return result


def get_section_name(folder: str) -> str:
    """Convert folder name to a human-readable section name."""
    if folder == "":
        return "Guides"
    return folder.replace("-", " ").replace("_", " ").title()


def generate_index(docs_dir: Path) -> str:
    """Generate the index.md content."""
    # Collect all markdown files grouped by directory
    docs_by_folder: dict[str, list[dict]] = {}

    for md_file in sorted(docs_dir.rglob("*.md")):
        # Skip index.md itself
        if md_file.name == "index.md":
            continue

        relative_path = md_file.relative_to(docs_dir)
        folder = str(relative_path.parent) if relative_path.parent != Path(".") else ""

        frontmatter = extract_frontmatter(md_file)
        title = frontmatter.get("title", md_file.stem.replace("-", " ").title())
        description = frontmatter.get("description", "")

        if folder not in docs_by_folder:
            docs_by_folder[folder] = []

        docs_by_folder[folder].append(
            {"path": str(relative_path), "title": title, "description": description}
        )

    # Generate markdown content
    lines = [
        "---",
        "title: Documentation",
        "description: Argo Trading documentation index",
        "---",
        "",
        "# Documentation",
        "",
    ]

    # Sort folders: root ("") first, then alphabetically
    sorted_folders = sorted(docs_by_folder.keys(), key=lambda x: (x != "", x))

    for folder in sorted_folders:
        docs = docs_by_folder[folder]
        section_name = get_section_name(folder)

        lines.append(f"## {section_name}")
        lines.append("")

        # Sort docs by title within each section
        for doc in sorted(docs, key=lambda x: x["title"].lower()):
            if doc["description"]:
                lines.append(f"- [{doc['title']}]({doc['path']}) - {doc['description']}")
            else:
                lines.append(f"- [{doc['title']}]({doc['path']})")

        lines.append("")

    return "\n".join(lines)


def main():
    # Find the docs directory relative to this script
    script_dir = Path(__file__).parent
    repo_root = script_dir.parent
    docs_dir = repo_root / "docs"

    if not docs_dir.exists():
        print(f"Error: docs directory not found at {docs_dir}")
        return 1

    index_content = generate_index(docs_dir)
    index_path = docs_dir / "index.md"

    with open(index_path, "w", encoding="utf-8") as f:
        f.write(index_content)

    print(f"Generated {index_path}")
    return 0


if __name__ == "__main__":
    exit(main())
