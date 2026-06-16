'use client';

import { Box } from '@mui/material';
import { useNamespace } from '@entities/namespace';
import { ResourceListPage } from '@widgets/resource-list-page';
import dynamic from 'next/dynamic';
import { TimeDisplay } from '@shared/preferences';
import { api } from '@shared/api';
import type { AgentPackage } from '@entities/agent-package';

// Lazy-loaded: the JSON editor pulls in js-yaml, only needed once a
// create/edit dialog opens — keep it out of the initial route bundle.
const JsonEditorDialog = dynamic(() => import('@shared/ui/JsonEditorDialog'));

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
          { header: 'Created', render: (p) => <TimeDisplay value={p.metadata.createdAt} /> },
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
