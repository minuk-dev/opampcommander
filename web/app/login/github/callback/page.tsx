'use client';

import {
  Alert,
  Box,
  Card,
  CardContent,
  CircularProgress,
  Typography,
} from '@mui/material';
import { useRouter, useSearchParams } from 'next/navigation';
import { Suspense, useEffect, useState } from 'react';
import { useAuth } from '@/components/AuthProvider';

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
    applyTokens({ token, refreshToken, expiresAt });
    // Sanitize `from` to a same-origin internal path.
    const rawFrom = search.get('from') || '/';
    const from =
      rawFrom.startsWith('/') && !rawFrom.startsWith('//') ? rawFrom : '/';
    router.replace(from);
  }, [applyTokens, router, search]);

  return (
    <Box
      display="flex"
      minHeight="100vh"
      alignItems="center"
      justifyContent="center"
      p={2}
    >
      <Card sx={{ maxWidth: 400, width: '100%' }}>
        <CardContent>
          {error ? (
            <Alert severity="error">{error}</Alert>
          ) : (
            <Box display="flex" alignItems="center" gap={2}>
              <CircularProgress size={24} />
              <Typography>Completing GitHub sign-in…</Typography>
            </Box>
          )}
        </CardContent>
      </Card>
    </Box>
  );
}

export default function GithubCallbackPage() {
  return (
    <Suspense fallback={null}>
      <CallbackInner />
    </Suspense>
  );
}
