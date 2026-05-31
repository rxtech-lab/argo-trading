---
name: docs-publishing
description: Set up automated documentation publishing for any repository. Authors markdown docs with frontmatter, generates API/code docs per language, drops in a stdlib uploader script plus a GitHub Actions workflow that pushes docs to the autopilot docs service on every push to main, and provisions the upload token as a repo secret. Use when a user wants to publish, upload, or sync a repo's docs to the docs service / autopilot, set up a docs CI pipeline, or wire up DOCS_UPLOAD_TOKEN.
---

# Docs Publishing

Set up a complete, token-authenticated documentation pipeline for **any** repository. On every push to `main` that touches `docs/`, a GitHub Actions job uploads the repo's markdown docs to the autopilot docs service (`https://autopilot.rxlab.app`).

This skill is portable across languages (Go, TypeScript/JavaScript, Java, Python, …). It bundles two templates:

- `templates/upload_docs.py` — the stdlib uploader (no pip dependencies).
- `templates/upload-docs.yaml` — the GitHub Actions workflow.

## Mental model

```
docs/**/*.md  ──parse frontmatter──▶  {docId: slug, content: body}
      │                                        │
      │  (per-language generators emit here)   ▼  POST in batches of 50
      ▼                          Bearer DOCS_UPLOAD_TOKEN
 handwritten + API + code docs ─────────────────▶ https://autopilot.rxlab.app
                                  /api/v1/docs/repositories/{urlencoded repo}/documents
```

The contract is **frontmatter `slug`**: every markdown file that should be published declares a unique `slug`, which becomes its remote `docId`. Files without a `slug` are skipped; duplicate slugs are a hard error.

## Steps

### 1. Author docs under `docs/`

All published docs live as markdown under `docs/` (nested folders are fine — the uploader walks recursively). Each file needs YAML frontmatter:

```markdown
---
slug: indicators/rsi          # REQUIRED, unique. Becomes the remote docId.
title: RSI (Relative Strength Index)
description: Momentum oscillator — configuration, values, signals.
---

# RSI (Relative Strength Index)

Body markdown here…
```

Rules enforced by the uploader:
- **No `slug` → skipped** (logged to stderr).
- **Duplicate `slug` → the run fails** (`exit 1`). Keep slugs unique across the whole tree.
- `title` / `description` are conventional and used by index generators; only `slug` is required for upload.

### 2. Produce each doc TYPE

Emit everything into `docs/` so one uploader handles all of it. Add frontmatter (`slug`/`title`/`description`) to each generated/authored file — for generated output, post-process or template a header so the `slug` is present.

- **Design / architecture docs** — handwritten markdown. *Always* include these (overview, data flow, key decisions). No tool; just write them under `docs/`.
- **API docs (OpenAPI / Swagger)** — commit the spec under `docs/` (e.g. `docs/api/openapi.yaml`) and, if you want rendered prose, generate markdown from it (e.g. `widdershins openapi.yaml -o docs/api/reference.md`). Give the rendered file a `slug`.
- **Code docs**, per language — generate into `docs/` and ensure each emitted page carries frontmatter:
  - **Go** → [`gomarkdoc`](https://github.com/princjef/gomarkdoc): `gomarkdoc ./... --output docs/api/{{.Dir}}.md`
  - **TypeScript / JavaScript** → [TypeDoc](https://typedoc.org) with the markdown plugin: `typedoc --plugin typedoc-plugin-markdown --out docs/api src/index.ts` (or JSDoc + `jsdoc-to-markdown`: `jsdoc2md src/**/*.js > docs/api/reference.md`)
  - **Java** → Javadoc: `javadoc -d docs/api -sourcepath src/main/java <packages>` (HTML; for markdown use a doclet, or keep HTML and publish via Pages instead)

> Tip: if a generator can't emit frontmatter, add a small post-process step (sed/python) to prepend a `slug` derived from the file path before the upload step runs.

### 3. Drop in the uploader script

Copy the bundled template into the repo:

```bash
mkdir -p scripts
cp .claude/skills/docs-publishing/templates/upload_docs.py scripts/upload_docs.py
chmod +x scripts/upload_docs.py
```

What it does (stdlib only — `urllib`, no pip installs):
- Walks `--docs-dir` (default `docs`), parses frontmatter, keeps files with a `slug`.
- Builds `{"documents": [{"docId": slug, "content": body}, …]}`.
- POSTs in batches of 50 to `{endpoint}/api/v1/docs/repositories/{urlencoded repo-id}/documents` with header `Authorization: Bearer <token>`.
- Config via env (CLI flags override): `DOCS_ENDPOINT` (default `https://autopilot.rxlab.app`), `DOCS_REPOSITORY_ID`, `DOCS_UPLOAD_TOKEN`. Supports `--dry-run`.

### 4. Drop in the workflow

```bash
mkdir -p .github/workflows
cp .claude/skills/docs-publishing/templates/upload-docs.yaml .github/workflows/upload-docs.yaml
```

Then edit one line — set `DOCS_REPOSITORY_ID` to your repo:

```yaml
              env:
                  DOCS_ENDPOINT: https://autopilot.rxlab.app
                  DOCS_REPOSITORY_ID: <OWNER>/<REPO>      # e.g. rxtech-lab/argo-trading
                  DOCS_UPLOAD_TOKEN: ${{ secrets.DOCS_UPLOAD_TOKEN }}
              run: python scripts/upload_docs.py
```

Triggers: push to `main` filtered to `paths: [docs/**, scripts/upload_docs.py, .github/workflows/upload-docs.yaml]`, plus manual `workflow_dispatch`. A `concurrency` group with `cancel-in-progress: false` prevents overlapping uploads from racing. If you also generate code/API docs in CI, add those generation steps **before** the "Upload docs" step in this same job.

### 5. Provision the upload token as a repo secret

Obtain a token from the github-pm docs service:

```
POST /api/v1/docs/repositories/[id]/upload-tokens
```

The token is prefixed `dput_` (per-upload) or `dpat_` (personal access). Then set it as a GitHub Actions secret with the `gh` CLI:

```bash
gh secret set DOCS_UPLOAD_TOKEN --body "<dput_ or dpat_ token>" --repo <OWNER>/<REPO>
```

(Run `gh auth login` first if needed. To verify it exists: `gh secret list --repo <OWNER>/<REPO>`.)

### 6. Verify

```bash
# 1. Local dry run — parses + reports, NO network call. Confirms frontmatter/slugs are valid.
python scripts/upload_docs.py --dry-run

# 2. Commit, push to main (or open+merge a PR touching docs/), then trigger manually:
gh workflow run upload-docs.yaml --repo <OWNER>/<REPO>

# 3. Watch the run and confirm it succeeds (look for the per-batch "-> 200" lines / jobId in logs):
gh run watch --repo <OWNER>/<REPO>
gh run list --workflow upload-docs.yaml --repo <OWNER>/<REPO>
```

A green run with `-> 200` responses per batch means the docs are uploaded and ready on the docs service.

## Reference: API contract

- **Endpoint:** `POST {DOCS_ENDPOINT}/api/v1/docs/repositories/{urlencoded DOCS_REPOSITORY_ID}/documents`
- **Auth:** `Authorization: Bearer {DOCS_UPLOAD_TOKEN}`
- **Body:** `{"documents": [{"docId": "<slug>", "content": "<markdown body>"}, …]}` (≤ 50 per request)
- **Token source:** `POST /api/v1/docs/repositories/[id]/upload-tokens` (github-pm) → `dput_…` / `dpat_…`

## Troubleshooting

- `no documents with slug frontmatter found` → no file under `docs/` has a `slug:` line. Add frontmatter.
- `error: duplicate slug …` → two files share a slug. Make them unique.
- `HTTP 401/403` → bad/missing token, or the secret isn't set. Re-run step 5; check `gh secret list`.
- `HTTP 404` → wrong `DOCS_REPOSITORY_ID`, or the repository isn't registered on the docs service.
- Workflow didn't trigger on push → your change didn't touch a path in the `paths:` filter. Edit a file under `docs/`, or run it manually with `workflow_dispatch`.
