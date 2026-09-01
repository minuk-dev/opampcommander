'use client';

import {
  Boxes,
  Cable,
  CircleUser,
  Container,
  Cpu,
  Gauge,
  IdCard,
  KeyRound,
  Layers,
  Link2,
  LogOut,
  Menu as MenuIcon,
  Package,
  Server,
  Settings,
  ShieldCheck,
  SlidersHorizontal,
  Users,
} from 'lucide-react';
import Image from 'next/image';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { type ComponentType, type ReactNode, useEffect, useState } from 'react';
import { useAuth, usePermissions } from '@entities/session';
import { NamespaceSelector } from '@features/namespace-select';
import { ThemeToggle, TimezoneButton } from '@shared/preferences';
import { cn } from '@shared/lib';
import {
  Button,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
  Sheet,
  SheetContent,
  SheetTitle,
  Switch,
  Tooltip,
} from '@shared/ui';
import { VersionFooter } from '@widgets/version-footer';

const SIDEBAR_OPEN_KEY = 'opamp.sidebarOpen';

interface NavItem {
  text: string;
  icon: ComponentType<{ className?: string; 'aria-hidden'?: boolean }>;
  href: string;
  // RBAC requirement to make this item visible. Items with no `requires`
  // (e.g. Dashboard) are always shown.
  requires?: { resource: string; action: string };
}
interface NavSection {
  heading: string;
  items: NavItem[];
}

// Sidebar ordering reflects domain importance:
//  - Agent domain first (AgentGroup before Agent — groups are the unit of intent).
//  - Access (user/role management).
//  - Admin (cluster-level operator views) at the bottom.
// Namespaces / Version excluded — namespace is a top-bar tenant selector,
// version is shown in the bottom-left footer.
const sections: NavSection[] = [
  {
    heading: 'Overview',
    items: [{ text: 'Dashboard', icon: Gauge, href: '/' }],
  },
  {
    heading: 'Agents',
    items: [
      {
        text: 'Agent Groups',
        icon: Boxes,
        href: '/agentgroups',
        requires: { resource: 'agentgroup', action: 'LIST' },
      },
      {
        text: 'Agents',
        icon: Cpu,
        href: '/agents',
        requires: { resource: 'agent', action: 'LIST' },
      },
      {
        text: 'Connections',
        icon: Cable,
        href: '/connections',
        requires: { resource: 'connection', action: 'LIST' },
      },
      {
        // Platform (hosts/containers) is discovered, not RBAC-scoped per
        // resource, so it has no `requires` and is shown to any authenticated user.
        text: 'Platform',
        icon: Layers,
        href: '/platform',
      },
      {
        text: 'Agent Packages',
        icon: Package,
        href: '/agentpackages',
        requires: { resource: 'agentpackage', action: 'LIST' },
      },
      {
        text: 'Remote Configs',
        icon: SlidersHorizontal,
        href: '/agentremoteconfigs',
        requires: { resource: 'agentremoteconfig', action: 'LIST' },
      },
      {
        text: 'Endpoints',
        icon: Container,
        href: '/endpoints',
        requires: { resource: 'endpoint', action: 'LIST' },
      },
      {
        text: 'Certificates',
        icon: ShieldCheck,
        href: '/certificates',
        requires: { resource: 'certificate', action: 'LIST' },
      },
    ],
  },
  {
    heading: 'Access',
    items: [
      {
        text: 'Users',
        icon: Users,
        href: '/users',
        requires: { resource: 'user', action: 'LIST' },
      },
      {
        text: 'Roles',
        icon: KeyRound,
        href: '/roles',
        requires: { resource: 'role', action: 'LIST' },
      },
      {
        text: 'Role Bindings',
        icon: Link2,
        href: '/rolebindings',
        requires: { resource: 'rolebinding', action: 'LIST' },
      },
    ],
  },
  {
    heading: 'Admin',
    items: [
      {
        text: 'Servers',
        icon: Server,
        href: '/servers',
        requires: { resource: 'server', action: 'LIST' },
      },
    ],
  },
];

export default function AppLayout({ children }: { children: ReactNode }) {
  const pathname = usePathname();
  const { email, logout } = useAuth();
  const { hasPermission, showAll, setShowAll } = usePermissions();
  const [desktopOpen, setDesktopOpen] = useState(true);
  const [mobileOpen, setMobileOpen] = useState(false);

  // Filter nav items by RBAC. Items without `requires` (Dashboard) always show;
  // others need the resource:LIST permission unless the user has toggled
  // "Show restricted menus" on (e.g. for exploration). Sections with no
  // visible items are dropped entirely so we don't leave dangling headers.
  const visibleSections = sections
    .map((section) => ({
      ...section,
      items: section.items.filter(
        (item) =>
          showAll || !item.requires || hasPermission(item.requires.resource, item.requires.action),
      ),
    }))
    .filter((section) => section.items.length > 0);

  const hiddenCount = sections.reduce(
    (sum, s) =>
      sum +
      s.items.filter(
        (item) => item.requires && !hasPermission(item.requires.resource, item.requires.action),
      ).length,
    0,
  );

  // Hydrate persisted sidebar state after mount (avoids SSR/CSR mismatch).
  // localStorage can throw in private browsing / when disabled — guard it.
  useEffect(() => {
    if (typeof window === 'undefined') return;
    try {
      const stored = window.localStorage.getItem(SIDEBAR_OPEN_KEY);
      if (stored === null) return;
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setDesktopOpen(stored === '1');
    } catch {
      // Keep the default open state.
    }
  }, []);

  // One button drives both modes: below `md` it opens the overlay sheet, above
  // it collapses the persistent rail. Which one the user sees is decided purely
  // by CSS breakpoints, so there is no pre-hydration flash either way.
  const toggleDrawer = () => {
    setMobileOpen((prev) => !prev);
    setDesktopOpen((prev) => {
      const next = !prev;
      try {
        window.localStorage.setItem(SIDEBAR_OPEN_KEY, next ? '1' : '0');
      } catch {
        // Best-effort: preference just won't persist.
      }
      return next;
    });
  };

  const nav = (
    <>
      <nav className="min-h-0 flex-1 overflow-y-auto px-2 py-2">
        {visibleSections.map((section) => (
          <div key={section.heading} className="mb-3 last:mb-0">
            <p className="px-2 pb-1 text-[10px] font-semibold tracking-wider text-muted-foreground uppercase">
              {section.heading}
            </p>
            <ul className="space-y-0.5">
              {section.items.map((item) => {
                // Boundary-aware match: a subroute like /agents/123 still
                // highlights /agents, but /agentgroups never highlights
                // /agents even though they share a prefix.
                const isActive =
                  item.href === '/'
                    ? pathname === '/'
                    : pathname === item.href || pathname.startsWith(`${item.href}/`);
                const Icon = item.icon;
                return (
                  <li key={item.text}>
                    <Link
                      href={item.href}
                      onClick={() => setMobileOpen(false)}
                      aria-current={isActive ? 'page' : undefined}
                      className={cn(
                        'flex h-8 items-center gap-2.5 rounded-md px-2 text-sm transition-colors',
                        isActive
                          ? 'bg-primary/12 font-medium text-primary'
                          : 'text-muted-foreground hover:bg-accent hover:text-foreground',
                      )}
                    >
                      <Icon className="size-4 shrink-0" aria-hidden />
                      <span className="truncate">{item.text}</span>
                    </Link>
                  </li>
                );
              })}
            </ul>
          </div>
        ))}
      </nav>
      <Tooltip
        side="right"
        content={
          hiddenCount > 0
            ? `Reveal ${hiddenCount} menu item${hiddenCount === 1 ? '' : 's'} hidden because you lack LIST permission. Pages may still return 403 when opened.`
            : 'Reveal menu items hidden by RBAC. You currently have access to all items.'
        }
      >
        <label className="flex cursor-pointer items-center gap-2 border-t border-border px-3 py-2 text-xs text-muted-foreground">
          <Switch checked={showAll} onCheckedChange={setShowAll} />
          <span>Show restricted menus</span>
          {hiddenCount > 0 && !showAll && <span className="tnum">({hiddenCount})</span>}
        </label>
      </Tooltip>
      <VersionFooter />
    </>
  );

  return (
    <div className="flex min-h-dvh flex-col">
      <header className="sticky top-0 z-30 flex h-12 shrink-0 items-center gap-2 border-b border-border bg-card/95 px-2 backdrop-blur sm:px-3">
        <Button
          variant="ghost"
          size="icon-sm"
          aria-label="toggle navigation"
          onClick={toggleDrawer}
        >
          <MenuIcon aria-hidden />
        </Button>
        <Link href="/" className="flex shrink-0 items-center gap-2">
          <Image src="/logo.png" alt="OpAMP Commander" width={24} height={24} priority />
          <span className="hidden text-sm font-semibold tracking-tight sm:block">
            OpAMP Commander
          </span>
        </Link>
        <div className="mx-1 hidden h-5 w-px bg-border sm:block" />
        <NamespaceSelector />
        <div className="flex-1" />
        <TimezoneButton />
        <ThemeToggle />
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="icon-sm" aria-label={email || 'Account'}>
              <CircleUser aria-hidden />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuLabel className="truncate">{email || 'unknown'}</DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuItem asChild>
              <Link href="/profile">
                <IdCard aria-hidden />
                My profile
              </Link>
            </DropdownMenuItem>
            <DropdownMenuItem asChild>
              <Link href="/preferences">
                <Settings aria-hidden />
                Preferences
              </Link>
            </DropdownMenuItem>
            <DropdownMenuItem onSelect={logout}>
              <LogOut aria-hidden />
              Sign out
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </header>

      <div className="flex min-h-0 flex-1">
        {/* Desktop: a persistent rail that reserves width only when open. */}
        <aside
          className={cn(
            'sticky top-12 hidden h-[calc(100dvh-3rem)] shrink-0 flex-col border-r border-border bg-card/50 transition-[width] md:flex',
            desktopOpen ? 'w-56' : 'w-0 overflow-hidden border-r-0',
          )}
        >
          {desktopOpen && nav}
        </aside>

        {/* Mobile: the same nav as an overlay sheet. */}
        <Sheet open={mobileOpen} onOpenChange={setMobileOpen}>
          <SheetContent className="md:hidden">
            <SheetTitle className="flex h-12 shrink-0 items-center gap-2 border-b border-border px-3 text-sm font-semibold">
              <Image src="/logo.png" alt="" width={20} height={20} />
              OpAMP Commander
            </SheetTitle>
            {nav}
          </SheetContent>
        </Sheet>

        <main className="min-w-0 flex-1 p-3 sm:p-5">{children}</main>
      </div>
    </div>
  );
}
