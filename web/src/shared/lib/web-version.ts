// Web app version & build metadata, the web-side analog of the apiserver's
// /api/v1/version payload (see api/v1/version/version.go).
//
// Two sources feed it:
//   - Build-time fields (gitVersion/gitCommit/gitTreeState/buildDate) are
//     baked at `next build` from NEXT_PUBLIC_* env vars injected by
//     web/Dockerfile, which CI fills from git — the same source that tags the
//     image, so the page matches the deployed image. NEXT_PUBLIC_* vars are
//     inlined into both the server and client bundles. In local dev (no env
//     vars) they fall back to package.json / placeholders so the page still
//     renders.
//   - Runtime fields (nodeVersion/platform) come from `process` and are only
//     meaningful server-side, so getWebVersionInfo() must be called from a
//     Server Component / route handler, never the browser.
import pkg from '../../../package.json';

const gitVersion = process.env.NEXT_PUBLIC_WEB_VERSION || pkg.version;
const gitCommit = process.env.NEXT_PUBLIC_WEB_GIT_COMMIT || '';
const gitTreeState = process.env.NEXT_PUBLIC_WEB_GIT_TREE_STATE || '';
const buildDate = process.env.NEXT_PUBLIC_WEB_BUILD_DATE || '';

// Backward-compatible single-string export; safe in client components.
export const WEB_VERSION: string = gitVersion;

// Build-time metadata, safe in client components (NEXT_PUBLIC_* is inlined).
// Excludes the Node runtime fields, which only exist server-side.
export const WEB_BUILD = { gitVersion, gitCommit, gitTreeState, buildDate };

export interface WebVersionInfo {
  major: string;
  minor: string;
  gitVersion: string;
  gitCommit: string;
  gitTreeState: string;
  buildDate: string;
  nodeVersion: string;
  nextVersion: string;
  platform: string;
}

// Pulls "0" / "1" out of versions like "0.1.41-SNAPSHOT-14bc29e" or "v0.1.0".
// Returns empty strings when the version isn't in a recognizable form.
export function parseMajorMinor(version: string): { major: string; minor: string } {
  const m = version.match(/^v?(\d+)\.(\d+)/);
  return { major: m?.[1] ?? '', minor: m?.[2] ?? '' };
}

// Server-only: reads Node runtime fields (process.version/platform) that are
// undefined in the browser bundle. Call from RSC / route handlers only.
export function getWebVersionInfo(): WebVersionInfo {
  const { major, minor } = parseMajorMinor(gitVersion);
  return {
    major,
    minor,
    gitVersion,
    gitCommit,
    gitTreeState,
    buildDate,
    nodeVersion: process.version,
    nextVersion: pkg.dependencies.next,
    platform: `${process.platform}/${process.arch}`,
  };
}
