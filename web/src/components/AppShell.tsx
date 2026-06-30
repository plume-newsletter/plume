import { useState } from 'react'
import { Outlet, NavLink, useNavigate, useLocation } from 'react-router-dom'
import {
  LayoutDashboard,
  BarChart3,
  List,
  Filter,
  ClipboardList,
  Building2,
  Mail,
  Workflow,
  FlaskConical,
  LayoutTemplate,
  ShieldCheck,
  Settings,
  Sparkles,
  LogOut,
  Menu,
  X,
  Feather,
} from 'lucide-react'
import { useMe, useLogout } from '@/features/auth/useAuth'
import { Button } from '@/components/ui/button'
import { TopBar } from '@/components/TopBar'
import { cn } from '@/lib/utils'
import { AiAssistantProvider, useAiAssistant } from '@/features/ai/AiAssistant'

type NavItem = {
  to: string
  label: string
  icon: typeof LayoutDashboard
  end?: boolean
  soon?: boolean
}
type NavGroup = { label: string; items: NavItem[] }

const navGroups: NavGroup[] = [
  {
    label: 'Overview',
    items: [
      { to: '/', label: 'Dashboard', icon: LayoutDashboard, end: true },
      { to: '/analytics', label: 'Analytics', icon: BarChart3 },
    ],
  },
  {
    label: 'Audience',
    items: [
      { to: '/lists', label: 'Lists & subscribers', icon: List },
      { to: '/segments', label: 'Segments', icon: Filter },
      { to: '/signup-forms', label: 'Signup forms', icon: ClipboardList },
      { to: '/brands', label: 'Brands', icon: Building2 },
    ],
  },
  {
    label: 'Campaigns',
    items: [
      { to: '/campaigns', label: 'Campaigns', icon: Mail },
      { to: '/automations', label: 'Automations', icon: Workflow },
      { to: '/ab-tests', label: 'A/B tests', icon: FlaskConical },
      { to: '/templates', label: 'Templates', icon: LayoutTemplate },
    ],
  },
  {
    label: 'System',
    items: [
      { to: '/deliverability', label: 'Deliverability', icon: ShieldCheck, soon: true },
      { to: '/settings', label: 'Settings', icon: Settings },
    ],
  },
]

function NavLinks({ onNavigate }: { onNavigate?: () => void }) {
  return (
    <nav className="flex-1 space-y-4 overflow-y-auto px-3 py-4">
      {navGroups.map((group) => (
        <div key={group.label}>
          <div className="px-3 pb-1 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
            {group.label}
          </div>
          <div className="space-y-1">
            {group.items.map(({ to, label, icon: Icon, end, soon }) => (
              <NavLink
                key={to}
                to={to}
                end={end}
                onClick={onNavigate}
                className={({ isActive }) =>
                  cn(
                    'flex items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium transition-colors',
                    isActive
                      ? 'bg-sidebar-active text-primary-text'
                      : 'text-muted-foreground hover:bg-muted hover:text-foreground',
                  )
                }
              >
                <Icon className="size-4 shrink-0" aria-hidden="true" />
                <span className="flex-1">{label}</span>
                {soon && (
                  <span className="rounded-full bg-muted px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide text-muted-foreground">
                    Soon
                  </span>
                )}
              </NavLink>
            ))}
          </div>
        </div>
      ))}
    </nav>
  )
}

function SidebarFooter({ onNavigate }: { onNavigate?: () => void }) {
  const nav = useNavigate()
  const me = useMe()
  const logout = useLogout()
  const ai = useAiAssistant()
  const fullName = me.data?.fullName || me.data?.email?.split('@')[0] || ''
  const initials = (fullName || me.data?.email || '··').slice(0, 2).toUpperCase()

  return (
    <div className="border-t p-3 space-y-2">
      <button
        type="button"
        onClick={() => {
          onNavigate?.()
          ai.toggle()
        }}
        className="flex w-full items-center gap-2 rounded-lg bg-primary-weak px-3 py-2 text-sm font-medium text-primary-text hover:opacity-90"
      >
        <Sparkles className="size-4 shrink-0" aria-hidden="true" />
        Ask Plume AI
      </button>
      <div className="flex items-center gap-2 px-1">
        <span
          className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-surface-2 text-xs font-semibold"
          aria-hidden="true"
        >
          {initials}
        </span>
        <span className="flex min-w-0 flex-1 flex-col">
          <span className="truncate text-sm font-medium">{fullName}</span>
          <span className="truncate text-xs text-muted-foreground">{me.data?.email}</span>
        </span>
        <Button
          variant="ghost"
          size="icon"
          aria-label="Log out"
          onClick={() => {
            onNavigate?.()
            logout.mutate(undefined, { onSuccess: () => nav('/login') })
          }}
        >
          <LogOut className="size-4" aria-hidden="true" />
        </Button>
      </div>
    </div>
  )
}

function Brand() {
  return (
    <div className="flex items-center gap-2 px-4 py-4">
      <span className="flex size-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
        <Feather className="size-4" aria-hidden="true" />
      </span>
      <span className="text-lg font-semibold">Plume</span>
      <span className="rounded-full bg-muted px-1.5 py-0.5 text-[10px] font-medium text-muted-foreground">
        v1.0
      </span>
    </div>
  )
}

export function AppShell() {
  const [open, setOpen] = useState(false)
  // The email builder fills the viewport (no centered, padded content wrapper).
  const fullBleed = /^\/campaigns\/[^/]+$/.test(useLocation().pathname)

  return (
    <AiAssistantProvider>
    <div className="flex min-h-screen">
      {/* Desktop sidebar */}
      <div className="hidden lg:flex lg:w-64 lg:flex-col lg:border-r bg-sidebar">
        <Brand />
        <NavLinks />
        <SidebarFooter />
      </div>

      {/* Mobile overlay sidebar */}
      {open && (
        <div className="fixed inset-0 z-40 flex lg:hidden">
          <div
            className="fixed inset-0 bg-black/50"
            onClick={() => setOpen(false)}
            aria-hidden="true"
          />
          <div className="relative z-50 flex w-64 flex-col bg-background border-r">
            <div className="flex items-center justify-between">
              <Brand />
              <Button
                variant="ghost"
                size="icon"
                className="mr-3"
                aria-label="Close menu"
                onClick={() => setOpen(false)}
              >
                <X className="size-4" aria-hidden="true" />
              </Button>
            </div>
            <NavLinks onNavigate={() => setOpen(false)} />
            <SidebarFooter onNavigate={() => setOpen(false)} />
          </div>
        </div>
      )}

      <div className="flex min-w-0 flex-1 flex-col">
        {/* Mobile menu button row (sidebar is overlay on mobile) */}
        <div className="flex items-center gap-2 border-b p-3 lg:hidden">
          <Button variant="ghost" size="icon" aria-label="Open menu" onClick={() => setOpen(true)}>
            <Menu className="size-4" aria-hidden="true" />
          </Button>
          <span className="text-lg font-semibold">Plume</span>
        </div>
        <div className="hidden lg:block">
          <TopBar />
        </div>
        <main className="flex-1 bg-background">
          {fullBleed ? (
            <Outlet />
          ) : (
            <div className="mx-auto max-w-6xl p-6">
              <Outlet />
            </div>
          )}
        </main>
      </div>
    </div>
    </AiAssistantProvider>
  )
}
