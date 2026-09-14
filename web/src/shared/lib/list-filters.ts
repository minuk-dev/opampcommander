// Server-side list filtering, shared by every list page.
//
// The API answers a metadata selector, a `fieldSelector` over an allowlisted set
// of a resource's own fields, and a name search, on every list endpoint.
// Filtering has to reach the datastore rather than happen after a page is cut, or
// the paginated total describes a set the rows on screen are not drawn from.
//
// The metadata selector likewise comes in two parameters, because they read
// different metadata: `labelSelector` reads what an operator set,
// `attributeSelector` reads
// what the resource reported about itself. Only agents have the latter, and the
// agents page has its own search UI, so the shared filter bar here always sends
// `labelSelector`. Sending the wrong one is a 400, not an ignored parameter.

// NameMatch is how the name field is matched, and it is a cost decision as much
// as a UX one.
//
// `contains` sends `nameContains`: what a filter box reads as to anyone typing in
// it, and the only way "tempo" finds "otel-tempo". No index can answer it, and the
// page limit does not bound it — an exact remaining count has to see the whole
// match — so the server scans the collection on every request.
//
// `prefix` sends `name`: an index range scan. It is the right default wherever the
// collection grows with the fleet rather than with what an operator wrote.
export type NameMatch = 'contains' | 'prefix';

export interface ListFilters {
  // The name to match, per nameMatch.
  name: string;
  // How `name` is matched. It lives in the filter state rather than alongside it
  // so the query and the placeholder cannot disagree about which parameter is
  // being sent.
  nameMatch: NameMatch;
  // Kubernetes-style expression over the resource's labels, e.g.
  // "env=prod,tier notin (canary,dev),!deprecated". Sent verbatim; the server
  // parses it and rejects a malformed one with a 400 naming the parameter.
  labelSelector: string;
  // Expression over the resource's own fields, e.g. "status.connected=true".
  fieldSelector: string;
}

export const EMPTY_LIST_FILTERS: ListFilters = {
  name: '',
  nameMatch: 'contains',
  labelSelector: '',
  fieldSelector: '',
};

// listFilterQuery renders the applied filters as query parameters, omitting the
// empty ones so an untouched filter bar leaves the request (and therefore the
// SWR cache key) exactly as it was.
export function listFilterQuery(filters: ListFilters): Record<string, string> {
  const query: Record<string, string> = {};
  if (filters.name) {
    if (filters.nameMatch === 'prefix') {
      query.name = filters.name;
    } else {
      query.nameContains = filters.name;
    }
  }
  if (filters.labelSelector) query.labelSelector = filters.labelSelector;
  if (filters.fieldSelector) query.fieldSelector = filters.fieldSelector;
  return query;
}

// hasListFilters reports whether anything is being filtered on, for empty-state
// copy that distinguishes "nothing here" from "nothing matches".
export function hasListFilters(filters: ListFilters): boolean {
  return Object.keys(listFilterQuery(filters)).length > 0;
}
