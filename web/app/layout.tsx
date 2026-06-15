import type { Metadata } from 'next';
import { AppRouterCacheProvider } from '@mui/material-nextjs/v16-appRouter';
import '@app/styles/globals.css';
import ThemeRegistry from '@app/providers/ThemeRegistry';
import AppShell from '@app/app-shell';
import { PreferencesProvider } from '@shared/preferences';

export const metadata: Metadata = {
  title: 'OpAMP Commander',
  description: 'OpAMP Commander Web Interface',
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <head>
        {/* Fonts served from the Google Fonts CDN. preconnect warms up the
            TCP+TLS connection to the font hosts before the stylesheet request. */}
        <link rel="preconnect" href="https://fonts.googleapis.com" />
        <link rel="preconnect" href="https://fonts.gstatic.com" crossOrigin="anonymous" />
        {/* App Router: a <link> in the root layout's <head> loads on every
            page, so the no-page-custom-font rule (aimed at the pages/ router)
            does not apply here. */}
        {/* eslint-disable-next-line @next/next/no-page-custom-font */}
        <link
          rel="stylesheet"
          href="https://fonts.googleapis.com/css2?family=Geist:wght@100..900&family=Geist+Mono:wght@100..900&display=swap"
        />
      </head>
      <body className="antialiased">
        {/* Emotion cache + useServerInsertedHTML so MUI styles render correctly
            during SSR/RSC and don't cause hydration mismatches. */}
        <AppRouterCacheProvider>
          <PreferencesProvider>
            <ThemeRegistry>
              <AppShell>{children}</AppShell>
            </ThemeRegistry>
          </PreferencesProvider>
        </AppRouterCacheProvider>
      </body>
    </html>
  );
}
