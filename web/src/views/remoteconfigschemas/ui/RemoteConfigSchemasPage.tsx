'use client';

import Link from 'next/link';
import { useNamespace } from '@entities/namespace';
import { type RemoteConfigSchema, remoteConfigSchemasPath } from '@entities/remoteconfigschema';
import { api } from '@shared/api';
import { CodeEditorDialog } from '@shared/ui';
import { TimeDisplay } from '@shared/preferences';
import { ResourceListPage } from '@widgets/resource-list-page';

export default function RemoteConfigSchemasPage() {
  const { namespace } = useNamespace();
  const path = remoteConfigSchemasPath(namespace);
  const detailHref = (schema: RemoteConfigSchema) =>
    `/remoteconfigschemas/${encodeURIComponent(namespace)}/${encodeURIComponent(schema.metadata.name)}`;
  return (
    <ResourceListPage<RemoteConfigSchema>
      key={namespace}
      title="Remote Config Schemas"
      subtitle={`Namespace: ${namespace}`}
      listPath={path}
      itemPath={(schema) => `${path}/${encodeURIComponent(schema.metadata.name)}`}
      itemName={(schema) => schema.metadata.name}
      detailHref={detailHref}
      filterable
      canEdit
      canDelete
      columns={[
        {
          header: 'Name',
          render: (schema) => (
            <Link className="text-primary hover:underline" href={detailHref(schema)}>
              {schema.metadata.name}
            </Link>
          ),
        },
        { header: 'Binary', render: (schema) => schema.spec.binary },
        { header: 'Version', render: (schema) => schema.spec.version },
        {
          header: 'Components',
          render: (schema) =>
            Object.values(schema.spec.components ?? {}).reduce(
              (count, components) => count + Object.keys(components).length,
              0,
            ),
        },
        {
          header: 'Created',
          render: (schema) => <TimeDisplay value={schema.metadata.createdAt} />,
        },
      ]}
      renderCreate={({ open, onClose, onSaved }) => (
        <CodeEditorDialog
          open={open}
          title="Create remote config schema"
          description="Set the name, collector binary and version, and component catalog grouped by class."
          initialValue={{
            metadata: { name: '', namespace, attributes: {} },
            spec: {
              binary: 'otelcol-contrib',
              version: '',
              components: {
                receivers: {},
                processors: {},
                exporters: {},
                extensions: {},
                connectors: {},
              },
            },
          }}
          onClose={onClose}
          onSave={async (parsed) => {
            await api.post(path, parsed);
            onSaved();
          }}
        />
      )}
      renderEdit={({ open, row, onClose, onSaved }) => (
        <CodeEditorDialog
          open={open}
          title={`Edit ${row.metadata.name}`}
          initialValue={row}
          onClose={onClose}
          onSave={async (parsed) => {
            await api.put(`${path}/${encodeURIComponent(row.metadata.name)}`, parsed);
            onSaved();
          }}
        />
      )}
    />
  );
}
