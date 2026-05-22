'use client';

import {
  AppBar,
  Box,
  Drawer,
  List,
  ListItem,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  ListSubheader,
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
  Menu as MenuIcon,
} from '@mui/icons-material';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { ReactNode, useEffect, useState } from 'react';
import { useAuth } from './AuthProvider';
import NamespaceSelector from './NamespaceSelector';
import VersionFooter from './VersionFooter';

const drawerWidth = 240;
const SIDEBAR_OPEN_KEY = 'opamp.sidebarOpen';

interface NavItem {
  text: string;
  icon: ReactNode;
  href: string;
}
interface NavSection {
  heading: string;
  items: NavItem[];
}

// Namespaces and Version are intentionally excluded:
//  - Namespace is a tenant context selected from the top bar.
//  - Version is shown in the bottom-left footer.
const sections: NavSection[] = [
  {
    heading: 'Overview',
    items: [
      { text: 'Dashboard', icon: <DashboardIcon />, href: '/' },
      { text: 'Servers', icon: <DnsIcon />, href: '/servers' },
    ],
  },
  {
    heading: 'Agents',
    items: [
      { text: 'Agents', icon: <ComputerIcon />, href: '/agents' },
      { text: 'Agent Groups', icon: <GroupIcon />, href: '/agentgroups' },
      { text: 'Connections', icon: <CableIcon />, href: '/connections' },
      { text: 'Agent Packages', icon: <PackageIcon />, href: '/agentpackages' },
      { text: 'Remote Configs', icon: <TuneIcon />, href: '/agentremoteconfigs' },
      { text: 'Certificates', icon: <CertIcon />, href: '/certificates' },
    ],
  },
  {
    heading: 'Access',
    items: [
      { text: 'Users', icon: <PeopleIcon />, href: '/users' },
      { text: 'Roles', icon: <RoleIcon />, href: '/roles' },
      { text: 'Role Bindings', icon: <RoleBindingIcon />, href: '/rolebindings' },
    ],
  },
];

export default function AppLayout({ children }: { children: ReactNode }) {
  const pathname = usePathname();
  const { email, logout } = useAuth();
  const [menuAnchor, setMenuAnchor] = useState<HTMLElement | null>(null);
  const [drawerOpen, setDrawerOpen] = useState(true);

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
          {sections.map((section) => (
            <List
              key={section.heading}
              dense
              subheader={
                <ListSubheader component="div" sx={{ bgcolor: 'transparent' }}>
                  {section.heading}
                </ListSubheader>
              }
            >
              {section.items.map((item) => (
                <ListItem key={item.text} disablePadding>
                  <ListItemButton
                    component={Link}
                    href={item.href}
                    selected={
                      item.href === '/'
                        ? pathname === '/'
                        : pathname.startsWith(item.href)
                    }
                  >
                    <ListItemIcon>{item.icon}</ListItemIcon>
                    <ListItemText primary={item.text} />
                  </ListItemButton>
                </ListItem>
              ))}
            </List>
          ))}
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
