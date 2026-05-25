'use client';

import { Alert, Box, Card, CardContent, CircularProgress, Stack, Typography } from '@mui/material';
import { useEffect, useState } from 'react';
import PageHeader from '@/components/PageHeader';
import JsonBlock from '@/components/JsonBlock';
import { api } from '@/lib/api-client';
import { WEB_VERSION } from '@/lib/web-version';

function InfoRows({ entries }: { entries: Array<[string, unknown]> }) {
  return (
    <Stack spacing={1}>
      {entries.map(([k, v]) => (
        <Stack key={k} direction="row" justifyContent="space-between">
          <Typography variant="body2" color="text.secondary">
            {k}
          </Typography>
          <Typography variant="body2" sx={{ fontFamily: 'monospace' }}>
            {typeof v === 'string' ? v : JSON.stringify(v)}
          </Typography>
        </Stack>
      ))}
    </Stack>
  );
}

export default function VersionPage() {
  const [apiInfo, setApiInfo] = useState<Record<string, unknown> | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    void (async () => {
      try {
        const data = await api.get<Record<string, unknown>>('/api/v1/version');
        setApiInfo(data);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch version');
      }
    })();
  }, []);

  const webInfo = { version: WEB_VERSION };

  return (
    <Box>
      <PageHeader title="Version" />
      <Stack spacing={2}>
        <Card>
          <CardContent>
            <Typography variant="overline" color="text.secondary">
              Web
            </Typography>
            <Box mt={1}>
              <InfoRows entries={Object.entries(webInfo)} />
            </Box>
          </CardContent>
        </Card>

        <Card>
          <CardContent>
            <Typography variant="overline" color="text.secondary">
              API server
            </Typography>
            {error && (
              <Box mt={1}>
                <Alert severity="error">{error}</Alert>
              </Box>
            )}
            {!apiInfo && !error && (
              <Box display="flex" justifyContent="center" mt={2}>
                <CircularProgress size={24} />
              </Box>
            )}
            {apiInfo && (
              <Box mt={1}>
                <InfoRows entries={Object.entries(apiInfo)} />
              </Box>
            )}
          </CardContent>
        </Card>

        {apiInfo && <JsonBlock title="Raw" value={{ web: webInfo, apiserver: apiInfo }} />}
      </Stack>
    </Box>
  );
}
