import { useState } from 'react'
import { useDeliverability } from './useDeliverability'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'

const pct = (n: number) => `${(n * 100).toFixed(2)}%`

// SES enforces a bounce rate under ~5% and complaint rate under ~0.1%.
// Color the rates against those thresholds so a sender problem is obvious.
function bounceTone(r: number) {
  if (r >= 0.05) return 'text-danger'
  if (r >= 0.03) return 'text-amber'
  return 'text-success'
}
function complaintTone(r: number) {
  if (r >= 0.001) return 'text-danger'
  if (r >= 0.0005) return 'text-amber'
  return 'text-success'
}

function Card({ label, value, tone, hint }: { label: string; value: string; tone?: string; hint?: string }) {
  return (
    <div className="rounded-2xl border bg-card p-[18px] shadow-[var(--shadow-sm)]">
      <div className="text-sm text-muted-foreground">{label}</div>
      <div className={cn('mt-1.5 font-mono text-[1.7rem] font-bold tabular-nums', tone)}>{value}</div>
      {hint && <div className="mt-0.5 text-xs text-faint">{hint}</div>}
    </div>
  )
}

const reasonTone: Record<string, string> = {
  complaint: 'bg-danger-weak text-danger',
  bounce: 'bg-amber/15 text-amber',
  unsubscribe: 'bg-surface-2 text-muted-foreground',
}

export function DeliverabilityPage() {
  const [window, setWindow] = useState<30 | 90>(30)
  const { data, isLoading } = useDeliverability(window)

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Deliverability</h1>
          <p className="mt-1 text-muted-foreground">Monitor bounces, complaints, and sender health.</p>
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
            <Card label="Sent" value={data.sent.toLocaleString()} />
            <Card label="Bounce rate" value={pct(data.bounceRate)} tone={bounceTone(data.bounceRate)} hint={`${data.bounces.toLocaleString()} bounces · keep under 5%`} />
            <Card label="Complaint rate" value={pct(data.complaintRate)} tone={complaintTone(data.complaintRate)} hint={`${data.complaints.toLocaleString()} complaints · keep under 0.1%`} />
            <Card label="Suppressed" value={data.suppressed.toLocaleString()} hint="addresses removed from sends" />
          </div>

          <div className="rounded-2xl border bg-card shadow-[var(--shadow-sm)]">
            <h3 className="border-b px-5 py-4 text-base font-bold">Suppression list</h3>
            {data.suppressions.length === 0 ? (
              <p className="px-5 py-10 text-center text-sm text-muted-foreground">
                No suppressed addresses. Bounces and complaints will appear here.
              </p>
            ) : (
              <table className="w-full text-sm">
                <thead>
                  <tr className="text-left text-xs uppercase tracking-wide text-faint">
                    <th className="px-5 py-2.5 font-semibold">Email</th>
                    <th className="px-5 py-2.5 font-semibold">Reason</th>
                    <th className="px-5 py-2.5 text-right font-semibold">Date</th>
                  </tr>
                </thead>
                <tbody>
                  {data.suppressions.map((s) => (
                    <tr key={s.email} className="border-t">
                      <td className="px-5 py-2.5 font-mono">{s.email}</td>
                      <td className="px-5 py-2.5">
                        <span className={cn('rounded-md px-1.5 py-0.5 text-xs font-semibold capitalize', reasonTone[s.reason] ?? 'bg-surface-2 text-muted-foreground')}>
                          {s.reason}
                        </span>
                      </td>
                      <td className="px-5 py-2.5 text-right tabular-nums text-muted-foreground">{s.date}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </>
      )}
    </div>
  )
}
