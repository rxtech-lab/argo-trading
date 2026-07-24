import "server-only";

/**
 * Resolve the macOS download link from the RxLab Autopilot release API instead
 * of hard-coding a URL. See the `autopilot-release-api` skill / contract:
 *   GET /api/v1/release-repositories/by-name/{owner}/{repo}                → repo id
 *   GET /api/v1/release-repositories/{repoId}/versions/latest             → release + files
 * The returned URL is the stable, application-owned download endpoint that
 * 302-redirects to the (public) object storage URL.
 */

const DEFAULT_BASE_URL = "https://autopilot.rxlab.app";
const DEFAULT_REPO = "rxtech-lab/argo-trading-macOS";

// Cache the release lookup for an hour; releases don't change per-request.
const REVALIDATE_SECONDS = 3600;

type LatestVersion = {
  release?: { id?: string; tagName?: string } | null;
  files?: Array<{ id?: string; filename?: string }> | null;
  primaryDownloadUrl?: string | null;
};

function baseUrl(): string {
  return (process.env.AUTOPILOT_BASE_URL ?? DEFAULT_BASE_URL).replace(/\/+$/, "");
}

function repo(): string {
  return process.env.AUTOPILOT_RELEASE_REPO ?? DEFAULT_REPO;
}

/** Pick the best macOS artifact from a release's file list. */
function selectFile(
  files: Array<{ id?: string; filename?: string }>,
): { id?: string; filename?: string } | null {
  const usable = files.filter((f) => f.id && f.filename);
  if (usable.length === 0) return null;

  const score = (name: string) => {
    const n = name.toLowerCase();
    let s = 0;
    if (n.endsWith(".dmg")) s += 30;
    else if (n.endsWith(".pkg")) s += 20;
    else if (n.endsWith(".zip")) s += 10;
    // Prefer Apple Silicon builds when several are present.
    if (n.includes("arm64") || n.includes("aarch64") || n.includes("apple")) s += 5;
    return s;
  };

  return [...usable].sort(
    (a, b) => score(b.filename!) - score(a.filename!),
  )[0];
}

async function fetchJson(url: string, token?: string): Promise<Response> {
  return fetch(url, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
    next: { revalidate: REVALIDATE_SECONDS },
  });
}

/**
 * Return the download URL for the latest macOS release, or `null` if it can't
 * be resolved. Falls back to `DOWNLOAD_URL` when the API is unreachable or no
 * token is configured for a private release listing.
 */
export async function getLatestMacDownloadUrl(): Promise<string | null> {
  const fallback = process.env.DOWNLOAD_URL || null;
  const token = process.env.AUTOPILOT_TOKEN || undefined;
  const base = baseUrl();
  const [owner, name] = repo().split("/");

  if (!owner || !name) return fallback;

  try {
    // Anonymous, public listing: the by-name endpoint hands back a ready-made
    // primaryDownloadUrl. Use it directly when there is no token.
    if (!token) {
      const res = await fetchJson(
        `${base}/api/v1/release-repositories/by-name/${owner}/${name}/versions/latest`,
      );
      if (!res.ok) return fallback;
      const data = (await res.json()) as LatestVersion;
      const primary = data.primaryDownloadUrl;
      if (primary) return primary.startsWith("http") ? primary : `${base}${primary}`;
      return fallback;
    }

    // Authenticated (private listing): resolve the repo id, then the latest
    // version, and build the stable download endpoint from the ids.
    const repoRes = await fetchJson(
      `${base}/api/v1/release-repositories/by-name/${owner}/${name}`,
      token,
    );
    if (!repoRes.ok) return fallback;
    const repoData = (await repoRes.json()) as { repository?: { id?: string } };
    const repoId = repoData.repository?.id;
    if (!repoId) return fallback;

    const latestRes = await fetchJson(
      `${base}/api/v1/release-repositories/${repoId}/versions/latest`,
      token,
    );
    if (!latestRes.ok) return fallback;
    const latest = (await latestRes.json()) as LatestVersion;

    const releaseId = latest.release?.id;
    const file = selectFile(latest.files ?? []);
    if (!releaseId || !file?.id) return fallback;

    return `${base}/api/v1/release-repositories/${repoId}/releases/${releaseId}/files/${file.id}/download`;
  } catch {
    return fallback;
  }
}
