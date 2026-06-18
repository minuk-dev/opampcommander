'use client';

import { useNamespace } from '@entities/namespace';
import dynamic from 'next/dynamic';
import { ResourceListPage } from '@widgets/resource-list-page';
import { TimeDisplay } from '@shared/preferences';
import { api } from '@shared/api';
import type { EndpointSignals } from '@shared/api';
import type { Endpoint } from '@entities/endpoint';

// Lazy-loaded: the JSON editor pulls in js-yaml — load only when a create/edit
// dialog is opened, not in the initial route bundle.
const JsonEditorDialog = dynamic(() => import('@shared/ui/JsonEditorDialog'));

function signalsLabel(signals?: EndpointSignals): string {
  if (!signals) return '-';
  const enabled = [
    signals.metrics ? 'metrics' : null,
    signals.logs ? 'logs' : null,
    signals.traces ? 'traces' : null,
  ].filter(Boolean);
  return enabled.length > 0 ? enabled.join(', ') : '-';
}

function emptyEndpoint(namespace: string): Endpoint {
  return {
    metadata: {
      name: '',
      namespace,
      attributes: {},
      createdAt: new Date().toISOString(),
    },
    spec: {
      url: '',
      protocol: 'otlp',
      signals: { metrics: false, logs: false, traces: false },
      tenants: [],
    },
  };
}

export default function EndpointsPage() {
  const { namespace } = useNamespace();
  const basePath = `/api/v1/namespaces/${namespace}/endpoints`;
  return (
    <ResourceListPage<Endpoint>
      title="Endpoints"
      subtitle={`Namespace: ${namespace}`}
      listPath={basePath}
      itemPath={(e) => `${basePath}/${e.metadata.name}`}
      itemName={(e) => e.metadata.name}
      deps={[namespace]}
      canEdit
      canDelete
      columns={[
        { header: 'Name', render: (e) => e.metadata.name },
        { header: 'URL', render: (e) => e.spec.url || '-' },
        { header: 'Protocol', render: (e) => e.spec.protocol || '-' },
        { header: 'Signals', render: (e) => signalsLabel(e.spec.signals) },
        { header: 'Tenants', render: (e) => String(e.spec.tenants?.length ?? 0) },
        { header: 'Created', render: (e) => <TimeDisplay value={e.metadata.createdAt} /> },
      ]}
      renderCreate={({ open, onClose, onSaved }) => (
        <JsonEditorDialog
          open={open}
          title="Create endpoint"
          description="metadata.name + spec.url + spec.protocol + spec.signals (metrics/logs/traces). Use spec.tenants for multi-tenant headers/tags."
          initialValue={emptyEndpoint(namespace)}
          samplesUrl="/samples/endpoints.yaml"
          samplesVars={{ namespace }}
          onClose={onClose}
          onSave={async (parsed) => {
            await api.post(basePath, parsed as Endpoint);
            onSaved();
          }}
        />
      )}
      renderEdit={({ open, row, onClose, onSaved }) => (
        <JsonEditorDialog
          open={open}
          title={`Edit ${row.metadata.name}`}
          initialValue={row}
          samplesUrl="/samples/endpoints.yaml"
          samplesVars={{ namespace }}
          onClose={onClose}
          onSave={async (parsed) => {
            await api.put(`${basePath}/${row.metadata.name}`, parsed as Endpoint);
            onSaved();
          }}
        />
      )}
    />
  );
}
