#!/usr/bin/env bash
#
# gen-remoteconfigschema.sh — regenerate the pre-built RemoteConfigSchema library.
#
# For every stable OTel Collector release (and each distribution below) this downloads
# the host-platform binary, runs `<binary> components` to read the authoritative
# component catalog, and renders a RemoteConfigSchema YAML via `opampctl generate
# remoteconfigschema`. The catalog can only be read from the compiled binary — the
# component *type* names (e.g. memory_limiter, health_check, oidcauth) are not
# derivable from the release manifest's Go module paths — so a binary per version is
# required.
#
# Disk-safe: each binary is downloaded, used, and deleted before the next, so peak
# usage stays around one distribution (~350MB) rather than accumulating.
# Resumable: an existing output file is skipped unless FORCE=1.
#
# Usage:
#   hack/gen-remoteconfigschema.sh                 # all stable releases, all distributions
#   VERSIONS="v0.130.0 v0.129.0" hack/gen-remoteconfigschema.sh
#   DISTS="otelcol-contrib" hack/gen-remoteconfigschema.sh
#   FROM=v0.100.0 hack/gen-remoteconfigschema.sh   # only tags >= FROM
#   FORCE=1 hack/gen-remoteconfigschema.sh         # re-render existing files
#
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="${OUT_DIR:-$REPO_ROOT/configs/apiserver/initial/remoteconfigschema}"
OPAMPCTL="${OPAMPCTL:-$REPO_ROOT/.bin/opampctl}"
DISTS="${DISTS:-otelcol otelcol-contrib otelcol-k8s}"
RELEASES_REPO="open-telemetry/opentelemetry-collector-releases"
NAMESPACE="${NAMESPACE:-default}"

log() { printf '%s\n' "$*" >&2; }

# host platform -> release asset suffix (e.g. darwin_arm64, linux_amd64)
detect_platform() {
  local os arch
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  arch="$(uname -m)"
  case "$arch" in
    x86_64 | amd64) arch=amd64 ;;
    aarch64 | arm64) arch=arm64 ;;
  esac
  printf '%s_%s' "$os" "$arch"
}

# list all stable release tags (v<major>.<minor>.<patch>), oldest first
list_versions() {
  git ls-remote --tags "https://github.com/$RELEASES_REPO" 2>/dev/null |
    sed -n 's#.*refs/tags/\(v[0-9][0-9.]*\)$#\1#p' |
    grep -vE 'nightly|rc|alpha|beta' | sort -V -u
}

ensure_opampctl() {
  if [[ -x "$OPAMPCTL" ]]; then return; fi
  log "==> building opampctl -> $OPAMPCTL"
  (cd "$REPO_ROOT" && go build -o "$OPAMPCTL" ./cmd/opampctl)
}

PLATFORM="$(detect_platform)"

# gen_one <dist> <tag>  — returns 0 on success, 1 on skip/failure (never aborts the run)
gen_one() {
  local dist="$1" tag="$2" ver="${2#v}"
  local out="$OUT_DIR/${dist}-${ver}.yaml"

  if [[ -s "$out" && "${FORCE:-0}" != "1" ]]; then
    return 0 # already generated (non-empty)
  fi

  local work
  work="$(mktemp -d)"
  # shellcheck disable=SC2064
  trap "rm -rf '$work'" RETURN

  local url="https://github.com/$RELEASES_REPO/releases/download/${tag}/${dist}_${ver}_${PLATFORM}.tar.gz"
  if ! curl -sfL --max-time 300 "$url" -o "$work/a.tgz" 2>/dev/null; then
    return 1 # no asset for this dist/version/platform
  fi

  if ! tar xzf "$work/a.tgz" -C "$work" "$dist" 2>/dev/null; then
    return 1 # archive layout unexpected
  fi
  rm -f "$work/a.tgz"

  if ! "$work/$dist" components >"$work/components.yaml" 2>/dev/null; then
    return 1 # binary predates the `components` subcommand
  fi
  rm -f "$work/$dist"

  if ! "$OPAMPCTL" generate remoteconfigschema \
    --from "$work/components.yaml" \
    --binary "$dist" --version "$ver" --name "${dist}-${ver}" \
    -n "$NAMESPACE" -o yaml >"$work/schema.yaml" 2>/dev/null; then
    return 1
  fi

  # Defensive: never commit an empty or malformed schema even if the generator
  # misreports success on an unrecognized components format.
  if [[ ! -s "$work/schema.yaml" ]] || ! grep -q '^kind: RemoteConfigSchema' "$work/schema.yaml"; then
    return 1
  fi

  mv "$work/schema.yaml" "$out"
  log "OK   ${dist}-${ver}"
  return 0
}

# runComponents downloads <dist>@<tag> for the host platform, captures its `components`
# output to <destfile>, and deletes the binary. Returns non-zero if unavailable.
run_components() {
  local dist="$1" tag="$2" ver="${2#v}" destfile="$3"
  local work
  work="$(mktemp -d)"
  # shellcheck disable=SC2064
  trap "rm -rf '$work'" RETURN

  local url="https://github.com/$RELEASES_REPO/releases/download/${tag}/${dist}_${ver}_${PLATFORM}.tar.gz"
  if ! curl -sfL --max-time 300 "$url" -o "$work/a.tgz" 2>/dev/null; then
    return 1
  fi
  if ! tar xzf "$work/a.tgz" -C "$work" "$dist" 2>/dev/null; then
    return 1
  fi

  "$work/$dist" components >"$destfile" 2>/dev/null
}

# ensure_dictionary builds the module->type dictionary sources (core + contrib
# `components` output) once per run. Used to resolve Linux-only distributions that
# cannot be run on the host. DICT_VERSION defaults to the newest processed release.
DICT_DIR=""
ensure_dictionary() {
  if [[ -n "$DICT_DIR" ]]; then
    return 0
  fi

  DICT_DIR="$(mktemp -d)"
  # Type names are stable across versions, so a union of a few releases across the
  # range resolves nearly every component of every version. DICT_VERSIONS defaults to
  # the newest release (exact for the default-seeded latest schema).
  local dversions="${DICT_VERSIONS:-${DICT_VERSION:?DICT_VERSION must be set before deriving schemas}}"

  log "==> building module->type dictionary from otelcol/otelcol-contrib @ [$dversions]"
  for dver in $dversions; do
    run_components otelcol "$dver" "$DICT_DIR/otelcol-${dver}.components.yaml" ||
      log "warn: could not capture otelcol@$dver components for dictionary"
    run_components otelcol-contrib "$dver" "$DICT_DIR/otelcol-contrib-${dver}.components.yaml" ||
      log "warn: could not capture otelcol-contrib@$dver components for dictionary"
  done
}

# gen_derived generates a schema for a Linux-only distribution by resolving its build
# manifest's module paths through the host-runnable core+contrib dictionary, then
# feeding a synthesized `components` document to the generator. Used for otelcol-k8s on
# platforms where it ships no binary.
gen_derived() {
  local dist="$1" tag="$2" ver="${2#v}"
  local out="$OUT_DIR/${dist}-${ver}.yaml"

  if [[ -s "$out" && "${FORCE:-0}" != "1" ]]; then
    return 0
  fi

  ensure_dictionary

  local work
  work="$(mktemp -d)"
  # shellcheck disable=SC2064
  trap "rm -rf '$work'" RETURN

  local manifest_url="https://raw.githubusercontent.com/$RELEASES_REPO/${tag}/distributions/${dist}/manifest.yaml"
  if ! curl -sfL --max-time 60 "$manifest_url" -o "$work/manifest.yaml" 2>/dev/null; then
    return 1 # distribution did not exist at this version
  fi

  # A non-zero exit means some modules were unresolved; the synthesized document still
  # contains every resolved component, so keep it and just warn. The newest version
  # (the default-seeded one) resolves fully against the newest dictionary.
  python3 "$REPO_ROOT/hack/synth_k8s_components.py" \
    "$work/manifest.yaml" "$dist" "$ver" \
    "$DICT_DIR"/*.components.yaml >"$work/components.yaml" 2>"$work/synth.err" ||
    log "warn ${dist}-${ver}: $(tail -1 "$work/synth.err")"

  if [[ ! -s "$work/components.yaml" ]]; then
    return 1
  fi

  if ! "$OPAMPCTL" generate remoteconfigschema \
    --from "$work/components.yaml" \
    --binary "$dist" --version "$ver" --name "${dist}-${ver}" \
    -n "$NAMESPACE" -o yaml >"$work/schema.yaml" 2>/dev/null; then
    return 1
  fi

  if [[ ! -s "$work/schema.yaml" ]] || ! grep -q '^kind: RemoteConfigSchema' "$work/schema.yaml"; then
    return 1
  fi

  mv "$work/schema.yaml" "$out"
  log "OK   ${dist}-${ver} (derived)"

  return 0
}

main() {
  ensure_opampctl
  mkdir -p "$OUT_DIR"

  local versions
  if [[ -n "${VERSIONS:-}" ]]; then
    versions="$(printf '%s\n' $VERSIONS)"
  else
    versions="$(list_versions)"
  fi
  if [[ -n "${FROM:-}" ]]; then
    versions="$(printf '%s\n' "$versions" | sort -V | awk -v f="$FROM" '$0==f{ok=1} ok')"
  fi

  # DICT_VERSION (used to resolve Linux-only distributions) defaults to the newest
  # release being processed.
  export DICT_VERSION="${DICT_VERSION:-$(printf '%s\n' "$versions" | sort -V | tail -1)}"

  log "==> platform=$PLATFORM dists=[$DISTS] out=$OUT_DIR"
  local ok=0 skip=0
  for tag in $versions; do
    for dist in $DISTS; do
      if gen_one "$dist" "$tag"; then
        ok=$((ok + 1))
      elif [[ "$dist" == "otelcol-k8s" ]] && gen_derived "$dist" "$tag"; then
        ok=$((ok + 1))
      else
        skip=$((skip + 1))
      fi
    done
    log "... through $tag (generated=$ok skipped=$skip)"
  done

  [[ -n "$DICT_DIR" ]] && rm -rf "$DICT_DIR"
  log "==> done. generated=$ok skipped=$skip out=$OUT_DIR"
}

main "$@"
