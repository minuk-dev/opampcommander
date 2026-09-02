'use client';

import { useNamespace } from '@entities/namespace';
import { ResourceListPage } from '@widgets/resource-list-page';
import dynamic from 'next/dynamic';
import { TimeDisplay } from '@shared/preferences';
import { api } from '@shared/api';
import { Badge } from '@shared/ui';
import type { RoleBinding } from '@entities/role-binding';

// Lazy-loaded: the JSON editor pulls in js-yaml, only needed once a
// create/edit dialog opens — keep it out of the initial route bundle.
const JsonEditorDialog = dynamic(() => import('@shared/ui/JsonEditorDialog'));

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
            <div className="flex flex-wrap gap-1">
              {(rb.spec.subjects ?? []).slice(0, 4).map((s, i) => (
                <Badge key={`${s.kind}-${s.name}-${i}`} variant="outline">
                  {s.kind}: {s.name}
                </Badge>
              ))}
              {(rb.spec.subjects ?? []).length > 4 && (
                <Badge variant="muted">+{rb.spec.subjects!.length - 4}</Badge>
              )}
            </div>
          ),
        },
        { header: 'Created', render: (rb) => <TimeDisplay value={rb.metadata.createdAt} /> },
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
  );
}
