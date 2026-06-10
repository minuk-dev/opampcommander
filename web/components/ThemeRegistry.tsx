'use client';

import { ThemeProvider, createTheme } from '@mui/material/styles';
import CssBaseline from '@mui/material/CssBaseline';
import { type ReactNode } from 'react';

const theme = createTheme({
  palette: {
    mode: 'light',
    primary: {
      main: '#1976d2',
    },
    secondary: {
      main: '#dc004e',
    },
  },
  components: {
    // Shrink the root font size so all rem-based typography scales down
    // uniformly (16px → 14px). Layout spacing is px-based and stays put, so
    // only the text gets smaller — a global, even reduction.
    MuiCssBaseline: {
      styleOverrides: {
        html: { fontSize: '87.5%' },
      },
    },
  },
});

export default function ThemeRegistry({ children }: { children: ReactNode }) {
  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      {children}
    </ThemeProvider>
  );
}
