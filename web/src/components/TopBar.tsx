import { useLocation, useNavigate } from 'react-router-dom'
import { Search, Moon, Sun, Plus } from 'lucide-react'
import { useTheme } from '@/components/theme/ThemeProvider'
import { Button } from '@/components/ui/button'

const TITLES: Record<string, string> = {
  '/': 'Dashboard',
  '/analytics': 'Analytics',
  '/lists': 'Lists & subscribers',
  '/segments': 'Segments',
  '/signup-forms': 'Signup forms',
  '/brands': 'Brands',
  '/campaigns': 'Campaigns',
  '/automations': 'Automations',
  '/ab-tests': 'A/B tests',
  '/templates': 'Templates',
  '/deliverability': 'Deliverability',
  '/settings': 'Settings',
  '/ai': 'Ask Plume AI',
}

function titleForPath(pathname: string): string {
  if (/^\/campaigns\/[^/]+\/report$/.test(pathname)) return 'Campaigns / Report'
  if (/^\/campaigns\/[^/]+$/.test(pathname)) return 'Campaigns / Editor'
  if (/^\/lists\/[^/]+$/.test(pathname)) return 'Lists & subscribers'
  return TITLES[pathname] ?? 'Plume'
}

export function TopBar() {
  const { pathname } = useLocation()
  const nav = useNavigate()
  const { theme, toggle } = useTheme()

  return (
    <header className="sticky top-0 z-20 flex h-16 items-center gap-4 border-b bg-surface px-6">
      <h1 className="text-sm font-medium text-muted-foreground whitespace-nowrap">
        {titleForPath(pathname)}
      </h1>
      <div className="relative mx-auto hidden w-full max-w-md md:block">
        <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-faint" aria-hidden="true" />
        <input
          type="search"
          placeholder="Search or jump to…"
          aria-label="Search"
          className="w-full rounded-lg border bg-surface-2 py-2 pl-9 pr-3 text-sm outline-none focus:ring-2 focus:ring-ring"
        />
      </div>
      <div className="ml-auto flex items-center gap-2">
        <Button
          variant="outline"
          size="icon"
          onClick={toggle}
          aria-label="Toggle theme"
        >
          {theme === 'dark' ? <Sun className="size-4" /> : <Moon className="size-4" />}
        </Button>
        <Button onClick={() => nav('/campaigns')} className="gap-1">
          <Plus className="size-4" aria-hidden="true" />
          Create
        </Button>
      </div>
    </header>
  )
}
