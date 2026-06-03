'use client';

import {
  Alert,
  Box,
  Button,
  CircularProgress,
  IconButton,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
} from '@mui/material';
import {
  Add as AddIcon,
  Delete as DeleteIcon,
  Edit as EditIcon,
  Refresh as RefreshIcon,
  Visibility as ViewIcon,
} from '@mui/icons-material';
import { type ReactNode, useState } from 'react';
import PageHeader from './PageHeader';
import ConfirmDialog from './ConfirmDialog';
import RowActionsMenu, { type RowAction } from './RowActionsMenu';
import { api } from '@/lib/api-client';
import { useApi } from '@/lib/swr';
import type { ListResponse } from '@/lib/types';

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
  extraActions?: (row: T) => RowAction[];
  query?: Record<string, string | number | boolean | undefined>;
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
  emptyMessage = 'No items',
}: Props<T>) {
  const [createOpen, setCreateOpen] = useState(false);
  const [editing, setEditing] = useState<T | null>(null);
  const [deleting, setDeleting] = useState<T | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

  const {
    data,
    error: fetchError,
    isLoading,
    isValidating,
    mutate,
  } = useApi<ListResponse<T>>([listPath, { limit: 200, ...query }]);

  const items = data?.items ?? [];
  const loading = isLoading;
  const error =
    actionError ??
    (fetchError instanceof Error ? fetchError.message : fetchError ? 'Failed to fetch' : null);

  const onDelete = async () => {
    if (!deleting) return;
    try {
      await api.delete(itemPath(deleting));
      setDeleting(null);
      setActionError(null);
      await mutate();
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Failed to delete');
    }
  };

  const hasActions = Boolean(canEdit || canDelete || detailHref || extraActions);
  const columnCount = columns.length + (hasActions ? 1 : 0);

  const buildActions = (row: T): RowAction[] => {
    const out: RowAction[] = [];
    if (detailHref) {
      out.push({
        label: 'View detail',
        icon: <ViewIcon fontSize="small" />,
        href: detailHref(row),
      });
    }
    if (canEdit && renderEdit) {
      out.push({
        label: 'Edit',
        icon: <EditIcon fontSize="small" />,
        onClick: () => setEditing(row),
      });
    }
    if (extraActions) {
      out.push(...extraActions(row));
    }
    if (canDelete) {
      out.push({
        label: 'Delete',
        icon: <DeleteIcon fontSize="small" />,
        destructive: true,
        divider: out.length > 0,
        onClick: () => setDeleting(row),
      });
    }
    return out;
  };

  return (
    <Box>
      <PageHeader
        title={title}
        subtitle={subtitle}
        actions={
          <>
            <IconButton color="primary" onClick={() => mutate()} disabled={isValidating}>
              <RefreshIcon />
            </IconButton>
            {renderCreate && (
              <Button
                startIcon={<AddIcon />}
                variant="contained"
                onClick={() => setCreateOpen(true)}
              >
                New
              </Button>
            )}
          </>
        }
      />

      {error && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {error}
        </Alert>
      )}

      <TableContainer component={Paper}>
        <Table>
          <TableHead>
            <TableRow>
              {columns.map((c) => (
                <TableCell key={c.header} sx={c.width ? { width: c.width } : undefined}>
                  {c.header}
                </TableCell>
              ))}
              {hasActions && <TableCell align="right">Actions</TableCell>}
            </TableRow>
          </TableHead>
          <TableBody>
            {loading ? (
              <TableRow>
                <TableCell colSpan={columnCount} align="center">
                  <CircularProgress size={24} />
                </TableCell>
              </TableRow>
            ) : items.length === 0 ? (
              <TableRow>
                <TableCell colSpan={columnCount} align="center">
                  {emptyMessage}
                </TableCell>
              </TableRow>
            ) : (
              items.map((row, i) => (
                <TableRow key={i} hover>
                  {columns.map((c) => (
                    <TableCell key={c.header}>{c.render(row)}</TableCell>
                  ))}
                  {hasActions && (
                    <TableCell align="right">
                      <RowActionsMenu actions={buildActions(row)} />
                    </TableCell>
                  )}
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableContainer>

      {renderCreate?.({
        open: createOpen,
        onClose: () => setCreateOpen(false),
        onSaved: () => {
          setCreateOpen(false);
          void mutate();
        },
      })}
      {editing &&
        renderEdit?.({
          open: editing !== null,
          row: editing,
          onClose: () => setEditing(null),
          onSaved: () => {
            setEditing(null);
            void mutate();
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
    </Box>
  );
}
