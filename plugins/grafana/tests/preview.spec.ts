import { test, expect } from '@grafana/plugin-e2e';
import pluginJson from '../src/plugin.json';

test('Viewer can open the preview without a misleading fleet status', async ({ gotoAppPage, page }, testInfo) => {
  await gotoAppPage({ pluginId: pluginJson.id });
  const overview = page.getByRole('region', { name: 'OpAMP Commander overview' });
  await expect(overview.getByRole('heading', { name: 'Fleet overview' })).toBeVisible();
  await expect(overview.getByText(/Fleet queries are not available/)).toBeVisible();
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
  await page.screenshot({ path: testInfo.outputPath('preview.png'), fullPage: true });
});

test('nested backend reports unavailable health and queries', async ({ request, readProvisionedDataSource }) => {
  const dataSource = await readProvisionedDataSource({ fileName: 'opampcommander.yaml' });
  const health = await request.get(`/api/datasources/uid/${dataSource.uid}/health`);
  expect(await health.json()).toEqual({
    status: 'ERROR',
    message: 'Connections and fleet queries are not available in this preview.',
  });

  const query = await request.post('/api/ds/query', {
    data: {
      from: '0',
      to: '1',
      queries: [{ refId: 'A', datasource: { uid: dataSource.uid, type: dataSource.type } }],
    },
  });
  const body = await query.json();
  expect(body.results.A.status).toBe(501);
  expect(body.results.A.error).toBe('Connections and fleet queries are not available in this preview.');
  expect(body.results.A.frames ?? []).toEqual([]);
});
