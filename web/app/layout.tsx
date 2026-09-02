import type { Metadata } from 'next';
import '@app/styles/globals.css';
import AppShell from '@app/app-shell';
import { ToastProvider, TooltipProvider } from '@shared/ui';
import { PreferencesProvider, ThemeScript } from '@shared/preferences';

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
    <html lang="en" suppressHydrationWarning>
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
        {/* Applies the stored theme before first paint (no white flash). */}
        <ThemeScript />
      </head>
      <body>
        <PreferencesProvider>
          <TooltipProvider delayDuration={300}>
            <ToastProvider>
              <AppShell>{children}</AppShell>
            </ToastProvider>
          </TooltipProvider>
        </PreferencesProvider>
      </body>
    </html>
  );
}
