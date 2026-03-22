import { Link, Outlet, useLocation, useNavigate } from 'react-router-dom';
import { useState, useEffect, useRef, useCallback } from 'react';
import {
  LayoutDashboard, Users, Settings, LogOut, Shield, Bell, CreditCard, Zap,
  Sun, Moon, Monitor, Megaphone, ChevronDown, Search, Menu, X,
  CalendarDays, Package, Tag, Warehouse, Box, Globe, Anchor, Ship,
  ShoppingCart, UserCheck, BarChart3, Layers, Building2, ChevronRight,
} from 'lucide-react';
import { useAuth } from '../contexts/AuthContext';
import { useTenant } from '../contexts/TenantContext';
import { useBranding } from '../contexts/BrandingContext';
import { useTheme } from '../contexts/ThemeContext';
import { messagesApi, plansApi, bundlesApi, announcementsApi } from '../api/client';
import ImpersonationBanner from './ImpersonationBanner';
import CommandPalette from './CommandPalette';
import type { LucideIcon } from 'lucide-react';

type ThemeMode = 'dark' | 'light' | 'system';

// ─── Navigation config ────────────────────────────────────────────────────────

type NavItem = { path: string; label: string; icon: LucideIcon; exact?: boolean };
type NavGroup = { label: string; items: NavItem[] };

const procurementGroups: NavGroup[] = [
  {
    label: 'Einkauf',
    items: [
      { path: '/procurement', label: 'Übersicht', icon: BarChart3, exact: true },
      { path: '/procurement/calendar', label: 'Kalender', icon: CalendarDays },
      { path: '/procurement/orders', label: 'Bestellungen', icon: ShoppingCart },
      { path: '/procurement/offers', label: 'Angebote', icon: Tag },
      { path: '/procurement/inventory', label: 'Lager', icon: Warehouse },
    ],
  },
  {
    label: 'Stammdaten',
    items: [
      { path: '/procurement/products', label: 'Produkte', icon: Package },
      { path: '/procurement/goods-groups', label: 'Warengruppen', icon: Layers },
      { path: '/procurement/suppliers', label: 'Lieferanten', icon: Building2 },
      { path: '/procurement/customers', label: 'Kunden', icon: UserCheck },
    ],
  },
  {
    label: 'Logistik',
    items: [
      { path: '/procurement/harbours', label: 'Häfen', icon: Anchor },
      { path: '/procurement/containers', label: 'Container', icon: Box },
      { path: '/procurement/freight-carriers', label: 'Frachtführer', icon: Ship },
      { path: '/procurement/countries', label: 'Länder', icon: Globe },
    ],
  },
];

const BREADCRUMB_MAP: Record<string, { label: string; section?: string }> = {
  '/dashboard': { label: 'Dashboard' },
  '/procurement': { label: 'Übersicht', section: 'Einkauf' },
  '/procurement/calendar': { label: 'Kalender', section: 'Einkauf' },
  '/procurement/orders': { label: 'Bestellungen', section: 'Einkauf' },
  '/procurement/offers': { label: 'Angebote', section: 'Einkauf' },
  '/procurement/inventory': { label: 'Lager', section: 'Einkauf' },
  '/procurement/products': { label: 'Produkte', section: 'Stammdaten' },
  '/procurement/goods-groups': { label: 'Warengruppen', section: 'Stammdaten' },
  '/procurement/suppliers': { label: 'Lieferanten', section: 'Stammdaten' },
  '/procurement/customers': { label: 'Kunden', section: 'Stammdaten' },
  '/procurement/harbours': { label: 'Häfen', section: 'Logistik' },
  '/procurement/containers': { label: 'Container', section: 'Logistik' },
  '/procurement/freight-carriers': { label: 'Frachtführer', section: 'Logistik' },
  '/procurement/countries': { label: 'Länder', section: 'Logistik' },
  '/procurement/config': { label: 'Konfiguration', section: 'Einkauf' },
  '/team': { label: 'Team' },
  '/plan': { label: 'Plan' },
  '/settings': { label: 'Einstellungen' },
  '/messages': { label: 'Nachrichten' },
  '/activity': { label: 'Aktivität' },
};

// ─── Sidebar nav link ─────────────────────────────────────────────────────────

function SidebarLink({ item, collapsed }: { item: NavItem; collapsed: boolean }) {
  const location = useLocation();
  const active = item.exact
    ? location.pathname === item.path
    : location.pathname === item.path || location.pathname.startsWith(item.path + '/');

  return (
    <Link
      to={item.path}
      title={collapsed ? item.label : undefined}
      className={`flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors mb-0.5 ${
        active
          ? 'bg-primary-500/15 text-primary-400'
          : 'text-dark-400 hover:text-dark-100 hover:bg-dark-800/60'
      }`}
    >
      <item.icon className={`w-4 h-4 flex-shrink-0 ${active ? 'text-primary-400' : ''}`} />
      {!collapsed && <span className="truncate">{item.label}</span>}
    </Link>
  );
}

// ─── Sidebar inner content (shared between desktop + mobile) ──────────────────

interface SidebarContentProps {
  collapsed: boolean;
  onCollapse: () => void;
  showTeam: boolean;
  showAdmin: boolean;
  cmdOpen: () => void;
  theme: ThemeMode;
  resolvedTheme: 'dark' | 'light';
  onCycleTheme: () => void;
  user: { displayName?: string; email?: string } | null;
  onLogout: () => void;
  branding: { appName?: string; logoUrl?: string; logoMode?: string };
  showCredits: boolean;
  tenantCredits: number;
  hasBundles: boolean;
  mobileClose?: () => void;
}

function SidebarContent({
  collapsed, onCollapse, showTeam, showAdmin, cmdOpen,
  theme, onCycleTheme, user, onLogout, branding,
  showCredits, tenantCredits, hasBundles, mobileClose,
}: SidebarContentProps) {
  const navigate = useNavigate();

  const appName = branding.appName || 'LastSaaS';
  const logoMode = branding.logoMode || 'text';
  const logoUrl = branding.logoUrl;

  const initials = user?.displayName
    ? user.displayName.split(' ').map(p => p[0]).slice(0, 2).join('').toUpperCase()
    : '?';

  const ThemeIcon = theme === 'dark' ? Moon : theme === 'light' ? Sun : Monitor;
  const themeLabel = theme === 'dark' ? 'Dunkel' : theme === 'light' ? 'Hell' : 'System';

  const workspaceItems: NavItem[] = [
    ...(showTeam ? [{ path: '/team', label: 'Team', icon: Users }] : []),
    { path: '/plan', label: 'Plan', icon: CreditCard },
    { path: '/settings', label: 'Einstellungen', icon: Settings },
  ];

  const handleClose = mobileClose ?? onCollapse;

  return (
    <>
      {/* Logo + collapse toggle */}
      <div className={`flex items-center flex-shrink-0 border-b border-dark-800 h-14 ${collapsed ? 'justify-center px-2' : 'justify-between px-4'}`}>
        {collapsed ? (
          <button
            onClick={onCollapse}
            title="Seitenleiste aufklappen"
            className="w-8 h-8 rounded-lg bg-gradient-to-br from-primary-500 to-primary-700 flex items-center justify-center text-white font-bold text-xs flex-shrink-0"
          >
            {appName.slice(0, 2).toUpperCase()}
          </button>
        ) : (
          <>
            <Link
              to="/dashboard"
              onClick={mobileClose}
              className="flex items-center gap-2.5 min-w-0"
            >
              {(logoMode === 'image' || logoMode === 'both') && logoUrl ? (
                <img src={logoUrl} alt={appName} className="h-7 w-7 rounded-lg object-contain flex-shrink-0" />
              ) : (
                <div className="w-7 h-7 rounded-lg bg-gradient-to-br from-primary-500 to-primary-700 flex items-center justify-center flex-shrink-0">
                  <span className="text-white font-bold text-xs">{appName.slice(0, 2).toUpperCase()}</span>
                </div>
              )}
              {(logoMode === 'text' || logoMode === 'both') && (
                <span className="font-semibold text-dark-100 text-sm truncate">{appName}</span>
              )}
            </Link>
            <button
              onClick={mobileClose ? handleClose : onCollapse}
              title="Seitenleiste einklappen"
              className="p-1.5 text-dark-500 hover:text-dark-300 hover:bg-dark-800 rounded-lg transition-colors flex-shrink-0"
            >
              {mobileClose ? <X className="w-4 h-4" /> : <ChevronRight className="w-4 h-4 rotate-180" />}
            </button>
          </>
        )}
      </div>

      {/* Search / Cmd+K trigger */}
      {!collapsed && (
        <div className="px-3 pt-3 pb-1 flex-shrink-0">
          <button
            onClick={cmdOpen}
            className="w-full flex items-center gap-2 px-3 py-2 rounded-lg bg-dark-800/60 border border-dark-700/50 text-dark-500 text-sm hover:text-dark-300 hover:border-dark-600 transition-colors"
          >
            <Search className="w-3.5 h-3.5 flex-shrink-0" />
            <span className="flex-1 text-left text-xs">Suchen…</span>
            <kbd className="text-xs bg-dark-700 px-1.5 py-0.5 rounded font-mono leading-none">⌘K</kbd>
          </button>
        </div>
      )}
      {collapsed && (
        <div className="px-2 pt-3 pb-1 flex-shrink-0">
          <button
            onClick={cmdOpen}
            title="Suchen (⌘K)"
            className="w-full flex items-center justify-center p-2 rounded-lg text-dark-500 hover:text-dark-300 hover:bg-dark-800/60 transition-colors"
          >
            <Search className="w-4 h-4" />
          </button>
        </div>
      )}

      {/* Nav */}
      <nav className="flex-1 overflow-y-auto px-2 py-1 space-y-0.5">
        {/* Dashboard */}
        <SidebarLink item={{ path: '/dashboard', label: 'Dashboard', icon: LayoutDashboard, exact: true }} collapsed={collapsed} />

        {/* Procurement groups */}
        {procurementGroups.map(group => (
          <div key={group.label} className="pt-2">
            {!collapsed && (
              <div className="px-3 pb-1 text-[10px] font-semibold text-dark-600 uppercase tracking-widest">
                {group.label}
              </div>
            )}
            {collapsed && <div className="border-t border-dark-800/60 my-1.5" />}
            {group.items.map(item => (
              <SidebarLink key={item.path} item={item} collapsed={collapsed} />
            ))}
          </div>
        ))}

        {/* Config */}
        {!collapsed && (
          <div className="pt-2">
            <div className="px-3 pb-1 text-[10px] font-semibold text-dark-600 uppercase tracking-widest">System</div>
            <SidebarLink item={{ path: '/procurement/config', label: 'Konfiguration', icon: Settings }} collapsed={collapsed} />
          </div>
        )}
        {collapsed && (
          <>
            <div className="border-t border-dark-800/60 my-1.5" />
            <SidebarLink item={{ path: '/procurement/config', label: 'Konfiguration', icon: Settings }} collapsed={collapsed} />
          </>
        )}

        {/* Workspace */}
        <div className="pt-2">
          {!collapsed && (
            <div className="px-3 pb-1 text-[10px] font-semibold text-dark-600 uppercase tracking-widest">Workspace</div>
          )}
          {collapsed && <div className="border-t border-dark-800/60 my-1.5" />}
          {workspaceItems.map(item => (
            <SidebarLink key={item.path} item={item} collapsed={collapsed} />
          ))}
          {showAdmin && (
            <SidebarLink item={{ path: '/last', label: 'Admin', icon: Shield }} collapsed={collapsed} />
          )}
        </div>
      </nav>

      {/* Bottom: Credits + Theme + User */}
      <div className="flex-shrink-0 border-t border-dark-800 p-2 space-y-1">
        {/* Credits */}
        {showCredits && !collapsed && (
          <button
            onClick={() => navigate(hasBundles ? '/buy-credits' : '/plan')}
            className="w-full flex items-center gap-2 px-3 py-2 rounded-lg text-sm text-dark-400 hover:text-dark-100 hover:bg-dark-800/60 transition-colors"
          >
            <Zap className="w-4 h-4 text-primary-400 flex-shrink-0" />
            <span className="flex-1 text-left">{tenantCredits.toLocaleString()} Credits</span>
          </button>
        )}

        {/* Theme toggle */}
        <button
          onClick={onCycleTheme}
          title={`Erscheinungsbild: ${themeLabel}`}
          className={`flex items-center gap-3 px-3 py-2 rounded-lg text-sm text-dark-400 hover:text-dark-100 hover:bg-dark-800/60 transition-colors ${collapsed ? 'w-full justify-center' : 'w-full'}`}
        >
          <ThemeIcon className="w-4 h-4 flex-shrink-0" />
          {!collapsed && <span className="flex-1 text-left">{themeLabel}</span>}
        </button>

        {/* User + Logout */}
        <div className={`flex items-center gap-2.5 px-2 py-1.5 rounded-lg ${collapsed ? 'justify-center' : ''}`}>
          <div className="w-7 h-7 rounded-full bg-primary-500/20 border border-primary-500/30 flex items-center justify-center flex-shrink-0">
            <span className="text-primary-400 text-[10px] font-semibold">{initials}</span>
          </div>
          {!collapsed && (
            <div className="flex-1 min-w-0">
              <div className="text-xs font-medium text-dark-200 truncate">{user?.displayName}</div>
              <div className="text-[10px] text-dark-500 truncate">{user?.email}</div>
            </div>
          )}
          <button
            onClick={onLogout}
            title="Abmelden"
            className="p-1 text-dark-500 hover:text-red-400 transition-colors flex-shrink-0"
          >
            <LogOut className="w-3.5 h-3.5" />
          </button>
        </div>
      </div>
    </>
  );
}

// ─── Main Layout ──────────────────────────────────────────────────────────────

export default function Layout() {
  const location = useLocation();
  const navigate = useNavigate();
  const { user, isAuthenticated, logout, memberships } = useAuth();
  const { activeTenant, setActiveTenant } = useTenant();
  const { branding } = useBranding();
  const { theme, resolvedTheme, setTheme } = useTheme();

  const [sidebarCollapsed, setSidebarCollapsed] = useState<boolean>(
    () => localStorage.getItem('sidebar_collapsed') === 'true'
  );
  const [mobileOpen, setMobileOpen] = useState(false);
  const [cmdOpen, setCmdOpen] = useState(false);

  const [unreadCount, setUnreadCount] = useState(0);
  const [showCredits, setShowCredits] = useState(false);
  const [tenantCredits, setTenantCredits] = useState(0);
  const [hasBundles, setHasBundles] = useState(false);
  const [showTeam, setShowTeam] = useState(true);
  const [latestAnnouncement, setLatestAnnouncement] = useState<{ id: string; title: string } | null>(null);
  const [dismissedAnnouncement, setDismissedAnnouncement] = useState<string>(
    () => localStorage.getItem('dismissed_announcement') || ''
  );
  const [showTenantMenu, setShowTenantMenu] = useState(false);
  const tenantMenuRef = useRef<HTMLDivElement>(null);

  const isImpersonating = localStorage.getItem('lastsaas_impersonating') === 'true';
  const isAdmin = memberships.some(m => m.isRoot);

  // Persist sidebar state
  useEffect(() => {
    localStorage.setItem('sidebar_collapsed', String(sidebarCollapsed));
  }, [sidebarCollapsed]);

  // Close mobile sidebar on route change
  useEffect(() => { setMobileOpen(false); }, [location.pathname]);

  // Load app data
  useEffect(() => {
    if (!isAuthenticated) return;
    Promise.allSettled([
      messagesApi.unreadCount(),
      plansApi.list(),
      bundlesApi.list(),
      announcementsApi.list(),
    ]).then(([messagesResult, plansResult, bundlesResult, announcementsResult]) => {
      if (messagesResult.status === 'fulfilled') setUnreadCount(messagesResult.value.count);
      if (plansResult.status === 'fulfilled') {
        const data = plansResult.value;
        setShowCredits(data.plans.some((p: { usageCreditsPerMonth: number; bonusCredits: number }) => p.usageCreditsPerMonth > 0 || p.bonusCredits > 0));
        setTenantCredits(data.tenantSubscriptionCredits + data.tenantPurchasedCredits);
        setShowTeam(data.maxPlanUserLimit !== 1);
      }
      if (bundlesResult.status === 'fulfilled') setHasBundles(bundlesResult.value.bundles.length > 0);
      if (announcementsResult.status === 'fulfilled') {
        const anns = announcementsResult.value.announcements;
        if (anns.length > 0) setLatestAnnouncement({ id: anns[0].id, title: anns[0].title });
      }
    });
  }, [isAuthenticated]);

  // Close tenant menu on outside click
  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (tenantMenuRef.current && !tenantMenuRef.current.contains(e.target as Node)) {
        setShowTenantMenu(false);
      }
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, []);

  // Cmd+K shortcut
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault();
        setCmdOpen(true);
      }
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, []);

  const handleLogout = useCallback(async () => {
    await logout();
    navigate('/login');
  }, [logout, navigate]);

  const cycleTheme = useCallback(() => {
    const order: ThemeMode[] = ['dark', 'light', 'system'];
    const idx = order.indexOf(theme as ThemeMode);
    setTheme(order[(idx + 1) % order.length]);
  }, [theme, setTheme]);

  // Breadcrumb
  const breadcrumbs = (() => {
    const path = location.pathname;
    if (path.match(/^\/procurement\/orders\/.+/)) return ['Einkauf', 'Bestellungen', 'Detail'];
    const entry = BREADCRUMB_MAP[path];
    if (!entry) return [];
    return entry.section ? [entry.section, entry.label] : [entry.label];
  })();

  const sidebarProps = {
    collapsed: sidebarCollapsed,
    onCollapse: () => setSidebarCollapsed(c => !c),
    showTeam,
    showAdmin: isAdmin,
    cmdOpen: () => setCmdOpen(true),
    theme: theme as ThemeMode,
    resolvedTheme,
    onCycleTheme: cycleTheme,
    user: user ? { displayName: user.displayName, email: user.email } : null,
    onLogout: handleLogout,
    branding,
    showCredits,
    tenantCredits,
    hasBundles,
  };

  return (
    <div className="flex h-screen overflow-hidden bg-dark-950">
      <ImpersonationBanner />
      <CommandPalette open={cmdOpen} onClose={() => setCmdOpen(false)} />

      {/* ── Mobile sidebar overlay ── */}
      {mobileOpen && (
        <>
          <div
            className="fixed inset-0 z-40 bg-dark-950/70 backdrop-blur-sm lg:hidden"
            onClick={() => setMobileOpen(false)}
          />
          <aside className="fixed inset-y-0 left-0 z-50 w-[240px] flex flex-col bg-dark-900 border-r border-dark-800 shadow-2xl lg:hidden">
            <SidebarContent {...sidebarProps} mobileClose={() => setMobileOpen(false)} />
          </aside>
        </>
      )}

      {/* ── Desktop sidebar ── */}
      <aside
        className="hidden lg:flex flex-col flex-shrink-0 bg-dark-900 border-r border-dark-800 overflow-hidden"
        style={{ width: sidebarCollapsed ? 72 : 240, transition: 'width 0.2s ease' }}
      >
        <SidebarContent {...sidebarProps} />
      </aside>

      {/* ── Main content ── */}
      <div className="flex-1 flex flex-col min-w-0 overflow-hidden">

        {/* Top bar */}
        <header className={`flex-shrink-0 flex items-center gap-3 px-4 sm:px-6 border-b border-dark-800 bg-dark-900 ${isImpersonating ? 'h-12' : 'h-14'}`}>
          {/* Mobile hamburger */}
          <button
            onClick={() => setMobileOpen(true)}
            className="lg:hidden p-1.5 text-dark-400 hover:text-dark-100 hover:bg-dark-800 rounded-lg transition-colors"
          >
            <Menu className="w-5 h-5" />
          </button>

          {/* Breadcrumb */}
          <div className="flex-1 min-w-0 flex items-center gap-1.5 text-sm">
            {breadcrumbs.map((crumb, i) => (
              <span key={i} className="flex items-center gap-1.5">
                {i > 0 && <ChevronRight className="w-3.5 h-3.5 text-dark-600 flex-shrink-0" />}
                <span className={i === breadcrumbs.length - 1 ? 'text-dark-200 font-medium' : 'text-dark-500'}>
                  {crumb}
                </span>
              </span>
            ))}
          </div>

          {/* Right actions */}
          <div className="flex items-center gap-2 flex-shrink-0">
            {/* Tenant switcher */}
            {memberships.length > 1 && (
              <div className="relative" ref={tenantMenuRef}>
                <button
                  onClick={() => setShowTenantMenu(!showTenantMenu)}
                  className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-dark-800 border border-dark-700 text-xs text-dark-300 hover:text-dark-100 hover:border-dark-600 transition-colors"
                >
                  <span className="max-w-[100px] truncate">{activeTenant?.tenantName}</span>
                  <ChevronDown className="w-3 h-3 flex-shrink-0" />
                </button>
                {showTenantMenu && (
                  <div className="absolute right-0 mt-2 w-52 bg-dark-800 border border-dark-700 rounded-xl shadow-xl py-1 z-50">
                    {memberships.map(m => (
                      <button
                        key={m.tenantId}
                        onClick={() => { setActiveTenant(m); setShowTenantMenu(false); }}
                        className={`w-full text-left px-4 py-2.5 text-sm transition-colors ${
                          m.tenantId === activeTenant?.tenantId
                            ? 'bg-primary-500/10 text-primary-400'
                            : 'text-dark-300 hover:bg-dark-700 hover:text-dark-100'
                        }`}
                      >
                        <div className="flex items-center justify-between">
                          <span className="truncate">{m.tenantName}</span>
                          <span className="text-xs text-dark-500 capitalize ml-2">{m.role}</span>
                        </div>
                      </button>
                    ))}
                  </div>
                )}
              </div>
            )}

            {/* Messages */}
            <Link
              to="/messages"
              className="relative p-2 text-dark-400 hover:text-dark-100 hover:bg-dark-800 rounded-lg transition-colors"
              aria-label="Nachrichten"
            >
              <Bell className="w-4 h-4" />
              {unreadCount > 0 && (
                <span className="absolute top-1 right-1 bg-primary-500 text-white text-[9px] font-bold rounded-full w-3.5 h-3.5 flex items-center justify-center">
                  {unreadCount > 9 ? '9+' : unreadCount}
                </span>
              )}
            </Link>
          </div>
        </header>

        {/* Announcement banner */}
        {latestAnnouncement && latestAnnouncement.id !== dismissedAnnouncement && (
          <div className="flex-shrink-0 bg-primary-500/10 border-b border-primary-500/20">
            <div className="px-6 py-2 flex items-center justify-between">
              <div className="flex items-center gap-2 text-sm">
                <Megaphone className="w-4 h-4 text-primary-400 flex-shrink-0" />
                <span className="text-primary-300 text-sm">{latestAnnouncement.title}</span>
              </div>
              <button
                onClick={() => {
                  setDismissedAnnouncement(latestAnnouncement.id);
                  localStorage.setItem('dismissed_announcement', latestAnnouncement.id);
                }}
                className="text-xs text-dark-500 hover:text-dark-300 transition-colors ml-4 flex-shrink-0"
              >
                <X className="w-3.5 h-3.5" />
              </button>
            </div>
          </div>
        )}

        {/* Page content */}
        <main className="flex-1 overflow-y-auto p-6 lg:p-8">
          <Outlet context={{ setUnreadCount, showTeam }} />
        </main>
      </div>
    </div>
  );
}
