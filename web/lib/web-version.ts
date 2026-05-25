// Web app version. In release/container builds, NEXT_PUBLIC_WEB_VERSION is
// injected at `next build` time from scripts/version.sh — the same source
// that tags the image — so the footer matches the deployed image tag. In
// local dev (no env var), fall back to package.json's version.
import pkg from '../package.json';

export const WEB_VERSION: string = process.env.NEXT_PUBLIC_WEB_VERSION || pkg.version;
