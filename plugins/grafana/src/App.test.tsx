import React from 'react';
import { render, screen } from '@testing-library/react';
import { App } from './App';

jest.mock('@grafana/runtime', () => ({
  PluginPage: ({ children }: React.PropsWithChildren) => <main>{children}</main>,
}));

test('explains that the preview does not report fleet status', () => {
  render(<App />);
  expect(screen.getByRole('heading', { name: 'Fleet overview' })).toBeInTheDocument();
  expect(screen.getByText(/Fleet queries are not available/)).toBeInTheDocument();
});
