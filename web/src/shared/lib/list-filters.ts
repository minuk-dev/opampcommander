// Server-side list filtering, shared by every list page.
//
// The API answers a metadata selector, a `fieldSelector` over an allowlisted set
// of a resource's own fields, and a name search, on every list endpoint.
// Filtering has to reach the datastore rather than happen after a page is cut, or
// the paginated total describes a set the rows on screen are not drawn from.
//
// The metadata selector comes in two parameters, because they read different
// metadata: `labelSelector` reads what an operator set, `attributeSelector` reads
// what the resource reported about itself. Only agents have the latter, and the
// agents page has its own search UI, so the shared filter bar here always sends
// `labelSelector`. Sending the wrong one is a 400, not an ignored parameter.

export interface ListFilters {
  // Substring of the resource's name, matched case-insensitively. It is sent as
  // `nameContains` rather than the `name` prefix parameter: a filter box reads as
  // "contains" to anyone typing in it, and every listing that shows this bar is
  // config-scale, where the scan that costs is not worth the surprise of "tempo"
  // not matching "otel-tempo".
  name: string;
  // Kubernetes-style expression over the resource's labels, e.g.
  // "env=prod,tier notin (canary,dev),!deprecated". Sent verbatim; the server
  // parses it and rejects a malformed one with a 400 naming the parameter.
  labelSelector: string;
  // Expression over the resource's own fields, e.g. "status.connected=true".
  fieldSelector: string;
}

export const EMPTY_LIST_FILTERS: ListFilters = {
  name: '',
  labelSelector: '',
  fieldSelector: '',
};

// listFilterQuery renders the applied filters as query parameters, omitting the
// empty ones so an untouched filter bar leaves the request (and therefore the
// SWR cache key) exactly as it was.
export function listFilterQuery(filters: ListFilters): Record<string, string> {
  const query: Record<string, string> = {};
  if (filters.name) query.nameContains = filters.name;
  if (filters.labelSelector) query.labelSelector = filters.labelSelector;
  if (filters.fieldSelector) query.fieldSelector = filters.fieldSelector;
  return query;
}

// hasListFilters reports whether anything is being filtered on, for empty-state
// copy that distinguishes "nothing here" from "nothing matches".
export function hasListFilters(filters: ListFilters): boolean {
  return Object.keys(listFilterQuery(filters)).length > 0;
}
