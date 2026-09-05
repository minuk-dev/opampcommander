import { describe, expect, it } from 'vitest';
import {
  EMPTY_LIST_FILTERS,
  hasListFilters,
  listFilterQuery,
  type ListFilters,
} from './list-filters';

describe('listFilterQuery', () => {
  it('omits every empty filter so an untouched bar leaves the request alone', () => {
    expect(listFilterQuery(EMPTY_LIST_FILTERS)).toEqual({});
  });

  it('maps each filter onto the query parameter the API answers', () => {
    const filters: ListFilters = {
      name: 'otel-',
      nameMatch: 'contains',
      labelSelector: 'env=prod,!deprecated',
      fieldSelector: 'spec.platform=kubernetes',
    };
    expect(listFilterQuery(filters)).toEqual({
      nameContains: 'otel-',
      labelSelector: 'env=prod,!deprecated',
      fieldSelector: 'spec.platform=kubernetes',
    });
  });

  // A filter box reads as "contains" to anyone typing in it, so that is the
  // default — otherwise "tempo" would not match "otel-tempo".
  it('sends the name as a substring by default', () => {
    const query = listFilterQuery({ ...EMPTY_LIST_FILTERS, name: 'tempo' });
    expect(query).toEqual({ nameContains: 'tempo' });
    expect(query.name).toBeUndefined();
  });

  // A substring is an unindexed scan the page limit does not bound, so listings
  // that grow with the fleet ask for the indexed prefix instead.
  it('sends the indexed prefix parameter when the listing asks for prefix match', () => {
    const query = listFilterQuery({ ...EMPTY_LIST_FILTERS, name: 'ip-10-', nameMatch: 'prefix' });
    expect(query).toEqual({ name: 'ip-10-' });
    expect(query.nameContains).toBeUndefined();
  });

  it('sends the selector verbatim, so the server sees what the user typed', () => {
    const query = listFilterQuery({
      ...EMPTY_LIST_FILTERS,
      labelSelector: 'tier notin (canary,dev)',
    });
    expect(query.labelSelector).toBe('tier notin (canary,dev)');
  });
});

describe('hasListFilters', () => {
  it('is false when nothing is filtered', () => {
    expect(hasListFilters(EMPTY_LIST_FILTERS)).toBe(false);
  });

  it('is true when any single filter is set', () => {
    expect(hasListFilters({ ...EMPTY_LIST_FILTERS, name: 'a' })).toBe(true);
    expect(hasListFilters({ ...EMPTY_LIST_FILTERS, labelSelector: 'env=prod' })).toBe(true);
    expect(hasListFilters({ ...EMPTY_LIST_FILTERS, fieldSelector: 'spec.platform=vm' })).toBe(true);
  });
});
