'use client';

import { Box, Chip, Stack } from '@mui/material';
import { ResourceListPage } from '@widgets/resource-list-page';
import { JsonEditorDialog } from '@shared/ui';
import { api } from '@shared/api';
import type { Role } from '@entities/role';

function emptyRole(): Role {
  return {
    kind: 'Role',
    apiVersion: 'v1',
    metadata: {
      uid: '',
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    },
    spec: {
      displayName: '',
      description: '',
      permissions: [],
      isBuiltIn: false,
    },
  };
}

export default function RolesPage() {
  return (
    <Box>
      <ResourceListPage<Role>
        title="Roles"
        listPath="/api/v1/roles"
        itemPath={(r) => `/api/v1/roles/${r.metadata.uid}`}
        itemName={(r) => r.spec.displayName || r.metadata.uid}
        canEdit
        canDelete
        columns={[
          { header: 'Display name', render: (r) => r.spec.displayName },
          { header: 'Description', render: (r) => r.spec.description },
          {
            header: 'Permissions',
            render: (r) => (
              <Stack direction="row" gap={0.5} flexWrap="wrap">
                {(r.spec.permissions ?? []).slice(0, 6).map((p) => (
                  <Chip key={p} label={p} size="small" variant="outlined" />
                ))}
                {(r.spec.permissions ?? []).length > 6 && (
                  <Chip label={`+${r.spec.permissions!.length - 6}`} size="small" />
                )}
              </Stack>
            ),
          },
          {
            header: 'Built-in',
            render: (r) => (
              <Chip
                size="small"
                label={r.spec.isBuiltIn ? 'yes' : 'no'}
                color={r.spec.isBuiltIn ? 'info' : 'default'}
              />
            ),
          },
        ]}
        renderCreate={({ open, onClose, onSaved }) => (
          <JsonEditorDialog
            open={open}
            title="Create role"
            description="Set spec.displayName, spec.description, spec.permissions[]."
            initialValue={emptyRole()}
            samplesUrl="/samples/roles.yaml"
            onClose={onClose}
            onSave={async (parsed) => {
              await api.post('/api/v1/roles', parsed as Role);
              onSaved();
            }}
          />
        )}
        renderEdit={({ open, row, onClose, onSaved }) => (
          <JsonEditorDialog
            open={open}
            title={`Edit ${row.spec.displayName}`}
            initialValue={row}
            samplesUrl="/samples/roles.yaml"
            onClose={onClose}
            onSave={async (parsed) => {
              await api.put(`/api/v1/roles/${row.metadata.uid}`, parsed as Role);
              onSaved();
            }}
          />
        )}
      />
    </Box>
  );
}
