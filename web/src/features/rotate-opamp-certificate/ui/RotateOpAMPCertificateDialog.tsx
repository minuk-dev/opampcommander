'use client';

import { useState } from 'react';
import { api, useApi, type ListResponse } from '@shared/api';
import {
  Alert,
  Button,
  Dialog,
  DialogBody,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Field,
  Input,
  Spinner,
} from '@shared/ui';
import type { Agent } from '@entities/agent';
import type { AgentGroup } from '@entities/agent-group';
import type { Certificate } from '@entities/certificate';

interface Props {
  namespace: string;
  group: AgentGroup;
  onClose: () => void;
  onApplied: () => void;
}

export default function RotateOpAMPCertificateDialog({
  namespace,
  group,
  onClose,
  onApplied,
}: Props) {
  const [certificateName, setCertificateName] = useState(
    group.spec.agentConfig?.connectionSettings?.opamp?.certificateName ?? '',
  );
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const groupUrl = `/api/v1/namespaces/${namespace}/agentgroups/${group.metadata.name}`;
  const {
    data: members,
    error: membersError,
    isLoading,
  } = useApi<ListResponse<Agent>>([`${groupUrl}/agents`, { limit: 2 }]);
  const singleAgent = members?.items.length === 1 && !members.metadata.continue;
  const endpoint = group.spec.agentConfig?.connectionSettings?.opamp?.destinationEndpoint;

  const save = async () => {
    const name = certificateName.trim();
    if (!singleAgent || !endpoint || !name) return;

    setBusy(true);
    setError(null);
    try {
      const certificate = await api.get<Certificate>(
        `/api/v1/namespaces/${namespace}/certificates/${encodeURIComponent(name)}`,
      );
      if (!certificate.spec.cert || !certificate.spec.privateKey) {
        throw new Error('Certificate needs a leaf certificate and private key.');
      }

      const latest = await api.get<AgentGroup>(groupUrl);
      const connectionSettings = latest.spec.agentConfig?.connectionSettings;
      if (!connectionSettings?.opamp?.destinationEndpoint) {
        throw new Error('Configure the group OpAMP destination endpoint first.');
      }

      await api.put(groupUrl, {
        ...latest,
        spec: {
          ...latest.spec,
          agentConfig: {
            ...latest.spec.agentConfig,
            connectionSettings: {
              ...connectionSettings,
              opamp: { ...connectionSettings.opamp, certificateName: name },
            },
          },
        },
      });
      onApplied();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to offer certificate');
    } finally {
      setBusy(false);
    }
  };

  return (
    <Dialog open onOpenChange={(next) => !next && onClose()}>
      <DialogContent size="sm">
        <DialogHeader>
          <DialogTitle>Rotate OpAMP client certificate</DialogTitle>
        </DialogHeader>
        <DialogBody className="space-y-3">
          {isLoading && <Spinner className="size-4" />}
          {membersError && <Alert severity="error">Failed to load group agents.</Alert>}
          {members && !singleAgent && (
            <Alert severity="warning">
              Rotation through a group requires exactly one agent. Each agent needs a certificate
              whose CN matches its own instance UID.
            </Alert>
          )}
          {!endpoint && (
            <Alert severity="warning">Configure the group OpAMP destination endpoint first.</Alert>
          )}
          {singleAgent && (
            <p className="break-all text-sm text-muted-foreground">
              Certificate CN must equal agent UID{' '}
              <code>{members.items[0].metadata.instanceUid}</code>. The agent will report the new
              settings as applied before it reconnects with the new certificate.
            </p>
          )}
          {error && <Alert severity="error">{error}</Alert>}
          <Field label="Certificate resource name">
            {(field) => (
              <Input
                {...field}
                value={certificateName}
                onChange={(event) => setCertificateName(event.target.value)}
                disabled={busy}
              />
            )}
          </Field>
        </DialogBody>
        <DialogFooter>
          <Button variant="ghost" onClick={onClose} disabled={busy}>
            Cancel
          </Button>
          <Button
            onClick={() => void save()}
            disabled={busy || isLoading || !singleAgent || !endpoint || !certificateName.trim()}
          >
            Offer certificate
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
