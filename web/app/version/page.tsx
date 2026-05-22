'use client';

import { Alert, Box, Card, CardContent, CircularProgress, Stack, Typography } from '@mui/material';
import { useEffect, useState } from 'react';
import PageHeader from '@/components/PageHeader';
import JsonBlock from '@/components/JsonBlock';
import { api } from '@/lib/api-client';

export default function VersionPage() {
  const [info, setInfo] = useState<Record<string, unknown> | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    void (async () => {
      try {
        const data = await api.get<Record<string, unknown>>('/api/v1/version');
        setInfo(data);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch version');
      }
    })();
  }, []);

  return (
    <Box>
      <PageHeader title="Server version" />
      {error && <Alert severity="error">{error}</Alert>}
      {!info && !error && (
        <Box display="flex" justifyContent="center" mt={4}>
          <CircularProgress />
        </Box>
      )}
      {info && (
        <Stack spacing={2}>
          <Card>
            <CardContent>
              <Stack spacing={1}>
                {Object.entries(info).map(([k, v]) => (
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
            </CardContent>
          </Card>
          <JsonBlock title="Raw" value={info} />
        </Stack>
      )}
    </Box>
  );
}
