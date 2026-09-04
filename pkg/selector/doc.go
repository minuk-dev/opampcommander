// Package selector implements Kubernetes-style label and field selectors.
//
// A label selector filters resources on their user-supplied metadata map — the
// aggregate's metadata.labels, or metadata.attributes where the aggregate calls
// it that. It supports equality (=, ==, !=), set membership (in, notin) and
// existence (key, !key), combined with AND:
//
//	env=prod,tier notin (canary,dev),!deprecated
//
// A field selector filters on a documented, indexed allowlist of the resource's
// own fields, with equality only (=, ==, !=), also combined with AND:
//
//	status.connected=true,metadata.namespace=prod
//
// Selectors are parsed once at the API boundary and pushed all the way into the
// datastore query, so a filtered page and its RemainingItemCount agree.
package selector
