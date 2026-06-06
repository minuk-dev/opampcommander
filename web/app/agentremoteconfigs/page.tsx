'use client';

import { Box } from '@mui/material';
import { PlaylistAddCheck as ApplyIcon } from '@mui/icons-material';
import { useState } from 'react';
import { useNamespace } from '@/components/NamespaceProvider';
import ResourceListPage from '@/components/ResourceListPage';
import JsonEditorDialog from '@/components/JsonEditorDialog';
import TimeDisplay from '@/components/TimeDisplay';
import { api } from '@/lib/api-client';
import type { AgentRemoteConfig } from '@/lib/types';
import ApplyToGroupDialog from './ApplyToGroupDialog';

function emptyConfig(namespace: string): AgentRemoteConfig {
  return {
    metadata: {
      name: '',
      namespace,
      attributes: {},
      createdAt: new Date().toISOString(),
    },
    spec: { value: '', contentType: 'text/yaml' },
  };
}

export default function AgentRemoteConfigsPage() {
  const { namespace } = useNamespace();
  const [applyTarget, setApplyTarget] = useState<AgentRemoteConfig | null>(null);
  return (
    <Box>
      <ResourceListPage<AgentRemoteConfig>
        title="Agent Remote Configs"
        subtitle={`Namespace: ${namespace}`}
        listPath={`/api/v1/namespaces/${namespace}/agentremoteconfigs`}
        itemPath={(c) => `/api/v1/namespaces/${namespace}/agentremoteconfigs/${c.metadata.name}`}
        itemName={(c) => c.metadata.name}
        deps={[namespace]}
        canEdit
        canDelete
        extraActions={(c) => [
          {
            label: 'Apply to agent group',
            icon: <ApplyIcon fontSize="small" />,
            onClick: () => setApplyTarget(c),
          },
        ]}
        columns={[
          { header: 'Name', render: (c) => c.metadata.name },
          { header: 'Content type', render: (c) => c.spec.contentType || '-' },
          {
            header: 'Preview',
            render: (c) => (
              <span style={{ fontFamily: 'monospace', fontSize: 12 }}>
                {(c.spec.value || '').slice(0, 60)}
                {(c.spec.value || '').length > 60 ? '…' : ''}
              </span>
            ),
          },
          { header: 'Created', render: (c) => <TimeDisplay value={c.metadata.createdAt} /> },
        ]}
        renderCreate={({ open, onClose, onSaved }) => (
          <JsonEditorDialog
            open={open}
            title="Create remote config"
            description="metadata.name + spec.value (config body) + spec.contentType (e.g. text/yaml)."
            initialValue={emptyConfig(namespace)}
            samplesUrl="/samples/agentremoteconfigs.yaml"
            samplesVars={{ namespace }}
            onClose={onClose}
            onSave={async (parsed) => {
              await api.post(
                `/api/v1/namespaces/${namespace}/agentremoteconfigs`,
                parsed as AgentRemoteConfig,
              );
              onSaved();
            }}
          />
        )}
        renderEdit={({ open, row, onClose, onSaved }) => (
          <JsonEditorDialog
            open={open}
            title={`Edit ${row.metadata.name}`}
            initialValue={row}
            samplesUrl="/samples/agentremoteconfigs.yaml"
            samplesVars={{ namespace }}
            onClose={onClose}
            onSave={async (parsed) => {
              await api.put(
                `/api/v1/namespaces/${namespace}/agentremoteconfigs/${row.metadata.name}`,
                parsed as AgentRemoteConfig,
              );
              onSaved();
            }}
          />
        )}
      />
      <ApplyToGroupDialog
        open={applyTarget !== null}
        namespace={namespace}
        config={applyTarget}
        onClose={() => setApplyTarget(null)}
        onApplied={() => setApplyTarget(null)}
      />
    </Box>
  );
}
