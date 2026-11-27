import React from 'react';
import { DataSourceJsonData, DataSourcePluginOptionsEditorProps } from '@grafana/data';
import { Alert, Button, Field, Input, Switch } from '@grafana/ui';

export interface CommanderOptions extends DataSourceJsonData {
  demo?: boolean;
  url?: string;
  namespace?: string;
  allowInsecureHTTP?: boolean;
}
interface Secrets {
  token?: string;
}

export function ConfigEditor({
  options,
  onOptionsChange,
}: DataSourcePluginOptionsEditorProps<CommanderOptions, Secrets>) {
  const update = (patch: Partial<CommanderOptions>) =>
    onOptionsChange({
      ...options,
      jsonData: { ...options.jsonData, namespace: options.jsonData.namespace || 'default', ...patch },
    });
  return (
    <div style={{ maxWidth: 640 }}>
      <Alert title="Read-only access" severity="info">
        Connect to a private Commander service or an access-controlled proxy. Everyone allowed to query this data source
        can view agents in its configured namespace.
      </Alert>
      <Field label="Commander URL">
        <Input
          id="commander-url"
          placeholder="https://commander.example.com"
          value={options.jsonData.url || ''}
          onChange={(event) => update({ url: event.currentTarget.value })}
        />
      </Field>
      <Field label="Namespace">
        <Input
          id="commander-namespace"
          value={options.jsonData.namespace || 'default'}
          onChange={(event) => update({ namespace: event.currentTarget.value })}
        />
      </Field>
      <Field
        label="Access token (optional)"
        description="Supply a token when the upstream requires bearer authentication."
      >
        <Input
          id="commander-token"
          type="password"
          autoComplete="new-password"
          placeholder={options.secureJsonFields?.token ? 'Configured' : 'Commander access token'}
          value={options.secureJsonData?.token || ''}
          onChange={(event) =>
            onOptionsChange({
              ...options,
              secureJsonData: { ...options.secureJsonData, token: event.currentTarget.value },
            })
          }
        />
      </Field>
      {options.secureJsonFields?.token && (
        <Button
          variant="secondary"
          onClick={() =>
            onOptionsChange({
              ...options,
              secureJsonFields: { ...options.secureJsonFields, token: false },
              secureJsonData: { ...options.secureJsonData, token: '' },
            })
          }
        >
          Reset token
        </Button>
      )}
      <Field label="Allow HTTP for local development" description="Use only on a trusted development network.">
        <Switch
          id="commander-http"
          value={!!options.jsonData.allowInsecureHTTP}
          onChange={(event) => update({ allowInsecureHTTP: event.currentTarget.checked })}
        />
      </Field>
    </div>
  );
}

export function QueryEditor() {
  return (
    <p>
      Lists the first 100 agents in the configured namespace. Use Fleet overview to navigate additional pages. This is
      current state, not time-series history.
    </p>
  );
}
