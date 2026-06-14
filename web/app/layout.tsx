import type { Metadata } from 'next';
import { Geist, Geist_Mono } from 'next/font/google';
import { AppRouterCacheProvider } from '@mui/material-nextjs/v16-appRouter';
import '@app/styles/globals.css';
import ThemeRegistry from '@app/providers/ThemeRegistry';
import AppShell from '@app/app-shell';
import { PreferencesProvider } from '@shared/preferences';

const geistSans = Geist({
  variable: '--font-geist-sans',
  subsets: ['latin'],
});

const geistMono = Geist_Mono({
  variable: '--font-geist-mono',
  subsets: ['latin'],
});

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
      <body className={`${geistSans.variable} ${geistMono.variable} antialiased`}>
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
