'use client';

import { Alert, Box, Snackbar } from '@mui/material';
import {
  Code as CodeIcon,
  PlaylistAddCheck as ApplyIcon,
  Sync as SyncIcon,
} from '@mui/icons-material';
import { useState } from 'react';
import { useNamespace } from '@entities/namespace';
import dynamic from 'next/dynamic';
import { ResourceListPage } from '@widgets/resource-list-page';
import { TimeDisplay } from '@shared/preferences';
import { api } from '@shared/api';
import { reconcileResource } from '@features/reconcile';
import type { AgentRemoteConfig } from '@entities/agent-remote-config';

// Lazy-loaded: none of these dialogs is needed to render the list, and the
// config editor pulls in the highlighter and diff chunks on top.
const AgentRemoteConfigEditDialog = dynamic(
  () => import('@features/agent-remote-config-edit/ui/AgentRemoteConfigEditDialog'),
);
const JsonEditorDialog = dynamic(() => import('@shared/ui/JsonEditorDialog'));
const ApplyToGroupDialog = dynamic(
  () => import('@features/apply-remote-config/ui/ApplyToGroupDialog'),
);

type ReconcileFeedback = { severity: 'success' | 'error'; message: string };

// Raw editing stays available for fields the form does not model (schemaRefs
// today) and for pasting a whole manifest.
interface RawTarget {
  row: AgentRemoteConfig;
  refresh: () => void;
}

export default function AgentRemoteConfigsPage() {
  const { namespace } = useNamespace();
  const [applyTarget, setApplyTarget] = useState<AgentRemoteConfig | null>(null);
  const [rawTarget, setRawTarget] = useState<RawTarget | null>(null);
  const [reconcileFeedback, setReconcileFeedback] = useState<ReconcileFeedback | null>(null);

  const reconcileConfig = async (c: AgentRemoteConfig) => {
    try {
      await reconcileResource('agentremoteconfig', namespace, c.metadata.name);
      setReconcileFeedback({
        severity: 'success',
        message: `Reconciled "${c.metadata.name}": detected endpoints and re-propagated to groups.`,
      });
    } catch (err) {
      setReconcileFeedback({
        severity: 'error',
        message: err instanceof Error ? err.message : `Failed to reconcile "${c.metadata.name}".`,
      });
    }
  };

  return (
    <Box>
      <ResourceListPage<AgentRemoteConfig>
        title="Agent Remote Configs"
        subtitle={`Namespace: ${namespace}`}
        listPath={`/api/v1/namespaces/${namespace}/agentremoteconfigs`}
        itemPath={(c) => `/api/v1/namespaces/${namespace}/agentremoteconfigs/${c.metadata.name}`}
        itemName={(c) => c.metadata.name}
        canEdit
        canDelete
        extraActions={(c, { refresh }) => [
          {
            label: 'Edit as YAML',
            icon: <CodeIcon fontSize="small" />,
            onClick: () => setRawTarget({ row: c, refresh }),
          },
          {
            label: 'Apply to agent group',
            icon: <ApplyIcon fontSize="small" />,
            onClick: () => setApplyTarget(c),
          },
          {
            label: 'Reconcile',
            icon: <SyncIcon fontSize="small" />,
            onClick: () => void reconcileConfig(c),
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
          <AgentRemoteConfigEditDialog
            open={open}
            mode="create"
            namespace={namespace}
            onClose={onClose}
            onSaved={onSaved}
          />
        )}
        renderEdit={({ open, row, onClose, onSaved }) => (
          <AgentRemoteConfigEditDialog
            open={open}
            mode="edit"
            namespace={namespace}
            initial={row}
            onClose={onClose}
            onSaved={onSaved}
          />
        )}
      />
      {applyTarget !== null && (
        <ApplyToGroupDialog
          open
          namespace={namespace}
          config={applyTarget}
          onClose={() => setApplyTarget(null)}
          onApplied={() => setApplyTarget(null)}
        />
      )}
      {rawTarget !== null && (
        <JsonEditorDialog
          open
          title={`Edit ${rawTarget.row.metadata.name} (raw)`}
          description="Edit the whole resource, including fields the form does not expose."
          initialValue={rawTarget.row}
          samplesUrl="/samples/agentremoteconfigs.yaml"
          samplesVars={{ namespace }}
          onClose={() => setRawTarget(null)}
          onSave={async (parsed) => {
            await api.put(
              `/api/v1/namespaces/${namespace}/agentremoteconfigs/${rawTarget.row.metadata.name}`,
              parsed as AgentRemoteConfig,
            );
            rawTarget.refresh();
            setRawTarget(null);
          }}
        />
      )}
      <Snackbar
        open={reconcileFeedback !== null}
        autoHideDuration={4000}
        onClose={() => setReconcileFeedback(null)}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
      >
        {reconcileFeedback === null ? undefined : (
          <Alert
            severity={reconcileFeedback.severity}
            onClose={() => setReconcileFeedback(null)}
            variant="filled"
          >
            {reconcileFeedback.message}
          </Alert>
        )}
      </Snackbar>
    </Box>
  );
}
