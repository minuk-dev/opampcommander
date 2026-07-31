#!/usr/bin/env bash
#
# gen-component-configs.sh — produce the per-component config field schema (a JSON
# ComponentConfigCatalog) for one collector distribution/version by reflecting each
# component's config struct.
#
# Component config field names/types are only knowable from the components' Go config
# structs (there is no machine-readable dump), so this generates a throwaway Go program
# that imports every factory listed in the distribution's build manifest, reflects each
# CreateDefaultConfig() type into a field tree, and prints the catalog as JSON. That
# JSON is fed to `opampctl generate remoteconfigschema --component-configs`.
#
# This compiles the component set, so it is heavy (large module downloads). Prefer
# running it in CI. Peak disk is bounded by cleaning the throwaway module each run.
#
# Usage:
#   hack/gen-component-configs.sh <manifest.yaml> <command> <version> > catalog.json
#   # or fetch the manifest for a release:
#   hack/gen-component-configs.sh --release otelcol v0.157.0 > catalog.json
#
set -euo pipefail

# Never auto-download a newer Go toolchain (blocked in some environments by GOSUMDB=off);
# use the installed one. Components that require a newer toolchain are skipped via SKIP_MODULES.
export GOTOOLCHAIN="${GOTOOLCHAIN:-local}"

# Route module downloads through the public proxy. Many components depend on vanity import
# paths (code.cloudfoundry.org, software.sslmate.com, ...) that a direct (GOPROXY=direct)
# fetch cannot resolve in a restricted network; the proxy serves them from one reachable host.
export GOPROXY="${GEN_GOPROXY:-https://proxy.golang.org,direct}"

RELEASES_REPO="open-telemetry/opentelemetry-collector-releases"

# SKIP_MODULES is an awk regex of component module paths to exclude (e.g. ones that
# require a newer Go toolchain than installed). Excluded components fall back to
# existence-only validation.
SKIP_MODULES="${SKIP_MODULES:-opentelemetry.io/obi}"

log() { printf '%s\n' "$*" >&2; }

resolve_manifest() {
  if [[ "${1:-}" == "--release" ]]; then
    local dist="$2" tag="$3"
    local url="https://raw.githubusercontent.com/$RELEASES_REPO/${tag}/distributions/${dist}/manifest.yaml"
    local tmp
    tmp="$(mktemp)"
    curl -sfL --max-time 60 "$url" -o "$tmp"
    printf '%s' "$tmp"
  else
    printf '%s' "$1"
  fi
}

main() {
  local manifest command version
  if [[ "${1:-}" == "--release" ]]; then
    manifest="$(resolve_manifest "$@")"
    command="$2"
    version="${3#v}"
  else
    manifest="$1"
    command="$2"
    version="${3#v}"
  fi

  # Parse "class<TAB>importpath<TAB>modversion" from the manifest, skipping SKIP_MODULES.
  local entries
  entries="$(awk -v skip="$SKIP_MODULES" '
    /^[a-z]+:[[:space:]]*$/ { cls=$1; sub(/:$/,"",cls); next }
    /^[[:space:]]*-[[:space:]]*gomod:/ {
      path=$3; ver=$4
      if ((cls=="receivers"||cls=="processors"||cls=="exporters"||cls=="extensions"||cls=="connectors") &&
          (skip=="" || path !~ skip))
        print cls "\t" path "\t" ver
    }
  ' "$manifest")"

  # Global (not local) so the EXIT trap can still see it under `set -u`.
  work="$(mktemp -d)"
  trap 'rm -rf "$work"' EXIT

  # Generate the reflection program.
  {
    printf 'package main\n\n'
    printf 'import (\n'
    printf '\t"encoding/json"\n\t"os"\n\t"reflect"\n\t"strings"\n\t"time"\n\n'
    printf '\t"go.opentelemetry.io/collector/component"\n'
    local i=0
    while IFS=$'\t' read -r cls path ver; do
      [[ -z "$path" ]] && continue
      printf '\ta%d "%s"\n' "$i" "$path"
      i=$((i + 1))
    done <<<"$entries"
    printf ')\n\n'
    printf 'type cfgFactory interface {\n\tType() component.Type\n\tCreateDefaultConfig() component.Config\n}\n\n'
    printf 'func factories() []entry {\n\treturn []entry{\n'
    i=0
    while IFS=$'\t' read -r cls path ver; do
      [[ -z "$path" ]] && continue
      printf '\t\t{"%s", a%d.NewFactory()},\n' "$cls" "$i"
      i=$((i + 1))
    done <<<"$entries"
    printf '\t}\n}\n'
    cat "$(dirname "${BASH_SOURCE[0]}")/componentconfig_reflect.go.tmpl"
  } >"$work/main.go"

  # Pin each factory module to the manifest's version, then build the field catalog.
  (
    cd "$work"
    go mod init componentconfiggen >/dev/null 2>&1
    log "==> resolving ${command}@${version} component modules (this compiles the component set)"
    # Write pinned requires directly (fast, no per-module graph resolution), then let a
    # single `go mod tidy` resolve and download the whole set at once.
    while IFS=$'\t' read -r cls path ver; do
      [[ -z "$path" ]] && continue
      go mod edit -require="$path@$ver"
    done <<<"$entries"
    go mod tidy >/dev/null
    go run . "$command" "$version"
  )
}

main "$@"
