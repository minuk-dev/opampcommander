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
  Folder as FolderIcon,
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
  Info as InfoIcon,
} from '@mui/icons-material';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { ReactNode, useState } from 'react';
import { useAuth } from './AuthProvider';
import NamespaceSelector from './NamespaceSelector';

const drawerWidth = 240;

interface NavItem {
  text: string;
  icon: ReactNode;
  href: string;
}
interface NavSection {
  heading: string;
  items: NavItem[];
}

const sections: NavSection[] = [
  {
    heading: 'Overview',
    items: [
      { text: 'Dashboard', icon: <DashboardIcon />, href: '/' },
      { text: 'Namespaces', icon: <FolderIcon />, href: '/namespaces' },
      { text: 'Servers', icon: <DnsIcon />, href: '/servers' },
      { text: 'Version', icon: <InfoIcon />, href: '/version' },
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

  return (
    <Box sx={{ display: 'flex' }}>
      <AppBar position="fixed" sx={{ zIndex: (theme) => theme.zIndex.drawer + 1 }}>
        <Toolbar sx={{ gap: 2 }}>
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
        variant="permanent"
        sx={{
          width: drawerWidth,
          flexShrink: 0,
          [`& .MuiDrawer-paper`]: { width: drawerWidth, boxSizing: 'border-box' },
        }}
      >
        <Toolbar />
        <Box sx={{ overflow: 'auto' }}>
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
      </Drawer>
      <Box component="main" sx={{ flexGrow: 1, p: 3, minWidth: 0 }}>
        <Toolbar />
        {children}
      </Box>
    </Box>
  );
}
