import { useState } from 'react'
import { useAnalytics } from './useAnalytics'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'

const pct = (n: number) => `${(n * 100).toFixed(1)}%`

function Card({ label, value, tone }: { label: string; value: string; tone?: string }) {
  return (
    <div className="rounded-2xl border bg-card p-[18px] shadow-[var(--shadow-sm)]">
      <div className="text-sm text-muted-foreground">{label}</div>
      <div className={cn('mt-1.5 font-mono text-[1.7rem] font-bold tabular-nums', tone)}>{value}</div>
    </div>
  )
}

export function AnalyticsPage() {
  const [window, setWindow] = useState<30 | 90>(30)
  const { data, isLoading } = useAnalytics(window)

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Analytics</h1>
          <p className="mt-1 text-muted-foreground">List growth, engagement trends, and pure send cost across every brand.</p>
        </div>
        <button onClick={() => setWindow((w) => (w === 30 ? 90 : 30))}
          className="rounded-lg border bg-card px-3 py-2 text-sm font-medium hover:bg-surface-2">
          Last {window === 30 ? 90 : 30} days
        </button>
      </div>

      {isLoading || !data ? (
        <Skeleton className="h-64 w-full rounded-2xl" />
      ) : (
        <>
          <div className="grid gap-3.5 [grid-template-columns:repeat(auto-fit,minmax(180px,1fr))]">
            <Card label="Net new subs" value={`+${data.netNewSubs.toLocaleString()}`} tone="text-success" />
            <Card label="Avg open rate" value={pct(data.avgOpenRate)} tone="text-primary" />
            <Card label="Click rate" value={pct(data.clickRate)} tone="text-amber" />
            <Card label="Total SES cost" value={`$${data.sendCost.toFixed(2)}`} />
          </div>

          <div className="rounded-2xl border bg-card p-5 shadow-[var(--shadow-sm)]">
            <h3 className="mb-4 text-base font-bold">Subscriber growth</h3>
            <GrowthChart points={data.subscriberGrowth} />
          </div>

          <div className="grid gap-4 lg:grid-cols-2">
            <div className="rounded-2xl border bg-card p-5 shadow-[var(--shadow-sm)]">
              <h3 className="mb-3.5 text-base font-bold">Best send times</h3>
              <div className="flex flex-col gap-2.5">
                {data.bestSendTimes.length === 0 && <p className="text-sm text-muted-foreground">No opens yet.</p>}
                {data.bestSendTimes.map((t) => (
                  <div key={t.label}>
                    <div className="mb-1 flex justify-between text-sm"><span>{t.label}</span><span className="font-semibold">{pct(t.rate)}</span></div>
                    <div className="h-1.5 rounded-full bg-surface-2"><div className="h-full rounded-full bg-primary" style={{ width: pct(t.rate) }} /></div>
                  </div>
                ))}
              </div>
            </div>
            <div className="rounded-2xl border bg-card p-5 shadow-[var(--shadow-sm)]">
              <h3 className="mb-3.5 text-base font-bold">Top campaigns by engagement</h3>
              <div className="flex flex-col gap-2.5 text-sm">
                {data.topCampaigns.length === 0 && <p className="text-muted-foreground">No campaigns yet.</p>}
                {data.topCampaigns.map((c) => (
                  <div key={c.id} className="flex justify-between"><span className="truncate">{c.subject}</span><span className="font-mono font-semibold">{c.opens.toLocaleString()} opens</span></div>
                ))}
              </div>
            </div>
          </div>
        </>
      )}
    </div>
  )
}

// Plots the cumulative subscriber count (running gained − lost) as a rising
// area+line — "growth" reads as a total over time, like the prototype.
function GrowthChart({ points }: { points: { date: string; gained?: number; lost?: number }[] }) {
  if (points.length === 0) return <p className="text-sm text-muted-foreground">No growth data in this window yet.</p>
  let running = 0
  const totals = points.map((p) => (running += (p.gained ?? 0) - (p.lost ?? 0)))
  const max = Math.max(1, ...totals)
  const step = points.length > 1 ? 600 / (points.length - 1) : 600
  // leave 10px headroom top, baseline at 158
  const xy = totals.map((t, i) => `${i * step},${158 - (t / max) * 148}`)
  const line = xy.join(' ')
  const area = `0,160 ${line} ${(points.length - 1) * step},160`
  return (
    <svg viewBox="0 0 600 160" preserveAspectRatio="none" className="block h-40 w-full">
      <polygon points={area} fill="var(--primary)" opacity={0.12} stroke="none" />
      <polyline points={line} fill="none" stroke="var(--primary)" strokeWidth={2.5} vectorEffect="non-scaling-stroke" />
    </svg>
  )
}
