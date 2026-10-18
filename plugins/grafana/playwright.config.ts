import type { PluginOptions } from '@grafana/plugin-e2e';
import { defineConfig } from '@playwright/test';
import baseConfig from './.config/playwright.config';

export default defineConfig<PluginOptions>({
  ...baseConfig,
  // Grafana loads plugin bundles lazily; allow for a cold development instance.
  expect: { timeout: 15000 },
  workers: 1,
  use: { ...baseConfig.use, baseURL: process.env.GRAFANA_URL || 'http://127.0.0.1:3300' },
  projects: [
    { name: 'desktop', use: { viewport: { width: 1280, height: 900 } } },
    { name: 'mobile', use: { viewport: { width: 375, height: 812 } } },
  ],
});
