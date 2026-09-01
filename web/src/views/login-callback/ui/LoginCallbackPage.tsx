'use client';

import { Alert, Card, CardContent, Spinner } from '@shared/ui';
import { useRouter, useSearchParams } from 'next/navigation';
import { Suspense, useEffect, useState } from 'react';
import { useAuth } from '@entities/session';

function CallbackInner() {
  const router = useRouter();
  const search = useSearchParams();
  const { applyTokens } = useAuth();
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const errParam = search.get('error');
    const errDesc = search.get('error_description');
    if (errParam) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setError(errDesc ? `${errParam}: ${errDesc}` : errParam);
      return;
    }
    const token = search.get('token');
    const refreshToken = search.get('refreshToken') ?? undefined;
    const expiresAt = search.get('expiresAt') ?? undefined;
    if (!token) {
      setError('Missing token in callback');
      return;
    }
    // Sanitize `from` to a same-origin internal path (and never /login).
    const rawFrom = search.get('from') || '/';
    const isInternal = rawFrom.startsWith('/') && !rawFrom.startsWith('//');
    const isLoginPath =
      rawFrom === '/login' || rawFrom.startsWith('/login/') || rawFrom.startsWith('/login?');
    const from = isInternal && !isLoginPath ? rawFrom : '/';
    // Wait for the session cookie to be set before navigating so middleware
    // doesn't bounce us back to /login.
    void applyTokens({ token, refreshToken, expiresAt }).then(() => router.replace(from));
  }, [applyTokens, router, search]);

  return (
    <div className="flex min-h-dvh items-center justify-center p-4">
      <Card className="w-full max-w-sm">
        <CardContent className="pt-4">
          {error ? (
            <Alert severity="error">{error}</Alert>
          ) : (
            <div className="flex items-center gap-3 text-sm">
              <Spinner className="size-5" />
              Completing GitHub sign-in…
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

export default function GithubCallbackPage() {
  return (
    <Suspense fallback={null}>
      <CallbackInner />
    </Suspense>
  );
}
