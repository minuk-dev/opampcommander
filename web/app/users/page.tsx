'use client';

import { Box, Chip } from '@mui/material';
import ResourceListPage from '@/components/ResourceListPage';
import JsonEditorDialog from '@/components/JsonEditorDialog';
import TimeDisplay from '@/components/TimeDisplay';
import { api } from '@/lib/api-client';
import type { User } from '@/lib/types';

function emptyUser(): User {
  return {
    kind: 'User',
    apiVersion: 'v1',
    metadata: {
      uid: '',
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    },
    spec: { email: '', username: '', isActive: true },
  };
}

export default function UsersPage() {
  return (
    <Box>
      <ResourceListPage<User>
        title="Users"
        listPath="/api/v1/users"
        itemPath={(u) => `/api/v1/users/${u.metadata.uid}`}
        itemName={(u) => u.spec.email || u.spec.username || u.metadata.uid}
        canDelete
        columns={[
          { header: 'Email', render: (u) => u.spec.email || '-' },
          { header: 'Username', render: (u) => u.spec.username || '-' },
          {
            header: 'Status',
            render: (u) => (
              <Chip
                size="small"
                label={u.spec.isActive ? 'active' : 'inactive'}
                color={u.spec.isActive ? 'success' : 'default'}
              />
            ),
          },
          {
            header: 'UID',
            render: (u) => (
              <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{u.metadata.uid}</span>
            ),
          },
          { header: 'Created', render: (u) => <TimeDisplay value={u.metadata.createdAt} /> },
        ]}
        renderCreate={({ open, onClose, onSaved }) => (
          <JsonEditorDialog
            open={open}
            title="Create user"
            description="Set spec.email, spec.username, spec.isActive."
            initialValue={emptyUser()}
            samplesUrl="/samples/users.yaml"
            onClose={onClose}
            onSave={async (parsed) => {
              await api.post('/api/v1/users', parsed as User);
              onSaved();
            }}
          />
        )}
      />
    </Box>
  );
}
