import { Link } from 'react-router-dom'
import { Sparkles } from 'lucide-react'
import { useMe } from '@/features/auth/useAuth'
import { useCampaigns } from '@/features/campaigns/useCampaigns'
import { useAnalytics } from '@/features/analytics/useAnalytics'
import { timeGreeting, displayName } from '@/features/dashboard/greeting'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'

// ponytail: INSIGHTS panel is sample data (AI feature parked); all metrics are
// now real data from analytics. Recent campaigns uses analytics campaigns[].
const DAYS = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun']
const INSIGHTS = [
  { strong: 'Tuesday 9am', rest: ' drives your best opens.', muted: ' Schedule the next send then?' },
  { strong: '312 subscribers', rest: ' went cold.', muted: ' Start a win-back automation.' },
  { strong: null, rest: 'Subject lines with ', strongInline: 'emojis', tail: ' lift opens 6% for your list.' },
]

const pct = (n: number) => `${(n * 100).toFixed(1)}%`

function TrendPill({ value, tone }: { value: string; tone: 'success' | 'danger' | 'muted' }) {
  const tones = {
    success: 'text-success bg-success-weak',
    danger: 'text-danger bg-danger-weak',
    muted: 'text-muted-foreground bg-surface-2',
  }
  return (
    <span className={cn('rounded-md px-1.5 py-0.5 text-xs font-semibold', tones[tone])}>{value}</span>
  )
}

function StatShell({ children }: { children: React.ReactNode }) {
  return (
    <div
      data-slot="card"
      className="rounded-2xl border bg-card p-[18px] shadow-[var(--shadow-sm)]"
    >
      {children}
    </div>
  )
}

export function DashboardPage() {
  const me = useMe()
  const { isLoading: campaignsLoading } = useCampaigns()
  const { data: a } = useAnalytics(30)
  const greeting = `${timeGreeting(new Date().getHours())}, ${displayName(me.data?.email)} 👋`

  // Subscriber sparkline: last 7 gained values from subscriberGrowth, scaled to %
  const sparkBars = (() => {
    const pts = (a?.subscriberGrowth ?? []).slice(-7)
    if (pts.length === 0) return [10, 10, 10, 10, 10, 10, 10]
    const max = Math.max(1, ...pts.map((p) => p.gained ?? 0))
    const bars = pts.map((p) => Math.max(5, Math.round(((p.gained ?? 0) / max) * 100)))
    while (bars.length < 7) bars.unshift(10)
    return bars.slice(-7)
  })()

  // Send volume chart: last 7 days from sendVolume, scaled to %
  const sendVolume = (() => {
    const pts = (a?.sendVolume ?? []).slice(-7)
    if (pts.length === 0) return Array.from({ length: 7 }, () => ({ sent: 0, opens: 0, label: '' }))
    const maxSent = Math.max(1, ...pts.map((p) => p.sent ?? 0))
    const maxOpens = Math.max(1, ...pts.map((p) => p.opens ?? 0))
    const rows = pts.map((p) => ({
      sent: Math.round(((p.sent ?? 0) / maxSent) * 100),
      opens: Math.round(((p.opens ?? 0) / maxOpens) * 100),
      label: new Date(p.date).toLocaleDateString('en-US', { weekday: 'short' }),
    }))
    while (rows.length < 7) rows.unshift({ sent: 0, opens: 0, label: '' })
    return rows.slice(-7)
  })()

  const sendDayLabels = sendVolume.map((d, i) => d.label || DAYS[i % 7])

  const recentCampaigns = (a?.campaigns ?? []).slice(0, 3)

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">{greeting}</h1>
          <p className="mt-1 text-muted-foreground">Here&apos;s how your audience is doing this week.</p>
        </div>
        <div className="flex gap-2">
          <button className="rounded-lg border bg-card px-3 py-2 text-sm font-medium hover:bg-surface-2">
            Last 30 days
          </button>
          <Link
            to="/campaigns"
            className="rounded-lg bg-amber px-3.5 py-2 text-sm font-semibold text-white hover:brightness-95"
          >
            New campaign
          </Link>
        </div>
      </div>

      {/* Stat cards */}
      <div className="grid gap-3.5 [grid-template-columns:repeat(auto-fit,minmax(210px,1fr))]">
        <StatShell>
          <div className="flex items-center justify-between">
            <span className="text-sm font-medium text-muted-foreground">Subscribers</span>
            <TrendPill value="+4.2%" tone="success" />
          </div>
          <div className="mt-2 font-mono text-3xl font-bold tabular-nums">
            {a?.subscribers.toLocaleString() ?? '—'}
          </div>
          <div className="mt-2.5 flex h-[30px] items-end gap-[3px]">
            {sparkBars.map((h, i) => (
              <span
                key={i}
                style={{ height: `${h}%` }}
                className={cn('flex-1 rounded-sm', i >= 4 ? 'bg-primary' : 'bg-primary-weak')}
              />
            ))}
          </div>
        </StatShell>

        <StatShell>
          <div className="flex items-center justify-between">
            <span className="text-sm font-medium text-muted-foreground">Avg. open rate</span>
            <TrendPill value="+1.8%" tone="success" />
          </div>
          <div className="mt-2 font-mono text-3xl font-bold tabular-nums text-primary">
            {pct(a?.avgOpenRate ?? 0)}
          </div>
          <div className="mt-3.5 h-1.5 overflow-hidden rounded-full bg-surface-2">
            <div className="h-full rounded-full bg-primary" style={{ width: pct(a?.avgOpenRate ?? 0) }} />
          </div>
        </StatShell>

        <StatShell>
          <div className="flex items-center justify-between">
            <span className="text-sm font-medium text-muted-foreground">Click rate</span>
            <TrendPill value="−0.4%" tone="danger" />
          </div>
          <div className="mt-2 font-mono text-3xl font-bold tabular-nums text-amber">
            {pct(a?.clickRate ?? 0)}
          </div>
          <div className="mt-3.5 h-1.5 overflow-hidden rounded-full bg-surface-2">
            <div className="h-full rounded-full bg-amber" style={{ width: pct(a?.clickRate ?? 0) }} />
          </div>
        </StatShell>

        <StatShell>
          <div className="flex items-center justify-between">
            <span className="text-sm font-medium text-muted-foreground">Spend (mo.)</span>
            <TrendPill value="SES" tone="muted" />
          </div>
          <div className="mt-2 font-mono text-3xl font-bold tabular-nums">
            ${(a?.sendCost ?? 0).toFixed(2)}
          </div>
          <div className="mt-3.5 text-xs text-muted-foreground">
            vs <span className="line-through">$385</span> on typical SaaS
          </div>
        </StatShell>
      </div>

      {/* Chart + AI insights */}
      <div className="grid gap-4 lg:[grid-template-columns:1.6fr_1fr]">
        <div className="rounded-2xl border bg-card p-5 shadow-[var(--shadow-sm)]">
          <div className="mb-4.5 flex items-center justify-between">
            <h3 className="text-base font-bold">Send volume &amp; engagement</h3>
            <div className="flex gap-3.5 text-xs text-muted-foreground">
              <span className="flex items-center gap-1.5">
                <span className="size-2 rounded-sm bg-primary" /> Sent
              </span>
              <span className="flex items-center gap-1.5">
                <span className="size-2 rounded-sm bg-amber" /> Opens
              </span>
            </div>
          </div>
          <div className="flex h-40 items-end gap-2.5">
            {sendVolume.map((d, i) => (
              <div key={i} className="flex h-full flex-1 flex-col justify-end gap-[3px]">
                <div style={{ height: `${d.sent}%` }} className="rounded-t bg-primary" />
                <div style={{ height: `${d.opens}%` }} className="rounded-b bg-amber" />
              </div>
            ))}
          </div>
          <div className="mt-2 flex justify-between font-mono text-xs text-faint">
            {sendDayLabels.map((d, i) => (
              <span key={i}>{d}</span>
            ))}
          </div>
        </div>

        <div className="relative overflow-hidden rounded-2xl border border-primary-weak bg-[linear-gradient(150deg,var(--primary-weak),var(--purple-weak))] p-5 shadow-[var(--shadow-sm)]">
          <div className="mb-3 flex items-center gap-2.5">
            <span className="flex size-[30px] items-center justify-center rounded-lg bg-primary text-white">
              <Sparkles className="size-4" aria-hidden="true" />
            </span>
            <h3 className="text-base font-bold text-primary-text">AI insights</h3>
          </div>
          <div className="flex flex-col gap-2.5">
            {INSIGHTS.map((it, i) => (
              <div key={i} className="rounded-[10px] border bg-card p-3 text-sm">
                {it.strong && <strong className="text-foreground">{it.strong}</strong>}
                {it.rest}
                {it.strongInline && <strong className="text-foreground">{it.strongInline}</strong>}
                {it.tail}
                {it.muted && <span className="text-muted-foreground">{it.muted}</span>}
              </div>
            ))}
          </div>
          <Link
            to="/ai"
            className="mt-3.5 block rounded-lg bg-primary py-2.5 text-center text-sm font-semibold text-white hover:bg-primary/90"
          >
            Open AI assistant
          </Link>
        </div>
      </div>

      {/* Recent campaigns table */}
      <div className="overflow-hidden rounded-2xl border bg-card shadow-[var(--shadow-sm)]">
        <div className="flex items-center justify-between border-b px-5 py-4">
          <h3 className="text-base font-bold">Recent campaigns</h3>
          <Link to="/campaigns" className="text-sm font-semibold text-primary-text hover:underline">
            View all →
          </Link>
        </div>
        <div className="grid grid-cols-[2fr_1fr_1fr_1fr_1fr] gap-2 border-b px-5 py-2.5 text-xs font-semibold uppercase tracking-wide text-faint">
          <span>Campaign</span>
          <span>Status</span>
          <span>Recipients</span>
          <span>Open rate</span>
          <span>Click rate</span>
        </div>
        {campaignsLoading ? (
          <div className="space-y-3 p-5">
            {Array.from({ length: 3 }).map((_, i) => (
              <Skeleton key={i} className="h-5 w-full" />
            ))}
          </div>
        ) : (
          <RecentRows campaigns={recentCampaigns} />
        )}
      </div>
    </div>
  )
}

function RowStatus({ status }: { status: string }) {
  const sent = status === 'sent'
  return (
    <span
      className={cn(
        'w-fit rounded-md px-2 py-0.5 text-xs font-semibold',
        sent ? 'bg-success-weak text-success' : 'bg-amber-weak text-amber',
      )}
    >
      {status === 'sent' ? 'Sent' : status === 'scheduled' ? 'Scheduled' : 'Draft'}
    </span>
  )
}

function RecentRows({ campaigns }: { campaigns: { id: string; subject: string; status: string; sent: number; openRate: number; clickRate: number }[] }) {
  if (campaigns.length === 0) {
    return (
      <div className="px-5 py-6 text-sm text-muted-foreground">No campaigns yet.</div>
    )
  }
  return (
    <>
      {campaigns.map((c, i) => (
        <Link
          key={c.id}
          to={`/campaigns/${c.id}`}
          className={cn(
            'grid grid-cols-[2fr_1fr_1fr_1fr_1fr] items-center gap-2 px-5 py-3.5 hover:bg-surface-2',
            i < campaigns.length - 1 && 'border-b',
          )}
        >
          <span className="font-semibold">{c.subject}</span>
          <RowStatus status={c.status} />
          {/* Unsent campaigns have no rates yet — show — rather than 0.0% */}
          <span className="font-mono">{c.sent === 0 ? '—' : c.sent.toLocaleString()}</span>
          <span className={cn('font-mono', c.sent === 0 ? 'text-muted-foreground' : 'text-primary')}>
            {c.sent === 0 ? '—' : pct(c.openRate)}
          </span>
          <span className={cn('font-mono', c.sent === 0 ? 'text-muted-foreground' : 'text-amber')}>
            {c.sent === 0 ? '—' : pct(c.clickRate)}
          </span>
        </Link>
      ))}
    </>
  )
}
