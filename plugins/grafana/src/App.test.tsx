import React from 'react';
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { dataFrameToJSON, toDataFrame } from '@grafana/data';
import { of, Subject } from 'rxjs';
import { App } from './App';

const mockGetList = jest.fn();
const mockFetch = jest.fn();
jest.mock('@grafana/runtime', () => ({
  PluginPage: ({ children }: React.PropsWithChildren) => <main>{children}</main>,
  getDataSourceSrv: () => ({ getList: mockGetList }),
  getBackendSrv: () => ({ fetch: mockFetch }),
}));

beforeEach(() => {
  jest.clearAllMocks();
  mockGetList.mockReturnValue([{ uid: 'commander', name: 'Commander', jsonData: { namespace: 'default' } }]);
});

function result(names: string[], nextCursor = '') {
  const frame = toDataFrame({
    fields: [
      { name: 'Instance UID', values: names },
      { name: 'Name', values: names },
      { name: 'Connection', values: names.map(() => 'Connected') },
      { name: 'Health', values: names.map(() => 'Unknown') },
    ],
  });
  frame.meta = { custom: { nextCursor } };
  return { data: { results: { A: { frames: [dataFrameToJSON(frame)] } } } };
}

test('explains how to connect when no data source exists', () => {
  mockGetList.mockReturnValue([]);
  render(<App />);
  expect(screen.getByText('Connect Commander')).toBeInTheDocument();
  expect(mockFetch).not.toHaveBeenCalled();
});

test('loads real frames and passes only the returned pagination cursor', async () => {
  mockFetch.mockReturnValueOnce(of(result(['collector-a'], 'page-2'))).mockReturnValueOnce(of(result(['collector-b'])));
  render(<App />);
  expect(await screen.findByRole('region', { name: 'Agent table' })).toBeInTheDocument();
  expect(screen.getAllByText('collector-a')).toHaveLength(2);
  fireEvent.click(screen.getByRole('button', { name: 'Next page' }));
  await waitFor(() => expect(mockFetch).toHaveBeenCalledTimes(2));
  expect(mockFetch.mock.calls[1][0].data.queries[0]).toMatchObject({
    cursor: 'page-2',
    datasource: { uid: 'commander' },
  });
  expect(screen.queryAllByText('collector-a')).toHaveLength(0);
  expect(screen.getAllByText('collector-b')).toHaveLength(2);
});

test('clears previous rows while refreshing and displays access failures', async () => {
  const refresh = new Subject();
  mockFetch.mockReturnValueOnce(of(result(['collector-a']))).mockReturnValueOnce(refresh);
  render(<App />);
  await screen.findByRole('region', { name: 'Agent table' });
  fireEvent.click(screen.getByRole('button', { name: 'Refresh' }));
  expect(screen.queryByRole('region', { name: 'Agent table' })).not.toBeInTheDocument();
  act(() => refresh.error({ data: { results: { A: { error: 'Token expired' } } } }));
  expect(await screen.findByText('Token expired')).toBeInTheDocument();
  expect(screen.queryByText('No agents found')).not.toBeInTheDocument();
});

test('distinguishes an authorized empty fleet from an error', async () => {
  mockFetch.mockReturnValue(of(result([])));
  render(<App />);
  expect(await screen.findByText('No agents found')).toBeInTheDocument();
  expect(screen.getByRole('button', { name: 'Next page' })).toBeDisabled();
});
