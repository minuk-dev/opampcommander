'use client';

import Link from 'next/link';
import type { ReactNode } from 'react';
import { useApi } from '@shared/api';
import { cn } from '@shared/lib';
import { TimeDisplay } from '@shared/preferences';
import {
  Alert,
  Badge,
  Card,
  CardContent,
  PageHeader,
  Separator,
  Spinner,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeaderCell,
  TableRow,
  TableWrap,
} from '@shared/ui';
import type { UserProfileResponse } from '@entities/user';

function Detail({ label, value, mono }: { label: string; value: ReactNode; mono?: boolean }) {
  return (
    <div>
      <p className="text-[10px] font-semibold tracking-wider text-muted-foreground uppercase">
        {label}
      </p>
      <div className={cn('text-sm', mono && 'font-mono')}>{value}</div>
    </div>
  );
}

export default function ProfilePage() {
  // Shares the /api/v1/users/me request with PermissionsProvider via SWR's
  // cache, so loading this page issues no extra fetch (guide 4.3).
  const {
    data: profile,
    error: fetchError,
    isLoading: loading,
  } = useApi<UserProfileResponse>('/api/v1/users/me');
  const error =
    fetchError instanceof Error ? fetchError.message : fetchError ? 'Failed to load profile' : null;

  if (loading) {
    return (
      <div className="mt-16 flex justify-center">
        <Spinner className="size-6" />
      </div>
    );
  }

  if (error || !profile?.user) {
    return (
      <div>
        <PageHeader title="My profile" />
        <Alert severity="error">{error ?? 'No profile data returned'}</Alert>
      </div>
    );
  }

  const { user, roles } = profile;
  const labelEntries = Object.entries(user.metadata.labels ?? {});

  return (
    <div>
      <PageHeader
        title="My profile"
        subtitle="The account you are currently signed in as, and the roles applied to you."
      />

      <Card className="mb-4">
        <CardContent className="pt-4">
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-5">
            <Detail label="Email" value={user.spec.email || '-'} mono />
            <Detail label="Username" value={user.spec.username || '-'} />
            <Detail
              label="Status"
              value={
                <Badge variant={user.spec.isActive ? 'success' : 'muted'}>
                  {user.spec.isActive ? 'active' : 'inactive'}
                </Badge>
              }
            />
            <Detail label="UID" value={user.metadata.uid || '(no DB record yet)'} mono />
            <Detail label="Created" value={<TimeDisplay value={user.metadata.createdAt} />} />
          </div>
          {labelEntries.length > 0 && (
            <>
              <Separator className="my-3" />
              <p className="mb-1 text-[10px] font-semibold tracking-wider text-muted-foreground uppercase">
                Labels
              </p>
              <div className="flex flex-wrap gap-1">
                {labelEntries.map(([k, v]) => (
                  <Badge key={k} variant="outline">
                    {k}: {v}
                  </Badge>
                ))}
              </div>
            </>
          )}
        </CardContent>
      </Card>

      <h2 className="mb-2 text-sm font-semibold">Roles applied to you</h2>
      {(!roles || roles.length === 0) && (
        <Alert severity="info">
          No roles are currently applied. Ask an administrator to create a role binding for you.
        </Alert>
      )}
      {roles && roles.length > 0 && (
        <TableWrap>
          <Table>
            <TableHead>
              <TableRow className="hover:bg-transparent">
                <TableHeaderCell>Role</TableHeaderCell>
                <TableHeaderCell>Source</TableHeaderCell>
                <TableHeaderCell>Permissions</TableHeaderCell>
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
                      <div className="flex items-center gap-2">
                        <Link href="/roles" className="font-medium hover:underline">
                          {role.spec.displayName || role.metadata.uid}
                        </Link>
                        {role.spec.isBuiltIn && <Badge variant="primary">built-in</Badge>}
                      </div>
                      {role.spec.description && (
                        <p className="text-xs text-muted-foreground">{role.spec.description}</p>
                      )}
                    </TableCell>
                    <TableCell>
                      {rb ? (
                        <>
                          <Link href="/rolebindings" className="hover:underline">
                            {rb.metadata.namespace}/{rb.metadata.name}
                          </Link>
                          <p className="text-xs text-muted-foreground">RoleBinding</p>
                        </>
                      ) : (
                        <span className="text-muted-foreground">
                          Auto-assigned (built-in default)
                        </span>
                      )}
                    </TableCell>
                    <TableCell>
                      {perms.length === 0 ? (
                        <span className="text-xs text-muted-foreground">(no permissions)</span>
                      ) : (
                        <div className="flex flex-wrap gap-1">
                          {perms.slice(0, 30).map((p) => (
                            <Badge key={p} variant="outline">
                              {p}
                            </Badge>
                          ))}
                          {perms.length > 30 && (
                            <Badge variant="muted">+{perms.length - 30} more</Badge>
                          )}
                        </div>
                      )}
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </TableWrap>
      )}
    </div>
  );
}
