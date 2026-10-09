'use client';

import Link from 'next/link';
import { useParams } from 'next/navigation';
import { type RemoteConfigSchema, remoteConfigSchemasPath } from '@entities/remoteconfigschema';
import { useApi } from '@shared/api';
import { TimeDisplay } from '@shared/preferences';
import {
  Alert,
  Button,
  CodeBlock,
  Collapsible,
  PageHeader,
  Spinner,
  TableWrap,
  Table,
  TableHead,
  TableRow,
  TableHeaderCell,
  TableBody,
  TableCell,
} from '@shared/ui';

export default function RemoteConfigSchemaDetailPage() {
  const { namespace, name } = useParams<{ namespace: string; name: string }>();
  const { data, error, isLoading } = useApi<RemoteConfigSchema>(
    `${remoteConfigSchemasPath(namespace)}/${encodeURIComponent(name)}`,
  );
  return (
    <div className="min-w-0 space-y-4">
      <Button variant="ghost" asChild>
        <Link href="/remoteconfigschemas">Back to schemas</Link>
      </Button>
      {isLoading ? (
        <Spinner className="mx-auto size-6" />
      ) : error || !data ? (
        <Alert severity="error">
          {error instanceof Error ? error.message : 'Schema not found'}
        </Alert>
      ) : (
        <>
          <PageHeader
            title={data.metadata.name}
            subtitle={`${namespace} · ${data.spec.binary} ${data.spec.version}`}
          />
          <p className="text-sm text-muted-foreground">
            Created: <TimeDisplay value={data.metadata.createdAt} />
          </p>
          {Object.keys(data.spec.components ?? {}).length === 0 && <p>No components</p>}
          {Object.entries(data.spec.components ?? {}).map(([componentClass, components]) => (
            <section key={componentClass} className="min-w-0 space-y-2">
              <h2 className="text-lg font-semibold">
                {componentClass} ({Object.keys(components).length})
              </h2>
              <TableWrap>
                <Table>
                  <TableHead>
                    <TableRow>
                      <TableHeaderCell>Type</TableHeaderCell>
                      <TableHeaderCell>Signals / pairs</TableHeaderCell>
                      <TableHeaderCell>Details</TableHeaderCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {Object.entries(components).map(([name, component]) => (
                      <TableRow key={name}>
                        <TableCell>{name}</TableCell>
                        <TableCell>
                          {[
                            ...(component.signals ?? []),
                            ...(component.pairs ?? []).map((pair) => `${pair.from} → ${pair.to}`),
                          ].join(', ') || '—'}
                        </TableCell>
                        <TableCell className="min-w-48">
                          <Collapsible label={`Full component: ${name}`}>
                            <CodeBlock value={component} />
                          </Collapsible>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </TableWrap>
            </section>
          ))}
        </>
      )}
    </div>
  );
}
