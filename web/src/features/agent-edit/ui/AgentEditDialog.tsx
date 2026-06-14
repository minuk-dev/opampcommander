'use client';

import { CodeEditorDialog } from '@shared/ui';
import { useNamespace } from '@entities/namespace';
import { api } from '@shared/api';
import type { Agent } from '@entities/agent';

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
      samplesUrl="/samples/agentspecs.yaml"
      onClose={onClose}
      onSave={async (parsed) => {
        // Reject non-object payloads explicitly so the user sees the error
        // instead of silently saving an empty spec.
        if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
          const got = Array.isArray(parsed) ? 'array' : typeof parsed;
          throw new Error(`spec must be an object (got ${got})`);
        }
        const next: Agent = { ...agent, spec: parsed as Agent['spec'] };
        const updated = await api.put<Agent>(
          `/api/v1/namespaces/${namespace}/agents/${agent.metadata.instanceUid}`,
          next,
        );
        onSaved(updated);
      }}
    />
  );
}
