// Backend version info returned by /api/v1/version. (The web app's own build
// version lives in @shared/lib/web-version.)
export interface VersionInfo {
  major?: string;
  minor?: string;
  gitVersion?: string;
  gitCommit?: string;
  gitTreeState?: string;
  buildDate?: string;
  goVersion?: string;
  compiler?: string;
  platform?: string;
}
