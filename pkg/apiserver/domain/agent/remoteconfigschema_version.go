package agentmodel

import (
	"strconv"
	"strings"
)

// collectorVersionParts is the number of numeric components (major.minor.patch) in a
// collector version.
const collectorVersionParts = 3

// CompareCollectorVersion compares two MAJOR.MINOR.PATCH collector versions, returning
// -1, 0, or 1. A leading "v" is ignored, and any pre-release/build suffix and
// non-numeric part sorts as 0, which is sufficient for ordering collector releases.
func CompareCollectorVersion(left, right string) int {
	leftParts := parseCollectorVersion(left)
	rightParts := parseCollectorVersion(right)

	for i := range leftParts {
		switch {
		case leftParts[i] < rightParts[i]:
			return -1
		case leftParts[i] > rightParts[i]:
			return 1
		}
	}

	return 0
}

// parseCollectorVersion splits a version into its numeric major/minor/patch parts,
// dropping any leading "v" and any pre-release/build suffix.
func parseCollectorVersion(version string) [collectorVersionParts]int {
	version = strings.TrimPrefix(version, "v")

	if idx := strings.IndexAny(version, "-+"); idx >= 0 {
		version = version[:idx]
	}

	var parts [collectorVersionParts]int

	segments := strings.SplitN(version, ".", collectorVersionParts)
	for i := 0; i < len(segments) && i < len(parts); i++ {
		parts[i], _ = strconv.Atoi(segments[i])
	}

	return parts
}

// ResolveSchemaForVersion picks the schema describing binary at version: the exact
// version when one is present, otherwise the newest schema for that binary older than
// version. It returns nil when the candidates hold no schema for the binary, or only
// schemas newer than the requested version.
//
// Schemas are published per release, but an agent runs whatever version it runs, so a
// collector between two published schemas is validated against the newer of the two it
// is a superset of — never against a schema describing components it does not have.
func ResolveSchemaForVersion(
	schemas []*RemoteConfigSchema, binary string, version string,
) *RemoteConfigSchema {
	var best *RemoteConfigSchema

	for _, schema := range schemas {
		if schema.Spec.Binary != binary || CompareCollectorVersion(schema.Spec.Version, version) > 0 {
			continue
		}

		if best == nil || CompareCollectorVersion(schema.Spec.Version, best.Spec.Version) > 0 {
			best = schema
		}
	}

	return best
}
