import React from 'react';
import { PluginPage } from '@grafana/runtime';
import { Alert } from '@grafana/ui';

export function App() {
  return (
    <PluginPage>
      <section aria-label="OpAMP Commander overview">
        <h2>Fleet overview</h2>
        <p>Inspect your OpenTelemetry collector fleet from Grafana.</p>
        <Alert title="Development preview" severity="info">
          Fleet queries are not available in this preview. Continue using OpAMP Commander to view and manage your
          agents.
        </Alert>
      </section>
    </PluginPage>
  );
}
