'use client';

import {
  AppBar,
  Box,
  Drawer,
  FormControlLabel,
  List,
  ListItem,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  ListSubheader,
  Stack,
  Switch,
  Toolbar,
  Typography,
  IconButton,
  Menu,
  MenuItem,
  Divider,
  Tooltip,
} from '@mui/material';
import {
  Dashboard as DashboardIcon,
  Group as GroupIcon,
  Computer as ComputerIcon,
  Cable as CableIcon,
  Dns as DnsIcon,
  Inventory2 as PackageIcon,
  Tune as TuneIcon,
  VerifiedUser as CertIcon,
  PeopleAlt as PeopleIcon,
  AdminPanelSettings as RoleIcon,
  Link as RoleBindingIcon,
  Logout as LogoutIcon,
  AccountCircle as AccountIcon,
  Badge as BadgeIcon,
  Menu as MenuIcon,
} from '@mui/icons-material';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { type ReactNode, useEffect, useState } from 'react';
import { useAuth } from './AuthProvider';
import NamespaceSelector from './NamespaceSelector';
import { usePermissions } from './PermissionsProvider';
import VersionFooter from './VersionFooter';

const drawerWidth = 240;
const SIDEBAR_OPEN_KEY = 'opamp.sidebarOpen';

interface NavItem {
  text: string;
  icon: ReactNode;
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
    items: [{ text: 'Dashboard', icon: <DashboardIcon />, href: '/' }],
  },
  {
    heading: 'Agents',
    items: [
      {
        text: 'Agent Groups',
        icon: <GroupIcon />,
        href: '/agentgroups',
        requires: { resource: 'agentgroup', action: 'LIST' },
      },
      {
        text: 'Agents',
        icon: <ComputerIcon />,
        href: '/agents',
        requires: { resource: 'agent', action: 'LIST' },
      },
      {
        text: 'Connections',
        icon: <CableIcon />,
        href: '/connections',
        requires: { resource: 'connection', action: 'LIST' },
      },
      {
        text: 'Agent Packages',
        icon: <PackageIcon />,
        href: '/agentpackages',
        requires: { resource: 'agentpackage', action: 'LIST' },
      },
      {
        text: 'Remote Configs',
        icon: <TuneIcon />,
        href: '/agentremoteconfigs',
        requires: { resource: 'agentremoteconfig', action: 'LIST' },
      },
      {
        text: 'Certificates',
        icon: <CertIcon />,
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
        icon: <PeopleIcon />,
        href: '/users',
        requires: { resource: 'user', action: 'LIST' },
      },
      {
        text: 'Roles',
        icon: <RoleIcon />,
        href: '/roles',
        requires: { resource: 'role', action: 'LIST' },
      },
      {
        text: 'Role Bindings',
        icon: <RoleBindingIcon />,
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
        icon: <DnsIcon />,
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
  const [menuAnchor, setMenuAnchor] = useState<HTMLElement | null>(null);
  const [drawerOpen, setDrawerOpen] = useState(true);

  // Filter nav items by RBAC. Items without `requires` (Dashboard) always show;
  // others need the resource:LIST permission unless the user has toggled
  // "Show restricted menus" on (e.g. for exploration). Sections with no
  // visible items are dropped entirely so we don't leave dangling headers.
  const visibleSections = sections
    .map((section) => ({
      ...section,
      items: section.items.filter(
        (item) => showAll || !item.requires || hasPermission(item.requires.resource, item.requires.action),
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
  useEffect(() => {
    if (typeof window === 'undefined') return;
    const stored = window.localStorage.getItem(SIDEBAR_OPEN_KEY);
    if (stored === null) return;
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setDrawerOpen(stored === '1');
  }, []);

  const toggleDrawer = () => {
    setDrawerOpen((prev) => {
      const next = !prev;
      if (typeof window !== 'undefined') {
        window.localStorage.setItem(SIDEBAR_OPEN_KEY, next ? '1' : '0');
      }
      return next;
    });
  };

  return (
    <Box sx={{ display: 'flex' }}>
      <AppBar position="fixed" sx={{ zIndex: (theme) => theme.zIndex.drawer + 1 }}>
        <Toolbar sx={{ gap: 2 }}>
          <IconButton
            color="inherit"
            aria-label="toggle navigation"
            onClick={toggleDrawer}
            edge="start"
            size="large"
          >
            <MenuIcon />
          </IconButton>
          <Typography variant="h6" noWrap component="div" sx={{ flexShrink: 0 }}>
            OpAMP Commander
          </Typography>
          <Box
            sx={{
              borderLeft: '1px solid rgba(255,255,255,0.3)',
              height: 32,
              alignSelf: 'center',
            }}
          />
          <NamespaceSelector />
          <Box sx={{ flexGrow: 1 }} />
          <Tooltip title={email || 'Account'}>
            <IconButton
              color="inherit"
              onClick={(e) => setMenuAnchor(e.currentTarget)}
              size="large"
            >
              <AccountIcon />
            </IconButton>
          </Tooltip>
          <Menu
            anchorEl={menuAnchor}
            open={Boolean(menuAnchor)}
            onClose={() => setMenuAnchor(null)}
          >
            <MenuItem disabled>
              <Typography variant="body2">{email || 'unknown'}</Typography>
            </MenuItem>
            <Divider />
            <MenuItem component={Link} href="/profile" onClick={() => setMenuAnchor(null)}>
              <ListItemIcon>
                <BadgeIcon fontSize="small" />
              </ListItemIcon>
              My profile
            </MenuItem>
            <MenuItem
              onClick={() => {
                setMenuAnchor(null);
                logout();
              }}
            >
              <ListItemIcon>
                <LogoutIcon fontSize="small" />
              </ListItemIcon>
              Sign out
            </MenuItem>
          </Menu>
        </Toolbar>
      </AppBar>
      <Drawer
        variant="persistent"
        open={drawerOpen}
        sx={{
          width: drawerOpen ? drawerWidth : 0,
          flexShrink: 0,
          [`& .MuiDrawer-paper`]: {
            width: drawerWidth,
            boxSizing: 'border-box',
            display: 'flex',
            flexDirection: 'column',
          },
        }}
      >
        <Toolbar />
        <Box sx={{ overflow: 'auto', flexGrow: 1 }}>
          {visibleSections.map((section) => (
            <List
              key={section.heading}
              dense
              subheader={
                <ListSubheader component="div" sx={{ bgcolor: 'transparent' }}>
                  {section.heading}
                </ListSubheader>
              }
            >
              {section.items.map((item) => {
                // Boundary-aware match: a subroute like /agents/123 still
                // highlights /agents, but /agentgroups never highlights
                // /agents even though they share a prefix.
                const isActive =
                  item.href === '/'
                    ? pathname === '/'
                    : pathname === item.href || pathname.startsWith(`${item.href}/`);
                return (
                  <ListItem key={item.text} disablePadding>
                    <ListItemButton component={Link} href={item.href} selected={isActive}>
                      <ListItemIcon>{item.icon}</ListItemIcon>
                      <ListItemText primary={item.text} />
                    </ListItemButton>
                  </ListItem>
                );
              })}
            </List>
          ))}
        </Box>
        <Box
          sx={{
            px: 2,
            py: 1,
            borderTop: '1px solid',
            borderColor: 'divider',
          }}
        >
          <Tooltip
            title={
              hiddenCount > 0
                ? `Reveal ${hiddenCount} menu item${hiddenCount === 1 ? '' : 's'} hidden because you lack LIST permission. Pages may still return 403 when opened.`
                : 'Reveal menu items hidden by RBAC. You currently have access to all items.'
            }
            placement="right"
          >
            <FormControlLabel
              control={
                <Switch
                  size="small"
                  checked={showAll}
                  onChange={(e) => setShowAll(e.target.checked)}
                />
              }
              label={
                <Stack direction="row" spacing={0.5} alignItems="baseline">
                  <Typography variant="caption">Show restricted menus</Typography>
                  {hiddenCount > 0 && !showAll && (
                    <Typography variant="caption" color="text.disabled">
                      ({hiddenCount})
                    </Typography>
                  )}
                </Stack>
              }
              sx={{ m: 0, '& .MuiFormControlLabel-label': { ml: 1 } }}
            />
          </Tooltip>
        </Box>
        <VersionFooter />
      </Drawer>
      <Box
        component="main"
        sx={{
          flexGrow: 1,
          p: 3,
          minWidth: 0,
          transition: (theme) =>
            theme.transitions.create('margin', {
              easing: theme.transitions.easing.sharp,
              duration: theme.transitions.duration.leavingScreen,
            }),
        }}
      >
        <Toolbar />
        {children}
      </Box>
    </Box>
  );
}
