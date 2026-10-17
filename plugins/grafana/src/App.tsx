import React, { useEffect, useMemo, useState } from 'react';
import { css } from '@emotion/css';
import { DataFrame, DataFrameJSON, dataFrameFromJSON } from '@grafana/data';
import { getBackendSrv, getDataSourceSrv, PluginPage } from '@grafana/runtime';
import { CommanderOptions } from './datasource/ConfigEditor';
import { Alert, Button, Field, Combobox, useStyles2 } from '@grafana/ui';

const pluginId = 'minukdev-opampcommander-datasource';
interface QueryResult {
  results: { A: { frames?: DataFrameJSON[]; error?: string } };
}

export function App() {
  const sources = useMemo(() => getDataSourceSrv().getList({ type: pluginId }), []);
  const [uid, setUid] = useState(sources[0]?.uid ?? '');
  const [cursor, setCursor] = useState('');
  const [refresh, setRefresh] = useState(0);
  const [frame, setFrame] = useState<DataFrame>();
  const [loading, setLoading] = useState(sources.length > 0);
  const [error, setError] = useState('');
  const source = sources.find((item) => item.uid === uid);
  const config = source?.jsonData as CommanderOptions | undefined;
  const styles = useStyles2((theme) => ({
    controls: css({
      display: 'flex',
      gap: theme.spacing(2),
      alignItems: 'end',
      flexWrap: 'wrap',
      marginBottom: theme.spacing(2),
    }),
    source: css({ width: 320, maxWidth: '100%' }),
    cards: css({ display: 'flex', flexWrap: 'wrap', gap: theme.spacing(2), margin: `${theme.spacing(2)} 0` }),
    card: css({
      padding: theme.spacing(2),
      border: `1px solid ${theme.colors.border.weak}`,
      borderRadius: theme.shape.radius.default,
      minWidth: 130,
    }),
    scroll: css({ maxWidth: '100%', overflowX: 'auto', marginBottom: theme.spacing(2) }),
    table: css({
      width: '100%',
      borderCollapse: 'collapse',
      '& th, & td': {
        padding: theme.spacing(1.5),
        textAlign: 'left',
        borderBottom: `1px solid ${theme.colors.border.weak}`,
        whiteSpace: 'nowrap',
      },
    }),
  }));

  const reload = (nextUid = uid, nextCursor = cursor) => {
    setFrame(undefined);
    setError('');
    setLoading(true);
    setUid(nextUid);
    setCursor(nextCursor);
    setRefresh((value) => value + 1);
  };

  useEffect(() => {
    if (!uid) {
      return;
    }
    const subscription = getBackendSrv()
      .fetch<QueryResult>({
        url: '/api/ds/query',
        method: 'POST',
        data: {
          from: '0',
          to: '1',
          queries: [{ refId: 'A', datasource: { uid, type: pluginId }, queryType: 'fleet', cursor }],
        },
      })
      .subscribe({
        next: ({ data }) => {
          const result = data.results?.A;
          if (result?.error) {
            setError(result.error);
          } else if (result?.frames?.[0]) {
            setFrame(dataFrameFromJSON(result.frames[0]));
          } else {
            setError('Commander returned no fleet frame.');
          }
          setLoading(false);
        },
        error: (failure) => {
          setError(
            failure?.data?.results?.A?.error ||
              'Fleet request failed. Check the data source connection and access token.'
          );
          setLoading(false);
        },
      });
    return () => subscription.unsubscribe();
  }, [uid, cursor, refresh]);

  const nextCursor = typeof frame?.meta?.custom?.nextCursor === 'string' ? frame.meta.custom.nextCursor : '';
  const values = (field: string) => frame?.fields.find((item) => item.name === field)?.values ?? [];
  const count = (field: string, value: string) => values(field).filter((item) => item === value).length;

  return (
    <PluginPage>
      <section aria-label="OpAMP Commander overview" style={{ minWidth: 0 }}>
        <h2>Fleet overview</h2>
        <p>Current collector status reported to OpAMP Commander.</p>
        {config?.demo && (
          <Alert title="Local demo" severity="info">
            These are demo collectors connected to a local Commander server. This is not your production fleet.
          </Alert>
        )}
        {!sources.length ? (
          <Alert title="Connect Commander" severity="info">
            Add an OpAMP Commander data source in Grafana Connections, then configure its URL, namespace and access
            token.
          </Alert>
        ) : (
          <>
            <div className={styles.controls}>
              <div className={styles.source}>
                <Field label="Data source">
                  <Combobox
                    id="fleet-source"
                    value={uid}
                    options={sources.map((item) => ({ label: item.name, value: item.uid }))}
                    onChange={(item) => {
                      reload(item.value!, '');
                    }}
                  />
                </Field>
              </div>
              <Button icon="sync" disabled={loading} onClick={() => reload()}>
                Refresh
              </Button>
            </div>
            <p>
              Namespace: <strong>{config?.namespace || 'Not configured'}</strong>. Access is limited by the data source
              identity.
            </p>
            {loading && <p role="status">Loading fleet…</p>}
            {error && (
              <Alert title="Unable to load fleet" severity="error">
                {error}
              </Alert>
            )}
            {frame && (
              <>
                <div className={styles.cards} aria-label="Current page summary">
                  <div className={styles.card}>
                    Agents on this page
                    <br />
                    <strong>{frame.length}</strong>
                  </div>
                  <div className={styles.card}>
                    Connected
                    <br />
                    <strong>{count('Connection', 'Connected')}</strong>
                  </div>
                  <div className={styles.card}>
                    Unhealthy
                    <br />
                    <strong>{count('Health', 'Unhealthy')}</strong>
                  </div>
                </div>
                {frame.length === 0 ? (
                  <Alert title="No agents found" severity="info">
                    No agents were returned for this namespace.
                  </Alert>
                ) : (
                  <div className={styles.scroll} tabIndex={0} role="region" aria-label="Agent table">
                    <table className={styles.table}>
                      <thead>
                        <tr>
                          {frame.fields.map((field) => (
                            <th scope="col" key={field.name}>
                              {field.name}
                            </th>
                          ))}
                        </tr>
                      </thead>
                      <tbody>
                        {Array.from({ length: frame.length }, (_, row) => (
                          <tr key={String(values('Instance UID')[row])}>
                            {frame.fields.map((field) => (
                              <td key={field.name}>{String(field.values[row] || '—')}</td>
                            ))}
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
                <p>
                  Up to 100 agents per page. Health is the last reported health; disconnected agents may have an older
                  report.
                </p>
              </>
            )}
            <div className={styles.controls}>
              <Button variant="secondary" disabled={!cursor || loading} onClick={() => reload(uid, '')}>
                First page
              </Button>
              <Button variant="secondary" disabled={!nextCursor || loading} onClick={() => reload(uid, nextCursor)}>
                Next page
              </Button>
            </div>
          </>
        )}
      </section>
    </PluginPage>
  );
}
