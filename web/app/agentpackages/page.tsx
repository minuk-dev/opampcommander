'use client';

import { Box } from '@mui/material';
import { useNamespace } from '@/components/NamespaceProvider';
import ResourceListPage from '@/components/ResourceListPage';
import JsonEditorDialog from '@/components/JsonEditorDialog';
import { api } from '@/lib/api-client';
import type { AgentPackage } from '@/lib/types';

function emptyPackage(namespace: string, name: string): AgentPackage {
  return {
    metadata: {
      name,
      namespace,
      attributes: {},
      createdAt: new Date().toISOString(),
    },
    spec: {
      packageType: '',
      version: '',
      downloadUrl: '',
    },
  };
}

export default function AgentPackagesPage() {
  const { namespace } = useNamespace();
  return (
    <Box>
      <ResourceListPage<AgentPackage>
        title="Agent Packages"
        subtitle={`Namespace: ${namespace}`}
        listPath={`/api/v1/namespaces/${namespace}/agentpackages`}
        itemPath={(p) => `/api/v1/namespaces/${namespace}/agentpackages/${p.metadata.name}`}
        itemName={(p) => p.metadata.name}
        deps={[namespace]}
        canEdit
        canDelete
        columns={[
          { header: 'Name', render: (p) => p.metadata.name },
          { header: 'Type', render: (p) => p.spec.packageType || '-' },
          { header: 'Version', render: (p) => p.spec.version || '-' },
          {
            header: 'Download URL',
            render: (p) => (
              <span style={{ fontFamily: 'monospace', fontSize: 12 }}>
                {p.spec.downloadUrl || '-'}
              </span>
            ),
          },
          { header: 'Created', render: (p) => p.metadata.createdAt },
        ]}
        renderCreate={({ open, onClose, onSaved }) => (
          <JsonEditorDialog
            open={open}
            title="Create agent package"
            description={
              <>
                Define metadata (<code>name</code>, <code>attributes</code>) and spec (
                <code>packageType</code>, <code>version</code>, <code>downloadUrl</code>, optional{' '}
                <code>contentHash</code>, <code>signature</code>, <code>headers</code>,{' '}
                <code>hash</code>).
              </>
            }
            initialValue={emptyPackage(namespace, '')}
            samplesUrl="/samples/agentpackages.yaml"
            samplesVars={{ namespace }}
            onClose={onClose}
            onSave={async (parsed) => {
              const body = parsed as AgentPackage;
              await api.post(`/api/v1/namespaces/${namespace}/agentpackages`, body);
              onSaved();
            }}
          />
        )}
        renderEdit={({ open, row, onClose, onSaved }) => (
          <JsonEditorDialog
            open={open}
            title={`Edit ${row.metadata.name}`}
            description="Edit the package as JSON."
            initialValue={row}
            samplesUrl="/samples/agentpackages.yaml"
            samplesVars={{ namespace }}
            onClose={onClose}
            onSave={async (parsed) => {
              const body = parsed as AgentPackage;
              await api.put(
                `/api/v1/namespaces/${namespace}/agentpackages/${row.metadata.name}`,
                body,
              );
              onSaved();
            }}
          />
        )}
      />
    </Box>
  );
}
