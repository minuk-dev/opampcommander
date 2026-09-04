'use client';

import { Plus, RefreshCw, Trash2 } from 'lucide-react';
import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Button,
  ConfirmDialog,
  Dialog,
  DialogBody,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Field,
  Input,
  ListFilterBar,
  PageHeader,
  Spinner,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeaderCell,
  TableRow,
  TableWrap,
  Textarea,
} from '@shared/ui';
import {
  cn,
  EMPTY_LIST_FILTERS,
  hasListFilters,
  listFilterQuery,
  type ListFilters,
} from '@shared/lib';
import { TimeDisplay } from '@shared/preferences';
import { useNamespace, type Namespace } from '@entities/namespace';
import { api, type ListResponse } from '@shared/api';

// The page size the listing requests. Filters are answered by the server, so
// narrowing the view shrinks the query rather than growing the fetch.
const PAGE_LIMIT = 200;

export default function NamespacesPage() {
  const { namespaces: ctxNamespaces, refresh: refreshCtx } = useNamespace();
  const [items, setItems] = useState<Namespace[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [createOpen, setCreateOpen] = useState(false);
  const [newName, setNewName] = useState('');
  const [labelsText, setLabelsText] = useState('{}');
  const [deleting, setDeleting] = useState<Namespace | null>(null);
  const [filters, setFilters] = useState<ListFilters>(EMPTY_LIST_FILTERS);

  const filtered = hasListFilters(filters);
  // Memoised on the filter state so fetchItems is stable between renders.
  const filterQuery = useMemo(() => listFilterQuery(filters), [filters]);

  const fetchItems = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await api.get<ListResponse<Namespace>>('/api/v1/namespaces', {
        query: { limit: PAGE_LIMIT, ...filterQuery },
      });
      setItems(res.items ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch namespaces');
    } finally {
      setLoading(false);
    }
  }, [filterQuery]);

  useEffect(() => {
    // The namespace switcher already holds the unfiltered list, so reuse it
    // rather than fetching it twice. A filter has to go to the server, so it
    // always fetches.
    if (!filtered && ctxNamespaces.length > 0) {
      setItems(ctxNamespaces);
      setLoading(false);
    } else {
      void fetchItems();
    }
  }, [ctxNamespaces, fetchItems, filtered]);

  const onCreate = async () => {
    setError(null);
    try {
      const labels = labelsText.trim() ? JSON.parse(labelsText) : undefined;
      await api.post('/api/v1/namespaces', {
        metadata: { name: newName, labels },
      });
      setCreateOpen(false);
      setNewName('');
      setLabelsText('{}');
      await fetchItems();
      await refreshCtx();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create');
    }
  };

  const onDelete = async () => {
    if (!deleting) return;
    try {
      await api.delete(`/api/v1/namespaces/${deleting.metadata.name}`);
      setDeleting(null);
      await fetchItems();
      await refreshCtx();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete');
    }
  };

  return (
    <div>
      <PageHeader
        title="Namespaces"
        actions={
          <>
            <Button
              variant="ghost"
              size="icon-sm"
              aria-label="Refresh"
              onClick={() => void fetchItems()}
            >
              <RefreshCw className={cn(loading && 'animate-spin')} aria-hidden />
            </Button>
            <Button size="sm" onClick={() => setCreateOpen(true)}>
              <Plus aria-hidden />
              New namespace
            </Button>
          </>
        }
      />

      <ListFilterBar value={filters} onChange={setFilters} />

      {error && (
        <Alert severity="error" className="mb-3">
          {error}
        </Alert>
      )}

      <TableWrap>
        <Table>
          <TableHead>
            <TableRow className="hover:bg-transparent">
              <TableHeaderCell>Name</TableHeaderCell>
              <TableHeaderCell>Labels</TableHeaderCell>
              <TableHeaderCell>Created</TableHeaderCell>
              <TableHeaderCell>Deleted</TableHeaderCell>
              <TableHeaderCell className="w-10 text-right">Actions</TableHeaderCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {loading ? (
              <TableRow className="hover:bg-transparent">
                <TableCell colSpan={5} className="py-8">
                  <Spinner className="mx-auto size-5" />
                </TableCell>
              </TableRow>
            ) : items.length === 0 ? (
              <TableRow className="hover:bg-transparent">
                <TableCell colSpan={5} className="py-8 text-center text-muted-foreground">
                  {filtered ? 'No namespaces match the filters' : 'No namespaces'}
                </TableCell>
              </TableRow>
            ) : (
              items.map((ns) => (
                <TableRow key={ns.metadata.name}>
                  <TableCell className="font-medium">{ns.metadata.name}</TableCell>
                  <TableCell className="font-mono text-xs">
                    {ns.metadata.labels ? JSON.stringify(ns.metadata.labels) : '-'}
                  </TableCell>
                  <TableCell>
                    <TimeDisplay value={ns.metadata.createdAt} />
                  </TableCell>
                  <TableCell>
                    <TimeDisplay value={ns.metadata.deletedAt} />
                  </TableCell>
                  <TableCell className="text-right">
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      aria-label={`Delete ${ns.metadata.name}`}
                      onClick={() => setDeleting(ns)}
                    >
                      <Trash2 aria-hidden />
                    </Button>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableWrap>

      <Dialog open={createOpen} onOpenChange={(next) => !next && setCreateOpen(false)}>
        <DialogContent size="sm">
          <DialogHeader>
            <DialogTitle>Create namespace</DialogTitle>
          </DialogHeader>
          <DialogBody className="space-y-3">
            <Field label="Name" required>
              {(field) => (
                <Input
                  {...field}
                  autoFocus
                  value={newName}
                  onChange={(e) => setNewName(e.target.value)}
                />
              )}
            </Field>
            <Field label="Labels (JSON)">
              {(field) => (
                <Textarea
                  {...field}
                  mono
                  rows={3}
                  value={labelsText}
                  onChange={(e) => setLabelsText(e.target.value)}
                />
              )}
            </Field>
          </DialogBody>
          <DialogFooter>
            <Button variant="ghost" onClick={() => setCreateOpen(false)}>
              Cancel
            </Button>
            <Button onClick={() => void onCreate()} disabled={!newName}>
              Create
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
      <ConfirmDialog
        open={deleting !== null}
        title="Delete namespace"
        message={`Delete "${deleting?.metadata.name}"? This will cascade to its resources.`}
        confirmLabel="Delete"
        destructive
        onClose={() => setDeleting(null)}
        onConfirm={onDelete}
      />
    </div>
  );
}
