import { Link, useParams } from 'react-router-dom'
import { ChevronLeft, BarChart3 } from 'lucide-react'
import { useReport, type ReportSummary } from './useReport'
import { useCampaign } from './useCampaign'
import { CampaignStatusBadge } from './CampaignsPage'
import { EmptyState } from '@/components/EmptyState'
import { Skeleton } from '@/components/ui/skeleton'

const SES_PER_1K = 0.1 // pure operating cost the owner pays SES; dev subscription is not counted

function pct(n: number, d: number) {
  return d > 0 ? `${((n / d) * 100).toFixed(1)}%` : '—'
}

function Tile({ label, value, sub, tone }: { label: string; value: string; sub: string; tone?: 'primary' | 'amber' }) {
  return (
    <div data-slot="card" className="rounded-2xl border bg-card p-[18px] shadow-[var(--shadow-sm)]">
      <div className="text-xs font-medium text-muted-foreground">{label}</div>
      <div className={`mt-1.5 font-mono text-[1.7rem] font-bold tabular-nums ${tone === 'primary' ? 'text-primary' : tone === 'amber' ? 'text-amber' : ''}`}>
        {value}
      </div>
      <div className={`mt-1 text-xs ${tone === 'primary' ? 'text-primary-text' : tone === 'amber' ? 'text-amber' : 'text-muted-foreground'}`}>{sub}</div>
    </div>
  )
}

// ponytail: the 48h engagement curve and top-links are sample until per-event
// analytics lands (feature E); the metric tiles use real report data.
const ENGAGE = [30, 62, 100, 78, 54, 40, 48, 33, 25, 18]
const LINKS = [
  { href: '/whats-new', clicks: 612, width: 88 },
  { href: '/pricing', clicks: 341, width: 52 },
  { href: '/changelog', clicks: 220, width: 34 },
]

function Report({ data }: { data: ReportSummary }) {
  const delivered = Math.max(0, data.recipients - data.bounces)
  return (
    <>
      <div className="mb-4 grid gap-3.5 [grid-template-columns:repeat(auto-fit,minmax(150px,1fr))]">
        <Tile label="Delivered" value={delivered.toLocaleString()} sub={`${pct(delivered, data.recipients)} delivery`} />
        <Tile label="Unique opens" value={data.opens.unique.toLocaleString()} sub={`${pct(data.opens.unique, delivered)} open rate`} tone="primary" />
        <Tile label="Clicks" value={data.clicks.total.toLocaleString()} sub={`${pct(data.clicks.total, delivered)} click rate`} tone="amber" />
        <Tile label="Bounced" value={data.bounces.toLocaleString()} sub={`${pct(data.bounces, data.recipients)} · auto-suppressed`} />
        <Tile label="SES cost" value={`$${((data.sent / 1000) * SES_PER_1K).toFixed(2)}`} sub="$0.10 / 1k" />
      </div>

      <div className="grid gap-4 lg:[grid-template-columns:1.5fr_1fr]">
        <div className="rounded-2xl border bg-card p-5 shadow-[var(--shadow-sm)]">
          <h3 className="mb-4 text-base font-bold">Engagement over 48 hours</h3>
          <div className="flex h-[150px] items-end gap-1.5">
            {ENGAGE.map((h, i) => (
              <div key={i} style={{ height: `${h}%` }} className={`flex-1 rounded-t ${i < 4 ? 'bg-primary' : 'bg-primary-weak'}`} />
            ))}
          </div>
          <div className="mt-2 flex justify-between font-mono text-xs text-faint">
            <span>0h</span><span>12h</span><span>24h</span><span>36h</span><span>48h</span>
          </div>
        </div>

        <div className="rounded-2xl border bg-card p-5 shadow-[var(--shadow-sm)]">
          <h3 className="mb-4 text-base font-bold">Top clicked links</h3>
          <div className="flex flex-col gap-3">
            {LINKS.map((l) => (
              <div key={l.href}>
                <div className="mb-1.5 flex justify-between text-sm">
                  <span className="truncate font-mono text-primary-text">{l.href}</span>
                  <span className="font-semibold">{l.clicks}</span>
                </div>
                <div className="h-1.5 rounded-full bg-surface-2">
                  <div className="h-full rounded-full bg-amber" style={{ width: `${l.width}%` }} />
                </div>
              </div>
            ))}
          </div>
          <div className="mt-4 rounded-xl border border-primary-weak bg-[linear-gradient(140deg,var(--primary-weak),var(--purple-weak))] p-3 text-sm">
            <strong className="text-primary-text">AI summary:</strong> Strong opens (+6pts vs avg). Clicks concentrated on “What&apos;s new” — consider a follow-up to non-clickers.
          </div>
        </div>
      </div>
    </>
  )
}

export function ReportPage() {
  const { id } = useParams<{ id: string }>()
  const { data, isLoading, error } = useReport(id!)
  const campaign = useCampaign(id!)

  return (
    <div className="space-y-5">
      <Link to="/campaigns"
        className="inline-flex w-fit items-center gap-1.5 rounded-lg border bg-card px-2.5 py-1.5 text-sm font-medium hover:bg-surface-2">
        <ChevronLeft className="size-3.5" aria-hidden="true" /> All campaigns
      </Link>

      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <div className="flex items-center gap-2.5">
            <h1 className="text-2xl font-bold tracking-tight">{campaign.data?.subject ?? 'Campaign report'}</h1>
            {campaign.data && <CampaignStatusBadge status={campaign.data.status} />}
          </div>
          <p className="mt-1 text-sm text-muted-foreground">Delivery, engagement, and pure SES cost for this send.</p>
        </div>
        <div className="flex gap-2">
          <button className="rounded-lg border bg-card px-3 py-2 text-sm font-medium hover:bg-surface-2">Export CSV</button>
          <button className="rounded-lg border bg-card px-3 py-2 text-sm font-medium hover:bg-surface-2">Duplicate</button>
        </div>
      </div>

      {isLoading && (
        <div className="grid gap-3.5 [grid-template-columns:repeat(auto-fit,minmax(150px,1fr))]">
          {Array.from({ length: 5 }).map((_, i) => (
            <div key={i} className="rounded-2xl border bg-card p-[18px]">
              <Skeleton className="h-4 w-20" /><Skeleton className="mt-3 h-7 w-16" />
            </div>
          ))}
        </div>
      )}

      {!isLoading && (error || !data) && (
        <EmptyState icon={BarChart3} title="No report yet." description="Send this campaign to see results." />
      )}

      {data && <Report data={data} />}
    </div>
  )
}
