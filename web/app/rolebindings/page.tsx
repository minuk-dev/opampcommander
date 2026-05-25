'use client';

import { Box, Chip, Stack } from '@mui/material';
import { useNamespace } from '@/components/NamespaceProvider';
import ResourceListPage from '@/components/ResourceListPage';
import JsonEditorDialog from '@/components/JsonEditorDialog';
import { api } from '@/lib/api-client';
import type { RoleBinding } from '@/lib/types';

function emptyRoleBinding(namespace: string): RoleBinding {
  return {
    kind: 'RoleBinding',
    apiVersion: 'v1',
    metadata: { namespace, name: '' },
    spec: {
      roleRef: { kind: 'Role', name: '' },
      subjects: [],
    },
  };
}

export default function RoleBindingsPage() {
  const { namespace } = useNamespace();
  return (
    <Box>
      <ResourceListPage<RoleBinding>
        title="Role Bindings"
        subtitle={`Namespace: ${namespace}`}
        listPath={`/api/v1/namespaces/${namespace}/rolebindings`}
        itemPath={(rb) => `/api/v1/namespaces/${namespace}/rolebindings/${rb.metadata.name}`}
        itemName={(rb) => rb.metadata.name}
        deps={[namespace]}
        canEdit
        canDelete
        columns={[
          { header: 'Name', render: (rb) => rb.metadata.name },
          {
            header: 'Role',
            render: (rb) => `${rb.spec.roleRef.kind}/${rb.spec.roleRef.name}`,
          },
          {
            header: 'Subjects',
            render: (rb) => (
              <Stack direction="row" gap={0.5} flexWrap="wrap">
                {(rb.spec.subjects ?? []).slice(0, 4).map((s, i) => (
                  <Chip
                    key={`${s.kind}-${s.name}-${i}`}
                    label={`${s.kind}: ${s.name}`}
                    size="small"
                    variant="outlined"
                  />
                ))}
                {(rb.spec.subjects ?? []).length > 4 && (
                  <Chip label={`+${rb.spec.subjects!.length - 4}`} size="small" />
                )}
              </Stack>
            ),
          },
          { header: 'Created', render: (rb) => rb.metadata.createdAt || '-' },
        ]}
        renderCreate={({ open, onClose, onSaved }) => (
          <JsonEditorDialog
            open={open}
            title="Create role binding"
            description="metadata.name + spec.roleRef + spec.subjects[]."
            initialValue={emptyRoleBinding(namespace)}
            samplesUrl="/samples/rolebindings.yaml"
            samplesVars={{ namespace }}
            onClose={onClose}
            onSave={async (parsed) => {
              await api.post(`/api/v1/namespaces/${namespace}/rolebindings`, parsed as RoleBinding);
              onSaved();
            }}
          />
        )}
        renderEdit={({ open, row, onClose, onSaved }) => (
          <JsonEditorDialog
            open={open}
            title={`Edit ${row.metadata.name}`}
            initialValue={row}
            samplesUrl="/samples/rolebindings.yaml"
            samplesVars={{ namespace }}
            onClose={onClose}
            onSave={async (parsed) => {
              await api.put(
                `/api/v1/namespaces/${namespace}/rolebindings/${row.metadata.name}`,
                parsed as RoleBinding,
              );
              onSaved();
            }}
          />
        )}
      />
    </Box>
  );
}
