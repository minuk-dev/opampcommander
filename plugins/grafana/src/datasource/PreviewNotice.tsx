import React from 'react';
import { Alert } from '@grafana/ui';

export function PreviewNotice() {
  return (
    <Alert title="Development preview" severity="info">
      Connections and fleet queries are not available in this preview.
    </Alert>
  );
}
