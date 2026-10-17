import { DataSourcePlugin } from '@grafana/data';
import { DataSource } from './datasource';
import { ConfigEditor, QueryEditor } from './ConfigEditor';

export const plugin = new DataSourcePlugin(DataSource).setConfigEditor(ConfigEditor).setQueryEditor(QueryEditor);
