'use client';

import { Eye, Pencil, Plus, RefreshCw, Trash2 } from 'lucide-react';
import { type ReactNode, useState } from 'react';
import { api } from '@shared/api';
import {
  cn,
  EMPTY_LIST_FILTERS,
  hasListFilters,
  listFilterQuery,
  useCursorPagination,
  type ListFilters,
} from '@shared/lib';
import {
  Alert,
  Button,
  ConfirmDialog,
  ListFilterBar,
  PageHeader,
  PaginationFooter,
  RowActionsMenu,
  type RowAction,
  Spinner,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeaderCell,
  TableRow,
  TableWrap,
} from '@shared/ui';

export interface Column<T> {
  header: string;
  render: (row: T) => ReactNode;
  width?: number | string;
}

interface Props<T> {
  title: string;
  subtitle?: string;
  listPath: string;
  itemPath: (row: T) => string;
  itemName: (row: T) => string;
  columns: Column<T>[];
  renderCreate?: (props: { open: boolean; onClose: () => void; onSaved: () => void }) => ReactNode;
  renderEdit?: (props: {
    open: boolean;
    row: T;
    onClose: () => void;
    onSaved: () => void;
  }) => ReactNode;
  canEdit?: boolean;
  canDelete?: boolean;
  // When set, the row action menu includes a "View detail" entry that
  // navigates to detailHref(row).
  detailHref?: (row: T) => string;
  // Extra actions added to the row menu (e.g. domain-specific operations).
  // `refresh` re-fetches the list, for actions that mutate the row themselves.
  extraActions?: (row: T, ctx: { refresh: () => void }) => RowAction[];
  query?: Record<string, string | number | boolean | undefined>;
  // Shows the server-side filter bar (name prefix + label selector). Opt-in
  // because it must only appear where the list endpoint answers the filters —
  // a bar whose input is ignored would be worse than none at all.
  filterable?: boolean;
  // Overrides the filter bar's name field for resources whose name is not
  // called "name" (a user's email, for instance).
  nameLabel?: string;
  namePlaceholder?: string;
  // Deprecated: SWR keys off listPath + query, so re-fetching when the
  // namespace changes is automatic. Kept so existing callers still type-check.
  deps?: ReadonlyArray<unknown>;
  emptyMessage?: string;
}

export default function ResourceListPage<T>({
  title,
  subtitle,
  listPath,
  itemPath,
  itemName,
  columns,
  renderCreate,
  renderEdit,
  canEdit,
  canDelete,
  detailHref,
  extraActions,
  query,
  filterable,
  nameLabel,
  namePlaceholder,
  emptyMessage = 'No items',
}: Props<T>) {
  const [createOpen, setCreateOpen] = useState(false);
  const [filters, setFilters] = useState<ListFilters>(EMPTY_LIST_FILTERS);
  const [editing, setEditing] = useState<T | null>(null);
  const [deleting, setDeleting] = useState<T | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

  // The filters go into the request, not into a pass over the fetched page, so
  // the paginated total keeps describing the set the rows were drawn from.
  const pagination = useCursorPagination<T>(listPath, {
    query: { ...query, ...listFilterQuery(filters) },
  });
  const { items, isLoading, isValidating, error: fetchError, refresh } = pagination;
  const error =
    actionError ??
    (fetchError instanceof Error ? fetchError.message : fetchError ? 'Failed to fetch' : null);

  const onDelete = async () => {
    if (!deleting) return;
    try {
      await api.delete(itemPath(deleting));
      setDeleting(null);
      setActionError(null);
      refresh();
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Failed to delete');
    }
  };

  const hasActions = Boolean(canEdit || canDelete || detailHref || extraActions);
  const columnCount = columns.length + (hasActions ? 1 : 0);

  const buildActions = (row: T): RowAction[] => {
    const out: RowAction[] = [];
    if (detailHref) {
      out.push({ label: 'View detail', icon: <Eye aria-hidden />, href: detailHref(row) });
    }
    if (canEdit && renderEdit) {
      out.push({ label: 'Edit', icon: <Pencil aria-hidden />, onClick: () => setEditing(row) });
    }
    if (extraActions) out.push(...extraActions(row, { refresh }));
    if (canDelete) {
      out.push({
        label: 'Delete',
        icon: <Trash2 aria-hidden />,
        destructive: true,
        divider: out.length > 0,
        onClick: () => setDeleting(row),
      });
    }
    return out;
  };

  return (
    <div>
      <PageHeader
        title={title}
        subtitle={subtitle}
        actions={
          <>
            <Button
              variant="ghost"
              size="icon-sm"
              aria-label="Refresh"
              onClick={() => refresh()}
              disabled={isValidating}
            >
              <RefreshCw className={cn(isValidating && 'animate-spin')} aria-hidden />
            </Button>
            {renderCreate && (
              <Button size="sm" onClick={() => setCreateOpen(true)}>
                <Plus aria-hidden />
                New
              </Button>
            )}
          </>
        }
      />

      {filterable && (
        <ListFilterBar
          value={filters}
          onChange={setFilters}
          nameLabel={nameLabel}
          namePlaceholder={namePlaceholder}
        />
      )}

      {error && (
        <Alert severity="error" className="mb-3">
          {error}
        </Alert>
      )}

      <TableWrap>
        <Table>
          <TableHead>
            <TableRow className="hover:bg-transparent">
              {columns.map((c) => (
                <TableHeaderCell key={c.header} style={c.width ? { width: c.width } : undefined}>
                  {c.header}
                </TableHeaderCell>
              ))}
              {hasActions && <TableHeaderCell className="w-10 text-right">Actions</TableHeaderCell>}
            </TableRow>
          </TableHead>
          <TableBody>
            {isLoading ? (
              <TableRow className="hover:bg-transparent">
                <TableCell colSpan={columnCount} className="py-8">
                  <Spinner className="mx-auto size-5" />
                </TableCell>
              </TableRow>
            ) : items.length === 0 ? (
              <TableRow className="hover:bg-transparent">
                <TableCell
                  colSpan={columnCount}
                  className="py-8 text-center text-sm text-muted-foreground"
                >
                  {hasListFilters(filters) ? 'No items match the filters' : emptyMessage}
                </TableCell>
              </TableRow>
            ) : (
              items.map((row) => (
                <TableRow key={itemName(row)}>
                  {columns.map((c) => (
                    <TableCell key={c.header}>{c.render(row)}</TableCell>
                  ))}
                  {hasActions && (
                    <TableCell className="text-right">
                      <RowActionsMenu actions={buildActions(row)} />
                    </TableCell>
                  )}
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableWrap>

      <PaginationFooter pagination={pagination} />

      {createOpen &&
        renderCreate?.({
          open: createOpen,
          onClose: () => setCreateOpen(false),
          onSaved: () => {
            setCreateOpen(false);
            refresh();
          },
        })}
      {editing &&
        renderEdit?.({
          open: editing !== null,
          row: editing,
          onClose: () => setEditing(null),
          onSaved: () => {
            setEditing(null);
            refresh();
          },
        })}
      <ConfirmDialog
        open={deleting !== null}
        title={`Delete ${title.toLowerCase()}`}
        message={`Delete "${deleting ? itemName(deleting) : ''}"?`}
        confirmLabel="Delete"
        destructive
        onClose={() => setDeleting(null)}
        onConfirm={onDelete}
      />
    </div>
  );
}
