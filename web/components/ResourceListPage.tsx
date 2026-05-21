'use client';

import {
  Alert,
  Box,
  Button,
  CircularProgress,
  IconButton,
  Paper,
  Stack,
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
} from '@mui/icons-material';
import { ReactNode, useCallback, useEffect, useState } from 'react';
import PageHeader from './PageHeader';
import ConfirmDialog from './ConfirmDialog';
import { api } from '@/lib/api-client';
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
  renderEdit?: (props: { open: boolean; row: T; onClose: () => void; onSaved: () => void }) => ReactNode;
  canEdit?: boolean;
  canDelete?: boolean;
  query?: Record<string, string | number | boolean | undefined>;
  // re-run when these external deps change (e.g. namespace)
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
  query,
  deps = [],
  emptyMessage = 'No items',
}: Props<T>) {
  const [items, setItems] = useState<T[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [createOpen, setCreateOpen] = useState(false);
  const [editing, setEditing] = useState<T | null>(null);
  const [deleting, setDeleting] = useState<T | null>(null);

  const fetchItems = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await api.get<ListResponse<T>>(listPath, {
        query: { limit: 200, ...query },
      });
      setItems(res.items ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch');
    } finally {
      setLoading(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [listPath, JSON.stringify(query)]);

  useEffect(() => {
    void fetchItems();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [fetchItems, ...deps]);

  const onDelete = async () => {
    if (!deleting) return;
    try {
      await api.delete(itemPath(deleting));
      setDeleting(null);
      await fetchItems();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete');
    }
  };

  const hasActions = canEdit || canDelete;
  const columnCount = columns.length + (hasActions ? 1 : 0);

  return (
    <Box>
      <PageHeader
        title={title}
        subtitle={subtitle}
        actions={
          <>
            <IconButton color="primary" onClick={fetchItems}>
              <RefreshIcon />
            </IconButton>
            {renderCreate && (
              <Button startIcon={<AddIcon />} variant="contained" onClick={() => setCreateOpen(true)}>
                New
              </Button>
            )}
          </>
        }
      />

      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

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
              <TableRow><TableCell colSpan={columnCount} align="center"><CircularProgress size={24} /></TableCell></TableRow>
            ) : items.length === 0 ? (
              <TableRow><TableCell colSpan={columnCount} align="center">{emptyMessage}</TableCell></TableRow>
            ) : (
              items.map((row, i) => (
                <TableRow key={i} hover>
                  {columns.map((c) => (
                    <TableCell key={c.header}>{c.render(row)}</TableCell>
                  ))}
                  {hasActions && (
                    <TableCell align="right">
                      <Stack direction="row" gap={0.5} justifyContent="flex-end">
                        {canEdit && renderEdit && (
                          <IconButton size="small" onClick={() => setEditing(row)}>
                            <EditIcon fontSize="small" />
                          </IconButton>
                        )}
                        {canDelete && (
                          <IconButton size="small" onClick={() => setDeleting(row)}>
                            <DeleteIcon fontSize="small" />
                          </IconButton>
                        )}
                      </Stack>
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
        onSaved: () => { setCreateOpen(false); void fetchItems(); },
      })}
      {editing && renderEdit?.({
        open: editing !== null,
        row: editing,
        onClose: () => setEditing(null),
        onSaved: () => { setEditing(null); void fetchItems(); },
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
