import { Alert, Box, Card, CardContent, Stack, Typography } from '@mui/material';
import PageHeader from '@/components/PageHeader';
import JsonBlock from '@/components/JsonBlock';
import { serverGet } from '@/lib/server-api';
import { getWebVersionInfo } from '@/lib/web-version';

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

// Server Component: the API version is fetched server-side using the bearer
// token from the httpOnly session cookie (see lib/server-api). No client-side
// effect/loading state needed — the page streams in already populated.
export default async function VersionPage() {
  const webInfo = getWebVersionInfo();

  let apiInfo: Record<string, unknown> | null = null;
  let error: string | null = null;
  try {
    apiInfo = await serverGet<Record<string, unknown>>('/api/v1/version');
  } catch (err) {
    error = err instanceof Error ? err.message : 'Failed to fetch version';
  }

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
