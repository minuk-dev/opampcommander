import CopyWebpackPlugin from 'copy-webpack-plugin';
import grafanaConfig, { Env } from './.config/webpack/webpack.config';

export default async function config(env: Env) {
  const base = await grafanaConfig(env);
  // The generated config copies the app logo only; nested plugins need their own asset path.
  base.plugins?.push(new CopyWebpackPlugin({ patterns: [{ from: 'datasource/img/logo.svg', to: 'datasource/img/' }] }));
  return base;
}
