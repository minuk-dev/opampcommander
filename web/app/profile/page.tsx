'use client';

import {
  Alert,
  Box,
  Card,
  CardContent,
  Chip,
  CircularProgress,
  Divider,
  Paper,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
} from '@mui/material';
import Link from 'next/link';
import { useEffect, useState } from 'react';
import PageHeader from '@/components/PageHeader';
import { api } from '@/lib/api-client';
import type { UserProfileResponse } from '@/lib/types';

export default function ProfilePage() {
  const [profile, setProfile] = useState<UserProfileResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    api
      .get<UserProfileResponse>('/api/v1/users/me')
      .then((p) => {
        if (!cancelled) setProfile(p);
      })
      .catch((err) => {
        if (!cancelled)
          setError(err instanceof Error ? err.message : 'Failed to load profile');
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" mt={6}>
        <CircularProgress />
      </Box>
    );
  }

  if (error || !profile) {
    return (
      <Box>
        <PageHeader title="My profile" />
        <Alert severity="error">{error ?? 'No profile data returned'}</Alert>
      </Box>
    );
  }

  const { user, roles } = profile;
  const labelEntries = Object.entries(user.metadata.labels ?? {});

  return (
    <Box>
      <PageHeader
        title="My profile"
        subtitle="The account you are currently signed in as, and the roles applied to you."
      />

      <Card variant="outlined" sx={{ mb: 3 }}>
        <CardContent>
          <Stack direction={{ xs: 'column', md: 'row' }} gap={4} flexWrap="wrap">
            <Field label="Email" value={user.spec.email || '-'} mono />
            <Field label="Username" value={user.spec.username || '-'} />
            <Field
              label="Status"
              value={
                <Chip
                  size="small"
                  label={user.spec.isActive ? 'active' : 'inactive'}
                  color={user.spec.isActive ? 'success' : 'default'}
                />
              }
            />
            <Field label="UID" value={user.metadata.uid || '(no DB record yet)'} mono />
            <Field label="Created" value={user.metadata.createdAt || '-'} />
          </Stack>
          {labelEntries.length > 0 && (
            <>
              <Divider sx={{ my: 2 }} />
              <Typography variant="overline" color="text.secondary">
                Labels
              </Typography>
              <Stack direction="row" gap={0.5} flexWrap="wrap" mt={0.5}>
                {labelEntries.map(([k, v]) => (
                  <Chip key={k} label={`${k}: ${v}`} size="small" variant="outlined" />
                ))}
              </Stack>
            </>
          )}
        </CardContent>
      </Card>

      <Typography variant="h6" gutterBottom>
        Roles applied to you
      </Typography>
      {(!roles || roles.length === 0) && (
        <Alert severity="info">
          No roles are currently applied. Ask an administrator to create a role binding for you.
        </Alert>
      )}
      {roles && roles.length > 0 && (
        <TableContainer component={Paper} variant="outlined">
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>Role</TableCell>
                <TableCell>Source</TableCell>
                <TableCell>Permissions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {roles.map((entry, i) => {
                const role = entry.role;
                const rb = entry.roleBinding;
                const perms = role.spec.permissions ?? [];
                return (
                  <TableRow key={`${role.metadata.uid}-${i}`}>
                    <TableCell>
                      <Stack direction="row" alignItems="center" gap={1}>
                        <Link
                          href="/roles"
                          style={{ color: 'inherit', textDecoration: 'underline' }}
                        >
                          {role.spec.displayName || role.metadata.uid}
                        </Link>
                        {role.spec.isBuiltIn && (
                          <Chip label="built-in" size="small" color="info" variant="outlined" />
                        )}
                      </Stack>
                      {role.spec.description && (
                        <Typography variant="caption" color="text.secondary">
                          {role.spec.description}
                        </Typography>
                      )}
                    </TableCell>
                    <TableCell>
                      {rb ? (
                        <Stack>
                          <Link href="/rolebindings" style={{ color: 'inherit' }}>
                            {rb.metadata.namespace}/{rb.metadata.name}
                          </Link>
                          <Typography variant="caption" color="text.secondary">
                            RoleBinding
                          </Typography>
                        </Stack>
                      ) : (
                        <Typography variant="body2" color="text.secondary">
                          Auto-assigned (built-in default)
                        </Typography>
                      )}
                    </TableCell>
                    <TableCell>
                      {perms.length === 0 ? (
                        <Typography variant="caption" color="text.secondary">
                          (no permissions)
                        </Typography>
                      ) : (
                        <Stack direction="row" gap={0.5} flexWrap="wrap">
                          {perms.slice(0, 30).map((p) => (
                            <Chip key={p} label={p} size="small" variant="outlined" />
                          ))}
                          {perms.length > 30 && (
                            <Chip label={`+${perms.length - 30} more`} size="small" />
                          )}
                        </Stack>
                      )}
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </TableContainer>
      )}
    </Box>
  );
}

function Field({
  label,
  value,
  mono,
}: {
  label: string;
  value: React.ReactNode;
  mono?: boolean;
}) {
  return (
    <Box>
      <Typography variant="overline" color="text.secondary">
        {label}
      </Typography>
      <Typography
        variant="body1"
        sx={mono ? { fontFamily: 'var(--font-geist-mono), monospace', fontSize: 14 } : undefined}
      >
        {value}
      </Typography>
    </Box>
  );
}
