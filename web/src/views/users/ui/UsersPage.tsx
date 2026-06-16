'use client';

import { Box, Chip } from '@mui/material';
import { ResourceListPage } from '@widgets/resource-list-page';
import dynamic from 'next/dynamic';
import { TimeDisplay } from '@shared/preferences';
import type { User } from '@entities/user';

// Lazy-loaded: the dialog and its form fields are only needed once the user
// opens the create flow — keep them out of the initial route bundle.
const UserCreateDialog = dynamic(() =>
  import('@features/user-create').then((m) => m.UserCreateDialog),
);

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
          <UserCreateDialog open={open} onClose={onClose} onSaved={onSaved} />
        )}
      />
    </Box>
  );
}
