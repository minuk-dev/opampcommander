'use client';

import CodeEditorDialog from '@/components/CodeEditorDialog';
import { useNamespace } from '@/components/NamespaceProvider';
import { api } from '@/lib/api-client';
import type { Agent } from '@/lib/types';

interface Props {
  open: boolean;
  agent: Agent;
  onClose: () => void;
  onSaved: (agent: Agent) => void;
}

export default function AgentEditDialog({ open, agent, onClose, onSaved }: Props) {
  const { namespace } = useNamespace();
  return (
    <CodeEditorDialog
      open={open}
      title="Edit agent spec"
      description={
        <>
          Edit <code>spec</code>. Common fields: <code>newInstanceUid</code>,{' '}
          <code>connectionSettings</code>, <code>remoteConfig</code>, <code>packagesAvailable</code>
          , <code>restartRequiredAt</code>.
        </>
      }
      initialValue={agent.spec ?? {}}
      onClose={onClose}
      onSave={async (parsed) => {
        const next: Agent = {
          ...agent,
          spec:
            parsed && typeof parsed === 'object' && !Array.isArray(parsed)
              ? (parsed as Agent['spec'])
              : {},
        };
        const updated = await api.put<Agent>(
          `/api/v1/namespaces/${namespace}/agents/${agent.metadata.instanceUid}`,
          next,
        );
        onSaved(updated);
      }}
    />
  );
}
