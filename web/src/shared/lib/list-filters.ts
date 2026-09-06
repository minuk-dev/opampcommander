// Server-side list filtering, shared by every list page.
//
// The API answers three filtering query parameters on every list endpoint:
// `labelSelector` over the resource's user-supplied metadata map,
// `fieldSelector` over an allowlisted set of its own fields, and `name` as a
// case-sensitive name prefix. Filtering has to reach the datastore rather than
// happen after a page is cut, or the paginated total describes a set the rows on
// screen are not drawn from.

export interface ListFilters {
  // Case-sensitive prefix of the resource's name.
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
  if (filters.name) query.name = filters.name;
  if (filters.labelSelector) query.labelSelector = filters.labelSelector;
  if (filters.fieldSelector) query.fieldSelector = filters.fieldSelector;
  return query;
}

// hasListFilters reports whether anything is being filtered on, for empty-state
// copy that distinguishes "nothing here" from "nothing matches".
export function hasListFilters(filters: ListFilters): boolean {
  return Object.keys(listFilterQuery(filters)).length > 0;
}
