#!/usr/bin/env python3
"""Synthesize an `otelcol components`-format document for a Linux-only distribution
(e.g. otelcol-k8s) that cannot be run on the host platform.

The distribution's build manifest lists its components only by Go module path, and the
module path does not reliably map to the registered component type name (e.g.
`memorylimiterprocessor` -> `memory_limiter`). This resolves each module to its real
type name using the `components` output of binaries that CAN be run on the host and
that contain the same modules (core + contrib), then emits a synthetic `components`
document that `opampctl generate remoteconfigschema` consumes unchanged.

Usage:
  synth_k8s_components.py <manifest.yaml> <command> <version> <components.yaml>...

Writes the synthetic document to stdout; prints a warning per unresolved module to
stderr and exits non-zero if any component could not be resolved.
"""

import re
import sys

CLASSES = ["receivers", "processors", "exporters", "extensions", "connectors"]


def module_path(value):
    """Return the module path without its trailing version (first whitespace token)."""
    return value.split()[0]


def build_dictionary(paths):
    """Map module path -> type name from one or more `components` outputs.

    Pass files oldest-first: later files overwrite, so the newest release wins on the
    (very rare) event that a module's registered type name changed across versions.
    """
    mod2type = {}
    for path in sorted(paths):
        name = None
        with open(path, encoding="utf-8") as handle:
            for line in handle:
                name_match = re.match(r"^\s+- name:\s*(\S+)", line)
                if name_match:
                    name = name_match.group(1).strip('"')
                    continue
                module_match = re.match(r"^\s+module:\s*(\S.*\S)", line)
                if module_match and name is not None:
                    mod2type[module_path(module_match.group(1))] = name
                    name = None
    return mod2type


def parse_manifest(path):
    """Map component class -> ordered list of module paths from a build manifest."""
    classes = {}
    current = None
    with open(path, encoding="utf-8") as handle:
        for line in handle:
            class_match = re.match(r"^([a-z]+):\s*$", line)
            if class_match:
                current = class_match.group(1)
                if current in CLASSES:
                    classes.setdefault(current, [])
                else:
                    current = None
                continue
            gomod_match = re.match(r"^\s+- gomod:\s*(\S.*\S)", line)
            if gomod_match and current is not None:
                classes[current].append(module_path(gomod_match.group(1)))
    return classes


def main():
    manifest, command, version = sys.argv[1], sys.argv[2], sys.argv[3]
    dictionary = build_dictionary(sys.argv[4:])
    classes = parse_manifest(manifest)

    missing = []
    lines = ["buildinfo:", f"    command: {command}", f"    version: {version}"]

    for klass in CLASSES:
        modules = classes.get(klass, [])
        entries = []
        for module in modules:
            type_name = dictionary.get(module)
            if type_name is None:
                missing.append(f"{klass}:{module}")
                continue
            entries.append(type_name)
        if not entries:
            continue
        lines.append(f"{klass}:")
        for type_name in entries:
            lines.append(f"    - name: {type_name}")

    sys.stdout.write("\n".join(lines) + "\n")

    if missing:
        for item in missing:
            print(f"warning: unresolved module {item}", file=sys.stderr)
        print(f"error: {len(missing)} unresolved component(s)", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
