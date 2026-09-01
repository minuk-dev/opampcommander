'use client';

import { Box } from '@mui/material';
import { Code as CodeIcon } from '@mui/icons-material';
import { useState } from 'react';
import dynamic from 'next/dynamic';
import { useNamespace } from '@entities/namespace';
import { ResourceListPage } from '@widgets/resource-list-page';
import { TimeDisplay } from '@shared/preferences';
import { api } from '@shared/api';
import type { AgentPackage } from '@entities/agent-package';

// Lazy-loaded: neither dialog is needed to render the list, so they stay out
// of the initial route bundle.
const AgentPackageEditDialog = dynamic(
  () => import('@features/agent-package-edit/ui/AgentPackageEditDialog'),
);
const JsonEditorDialog = dynamic(() => import('@shared/ui/JsonEditorDialog'));

// Raw editing stays available for fields the form does not model (and for
// anyone who prefers pasting a manifest); it is one menu entry away.
interface RawTarget {
  row: AgentPackage;
  refresh: () => void;
}

export default function AgentPackagesPage() {
  const { namespace } = useNamespace();
  const [rawTarget, setRawTarget] = useState<RawTarget | null>(null);

  return (
    <Box>
      <ResourceListPage<AgentPackage>
        title="Agent Packages"
        subtitle={`Namespace: ${namespace}`}
        listPath={`/api/v1/namespaces/${namespace}/agentpackages`}
        itemPath={(p) => `/api/v1/namespaces/${namespace}/agentpackages/${p.metadata.name}`}
        itemName={(p) => p.metadata.name}
        canEdit
        canDelete
        extraActions={(row, { refresh }) => [
          {
            label: 'Edit as YAML',
            icon: <CodeIcon fontSize="small" />,
            onClick: () => setRawTarget({ row, refresh }),
          },
        ]}
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
          <AgentPackageEditDialog
            open={open}
            mode="create"
            namespace={namespace}
            onClose={onClose}
            onSaved={onSaved}
          />
        )}
        renderEdit={({ open, row, onClose, onSaved }) => (
          <AgentPackageEditDialog
            open={open}
            mode="edit"
            namespace={namespace}
            initial={row}
            onClose={onClose}
            onSaved={onSaved}
          />
        )}
      />
      {rawTarget !== null && (
        <JsonEditorDialog
          open
          title={`Edit ${rawTarget.row.metadata.name} (raw)`}
          description="Edit the whole package manifest."
          initialValue={rawTarget.row}
          samplesUrl="/samples/agentpackages.yaml"
          samplesVars={{ namespace }}
          onClose={() => setRawTarget(null)}
          onSave={async (parsed) => {
            await api.put(
              `/api/v1/namespaces/${namespace}/agentpackages/${rawTarget.row.metadata.name}`,
              parsed as AgentPackage,
            );
            rawTarget.refresh();
            setRawTarget(null);
          }}
        />
      )}
    </Box>
  );
}
