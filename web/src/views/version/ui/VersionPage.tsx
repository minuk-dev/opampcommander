import { Alert, Card, CardContent, CardHeader, CardTitle, JsonBlock, PageHeader } from '@shared/ui';
import { serverGet } from '@shared/api/server';
import { getWebVersionInfo } from '@shared/lib/web-version';

function InfoRows({ entries }: { entries: Array<[string, unknown]> }) {
  return (
    <dl className="space-y-1 text-sm">
      {entries.map(([k, v]) => (
        <div key={k} className="flex justify-between gap-4">
          <dt className="text-muted-foreground">{k}</dt>
          <dd className="truncate font-mono">{typeof v === 'string' ? v : JSON.stringify(v)}</dd>
        </div>
      ))}
    </dl>
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
    <div>
      <PageHeader title="Version" />
      <div className="space-y-3">
        <Card>
          <CardHeader>
            <CardTitle>Web</CardTitle>
          </CardHeader>
          <CardContent>
            <InfoRows entries={Object.entries(webInfo)} />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>API server</CardTitle>
          </CardHeader>
          <CardContent className="space-y-2">
            {error && <Alert severity="error">{error}</Alert>}
            {apiInfo && <InfoRows entries={Object.entries(apiInfo)} />}
          </CardContent>
        </Card>

        {apiInfo && (
          <Card>
            <CardContent className="pt-4">
              <JsonBlock title="Raw" value={{ web: webInfo, apiserver: apiInfo }} />
            </CardContent>
          </Card>
        )}
      </div>
    </div>
  );
}
