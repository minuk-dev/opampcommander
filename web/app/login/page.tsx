'use client';

import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Divider,
  Stack,
  TextField,
  Typography,
} from '@mui/material';
import { GitHub as GitHubIcon, Login as LoginIcon } from '@mui/icons-material';
import { useRouter, useSearchParams } from 'next/navigation';
import { Suspense, useEffect, useState } from 'react';
import { useAuth } from '@/components/AuthProvider';
import { api } from '@/lib/api-client';
import type { OAuth2AuthCodeURLResponse } from '@/lib/types';

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
    <Box
      display="flex"
      minHeight="100vh"
      alignItems="center"
      justifyContent="center"
      sx={{
        background: 'linear-gradient(135deg, #1a237e 0%, #283593 60%, #1976d2 100%)',
        p: 2,
      }}
    >
      <Card sx={{ maxWidth: 420, width: '100%' }}>
        <CardContent>
          <Stack spacing={2}>
            <Box textAlign="center">
              <Typography variant="h5" component="h1" gutterBottom>
                OpAMP Commander
              </Typography>
              <Typography variant="body2" color="text.secondary">
                Sign in to continue
              </Typography>
            </Box>

            {error && <Alert severity="error">{error}</Alert>}

            <form onSubmit={onBasicSubmit}>
              <Stack spacing={2}>
                <TextField
                  label="Username or email"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  autoComplete="username"
                  autoFocus
                  fullWidth
                  required
                />
                <TextField
                  label="Password"
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  autoComplete="current-password"
                  fullWidth
                  required
                />
                <Button
                  type="submit"
                  variant="contained"
                  startIcon={<LoginIcon />}
                  disabled={submitting || !username || !password}
                  size="large"
                >
                  {submitting ? 'Signing in…' : 'Sign in'}
                </Button>
              </Stack>
            </form>

            <Divider>or</Divider>

            <Button
              variant="outlined"
              startIcon={<GitHubIcon />}
              size="large"
              onClick={onGithubClick}
              disabled={githubBusy}
            >
              {githubBusy ? 'Redirecting…' : 'Continue with GitHub'}
            </Button>
          </Stack>
        </CardContent>
      </Card>
    </Box>
  );
}

export default function LoginPage() {
  return (
    <Suspense fallback={null}>
      <LoginInner />
    </Suspense>
  );
}
