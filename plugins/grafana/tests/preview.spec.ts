import { test, expect } from '@grafana/plugin-e2e';
import pluginJson from '../src/plugin.json';

test.skip(!!process.env.COMMANDER_DEMO, 'The demo has a configured live data source.');

test('Viewer sees a setup error for an unconfigured data source', async ({ gotoAppPage, page }, testInfo) => {
  await gotoAppPage({ pluginId: pluginJson.id });
  const overview = page.getByRole('region', { name: 'OpAMP Commander overview' });
  await expect(overview.getByRole('heading', { name: 'Fleet overview' })).toBeVisible();
  await expect(overview.getByText(/configure a Commander URL/)).toBeVisible();
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
  await page.screenshot({ path: testInfo.outputPath('preview.png'), fullPage: true });
});

test('nested backend rejects unconfigured queries', async ({ request, readProvisionedDataSource }) => {
  const dataSource = await readProvisionedDataSource({ fileName: 'opampcommander.yaml' });
  const health = await request.get(`/api/datasources/uid/${dataSource.uid}/health`);
  expect(await health.json()).toEqual({
    status: 'ERROR',
    message: 'configure a Commander URL, namespace and access token in the data source',
  });

  const query = await request.post('/api/ds/query', {
    data: {
      from: '0',
      to: '1',
      queries: [{ refId: 'A', datasource: { uid: dataSource.uid, type: dataSource.type } }],
    },
  });
  const body = await query.json();
  expect(body.results.A.status).toBe(400);
  expect(body.results.A.error).toBe('configure a Commander URL, namespace and access token in the data source');
  expect(body.results.A.frames ?? []).toEqual([]);
});
