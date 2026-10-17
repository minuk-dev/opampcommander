import { DataSourcePlugin } from '@grafana/data';
import { DataSource } from './datasource';
import { PreviewNotice } from './PreviewNotice';

export const plugin = new DataSourcePlugin(DataSource).setConfigEditor(PreviewNotice).setQueryEditor(PreviewNotice);
