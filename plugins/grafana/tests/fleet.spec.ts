import { test, expect } from '@grafana/plugin-e2e';
import pluginJson from '../src/plugin.json';

test.skip(!process.env.COMMANDER_DEMO, 'Requires make grafana-demo.');

test('Viewer can inspect connected and disconnected demo collectors and refresh', async ({
  gotoAppPage,
  page,
}, testInfo) => {
  await gotoAppPage({ pluginId: pluginJson.id });
  const overview = page.getByRole('region', { name: 'OpAMP Commander overview' });
  await expect(overview.getByText('Local demo', { exact: true })).toBeVisible();
  const table = overview.getByRole('table');
  await expect(table.getByText('demo-collector-a', { exact: true })).toBeVisible();
  await expect(table.getByText('demo-collector-b', { exact: true })).toBeVisible();
  await expect(table.getByRole('cell', { name: 'Connected', exact: true })).toHaveCount(1);
  await expect(table.getByRole('cell', { name: 'Disconnected', exact: true })).toHaveCount(1);
  await overview.getByRole('button', { name: 'Refresh' }).click();
  await expect(table.getByText('demo-collector-a', { exact: true })).toBeVisible();
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
  await page.screenshot({ path: testInfo.outputPath('fleet.png'), fullPage: true });
});

test('Grafana backend reads the protected API without exposing its token or configuration', async ({
  request,
  readProvisionedDataSource,
}) => {
  const source = await readProvisionedDataSource({ fileName: 'opampcommander.yaml' });
  const health = await request.get(`/api/datasources/uid/${source.uid}/health`);
  expect((await health.json()).status).toBe('OK');
  const response = await request.post('/api/ds/query', {
    data: {
      from: '0',
      to: '1',
      queries: [{ refId: 'A', queryType: 'fleet', datasource: { uid: source.uid, type: source.type } }],
    },
  });
  const result = (await response.json()).results.A;
  expect(result.frames[0].data.values[1]).toEqual(expect.arrayContaining(['demo-collector-a', 'demo-collector-b']));
  expect(result.frames[0].schema.fields.map((field: { name: string }) => field.name)).toEqual([
    'Instance UID',
    'Name',
    'Namespace',
    'Type',
    'Connection',
    'Health',
    'Last reported',
  ]);
  const settings = await request.get(`/api/datasources/uid/${source.uid}`);
  expect((await settings.json()).secureJsonData).toBeUndefined();
});
