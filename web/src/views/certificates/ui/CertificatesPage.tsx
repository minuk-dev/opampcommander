'use client';

import { Box, Chip } from '@mui/material';
import { useNamespace } from '@entities/namespace';
import { ResourceListPage } from '@widgets/resource-list-page';
import { JsonEditorDialog } from '@shared/ui';
import { TimeDisplay } from '@shared/preferences';
import { api } from '@shared/api';
import type { Certificate } from '@entities/certificate';

function emptyCertificate(namespace: string): Certificate {
  return {
    kind: 'Certificate',
    apiVersion: 'v1',
    metadata: {
      name: '',
      namespace,
      attributes: {},
      createdAt: new Date().toISOString(),
    },
    spec: {
      cert: '',
      privateKey: '',
      caCert: '',
    },
  };
}

export default function CertificatesPage() {
  const { namespace } = useNamespace();
  return (
    <Box>
      <ResourceListPage<Certificate>
        title="Certificates"
        subtitle={`Namespace: ${namespace}`}
        listPath={`/api/v1/namespaces/${namespace}/certificates`}
        itemPath={(c) => `/api/v1/namespaces/${namespace}/certificates/${c.metadata.name}`}
        itemName={(c) => c.metadata.name}
        deps={[namespace]}
        canEdit
        canDelete
        columns={[
          { header: 'Name', render: (c) => c.metadata.name },
          {
            header: 'Has cert',
            render: (c) => (
              <Chip
                size="small"
                label={c.spec.cert ? 'yes' : 'no'}
                color={c.spec.cert ? 'success' : 'default'}
              />
            ),
          },
          {
            header: 'Has private key',
            render: (c) => (
              <Chip
                size="small"
                label={c.spec.privateKey ? 'yes' : 'no'}
                color={c.spec.privateKey ? 'success' : 'default'}
              />
            ),
          },
          {
            header: 'Has CA',
            render: (c) => (
              <Chip
                size="small"
                label={c.spec.caCert ? 'yes' : 'no'}
                color={c.spec.caCert ? 'success' : 'default'}
              />
            ),
          },
          { header: 'Created', render: (c) => <TimeDisplay value={c.metadata.createdAt} /> },
        ]}
        renderCreate={({ open, onClose, onSaved }) => (
          <JsonEditorDialog
            open={open}
            title="Create certificate"
            description="PEM-encoded cert/privateKey/caCert as JSON strings."
            initialValue={emptyCertificate(namespace)}
            samplesUrl="/samples/certificates.yaml"
            samplesVars={{ namespace }}
            onClose={onClose}
            onSave={async (parsed) => {
              await api.post(`/api/v1/namespaces/${namespace}/certificates`, parsed as Certificate);
              onSaved();
            }}
          />
        )}
        renderEdit={({ open, row, onClose, onSaved }) => (
          <JsonEditorDialog
            open={open}
            title={`Edit ${row.metadata.name}`}
            initialValue={row}
            samplesUrl="/samples/certificates.yaml"
            samplesVars={{ namespace }}
            onClose={onClose}
            onSave={async (parsed) => {
              await api.put(
                `/api/v1/namespaces/${namespace}/certificates/${row.metadata.name}`,
                parsed as Certificate,
              );
              onSaved();
            }}
          />
        )}
      />
    </Box>
  );
}
