'use client';

import { LogIn } from 'lucide-react';
import Image from 'next/image';
import { useRouter, useSearchParams } from 'next/navigation';
import { Suspense, useEffect, useState } from 'react';
import { useAuth, type OAuth2AuthCodeURLResponse } from '@entities/session';
import { api } from '@shared/api';
import { Alert, Button, Card, CardContent, Field, Input } from '@shared/ui';

// lucide dropped brand marks, so the GitHub logo lives here as inline SVG.
function GithubMark() {
  return (
    <svg viewBox="0 0 16 16" fill="currentColor" aria-hidden className="size-4">
      <path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82a7.4 7.4 0 0 1 2-.27c.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.01 8.01 0 0 0 16 8c0-4.42-3.58-8-8-8Z" />
    </svg>
  );
}

// Ask the browser's password manager to save these credentials. Uses the
// Credential Management API, which only exists in Chromium-based browsers; it's
// silently skipped elsewhere. Failures (e.g. the user declining) are ignored so
// they never block the post-login redirect.
async function storePasswordCredential(username: string, password: string): Promise<void> {
  if (typeof window === 'undefined') return;
  const PasswordCredentialCtor = (
    window as unknown as {
      PasswordCredential?: new (data: { id: string; password: string }) => Credential;
    }
  ).PasswordCredential;
  if (!PasswordCredentialCtor || !navigator.credentials?.store) return;
  try {
    const credential = new PasswordCredentialCtor({ id: username, password });
    await navigator.credentials.store(credential);
  } catch {
    // Ignore — saving credentials is best-effort.
  }
}

function LoginInner() {
  const router = useRouter();
  const search = useSearchParams();
  // Constrain `from` to same-origin internal paths so a malicious link like
  // `/login?from=javascript:alert(1)` or `/login?from=https://evil` can't
  // pivot the post-login redirect off-site. Also exclude /login itself so
  // we don't redirect back here after sign-in.
  const rawFrom = search.get('from') || '/';
  const isInternal = rawFrom.startsWith('/') && !rawFrom.startsWith('//');
  const isLoginPath =
    rawFrom === '/login' || rawFrom.startsWith('/login/') || rawFrom.startsWith('/login?');
  const from = isInternal && !isLoginPath ? rawFrom : '/';
  const { authenticated, loginBasic } = useAuth();

  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [githubBusy, setGithubBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (authenticated) router.replace(from);
  }, [authenticated, from, router]);

  const onBasicSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setSubmitting(true);
    try {
      await loginBasic(username, password);
      // This is an AJAX login with a client-side (SPA) redirect, so the browser's
      // heuristic for offering to save credentials — which keys off a real form
      // navigation — never fires. Explicitly hand the credentials to the browser's
      // password manager via the Credential Management API (Chromium-only; a no-op
      // elsewhere) so it can prompt to save them.
      await storePasswordCredential(username, password);
      router.replace(from);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Login failed');
    } finally {
      setSubmitting(false);
    }
  };

  const onGithubClick = async () => {
    setError(null);
    setGithubBusy(true);
    try {
      // The backend's loopback flow accepts http(s)://(127.0.0.1|::1|localhost):PORT
      // redirect_uri values; the browser then receives ?token=... back.
      const origin = window.location.origin;
      const redirectUri = `${origin}/login/github/callback?from=${encodeURIComponent(from)}`;
      const res = await api.get<OAuth2AuthCodeURLResponse>('/api/v1/auth/github/authcode', {
        query: { redirect_uri: redirectUri },
        noAuthRedirect: true,
      });
      window.location.assign(res.url);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'GitHub login failed');
      setGithubBusy(false);
    }
  };

  return (
    <div className="relative flex min-h-dvh items-center justify-center overflow-hidden bg-background p-4">
      {/* A soft brand wash rather than a flat slab of colour — it reads as depth
          without fighting the card in either theme. */}
      <div
        aria-hidden
        className="pointer-events-none absolute inset-0"
        style={{
          background:
            'radial-gradient(55rem 35rem at 50% -15%, color-mix(in oklch, var(--primary) 30%, transparent), transparent 70%)',
        }}
      />
      <Card className="relative w-full max-w-sm shadow-xl">
        <CardContent className="space-y-4 pt-6">
          <div className="text-center">
            <Image
              src="/logo.png"
              alt="OpAMP Commander"
              width={56}
              height={56}
              priority
              className="mx-auto mb-2"
            />
            <h1 className="text-lg font-semibold tracking-tight">OpAMP Commander</h1>
            <p className="text-sm text-muted-foreground">Sign in to continue</p>
          </div>

          {error && <Alert severity="error">{error}</Alert>}

          <form onSubmit={onBasicSubmit} className="space-y-3">
            <Field label="Username or email" required>
              {(field) => (
                <Input
                  {...field}
                  name="username"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  autoComplete="username"
                  autoFocus
                  required
                />
              )}
            </Field>
            <Field label="Password" required>
              {(field) => (
                <Input
                  {...field}
                  type="password"
                  name="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  autoComplete="current-password"
                  required
                />
              )}
            </Field>
            <Button
              type="submit"
              size="lg"
              className="w-full"
              disabled={submitting || !username || !password}
            >
              <LogIn aria-hidden />
              {submitting ? 'Signing in…' : 'Sign in'}
            </Button>
          </form>

          <div className="flex items-center gap-3 text-xs text-muted-foreground">
            <span className="h-px flex-1 bg-border" />
            or
            <span className="h-px flex-1 bg-border" />
          </div>

          <Button
            variant="outline"
            size="lg"
            className="w-full"
            onClick={() => void onGithubClick()}
            disabled={githubBusy}
          >
            <GithubMark />
            {githubBusy ? 'Redirecting…' : 'Continue with GitHub'}
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}

export default function LoginPage() {
  return (
    <Suspense fallback={null}>
      <LoginInner />
    </Suspense>
  );
}
