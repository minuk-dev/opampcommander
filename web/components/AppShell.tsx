'use client';

import { type ReactNode } from 'react';
import { usePathname } from 'next/navigation';
import { AuthProvider, useAuth } from './AuthProvider';
import { NamespaceProvider } from './NamespaceProvider';
import { PermissionsProvider } from './PermissionsProvider';
import AppLayout from './AppLayout';

const PUBLIC_ROUTES = new Set<string>(['/login', '/login/github/callback']);

function ShellInner({ children }: { children: ReactNode }) {
  const pathname = usePathname();
  const { authenticated } = useAuth();

  if (PUBLIC_ROUTES.has(pathname)) {
    return <>{children}</>;
  }

  if (!authenticated) {
    // AuthProvider will redirect to /login. Render nothing meanwhile.
    return null;
  }

  return (
    <NamespaceProvider>
      <PermissionsProvider>
        <AppLayout>{children}</AppLayout>
      </PermissionsProvider>
    </NamespaceProvider>
  );
}

export default function AppShell({ children }: { children: ReactNode }) {
  return (
    <AuthProvider>
      <ShellInner>{children}</ShellInner>
    </AuthProvider>
  );
}
