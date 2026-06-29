import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Mail, Plus, Search } from 'lucide-react'
import { toast } from 'sonner'
import { useCampaigns, useCreateCampaign, type Campaign } from './useCampaigns'
import { useBrands } from '@/features/brands/useBrands'
import { useAnalytics } from '@/features/analytics/useAnalytics'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Dialog, DialogTrigger, DialogContent, DialogHeader, DialogTitle, DialogFooter,
} from '@/components/ui/dialog'
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { EmptyState } from '@/components/EmptyState'
import { cn } from '@/lib/utils'

// Status badge: raw status text (capitalized via CSS so tests still match the value),
// colored by lifecycle. sent→success, scheduled/queued/sending→amber, failed→danger.
export function CampaignStatusBadge({ status }: { status: string }) {
  const tone =
    status === 'sent' ? 'bg-success-weak text-success'
    : status === 'failed' ? 'bg-danger-weak text-danger'
    : status === 'draft' ? 'bg-surface-2 text-muted-foreground'
    : 'bg-amber-weak text-amber'
  return <span className={cn('rounded-md px-2 py-0.5 text-xs font-semibold capitalize', tone)}>{status}</span>
}

const createSchema = z.object({ subject: z.string().min(1) })

function NewCampaignDialog({ trigger }: { trigger: React.ReactElement }) {
  const [open, setOpen] = useState(false)
  const [brandId, setBrandId] = useState('')
  const { data: brands } = useBrands()
  const create = useCreateCampaign()
  const navigate = useNavigate()
  const { register, handleSubmit, reset, formState: { errors } } = useForm<{ subject: string }>({
    resolver: zodResolver(createSchema),
    defaultValues: { subject: '' },
  })

  const onSubmit = (data: { subject: string }) => {
    if (!brandId) return
    create.mutate(
      { brandId, subject: data.subject, htmlBody: '', plainBody: '' },
      {
        onSuccess: (created: Campaign) => {
          setOpen(false); reset(); setBrandId('')
          toast.success('Campaign created')
          navigate(`/campaigns/${created.id}`)
        },
        onError: () => toast.error('Something went wrong'),
      },
    )
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={trigger} />
      <DialogContent>
        <DialogHeader><DialogTitle>New campaign</DialogTitle></DialogHeader>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-3">
          <div>
            <Label htmlFor="campaign-brand">Brand</Label>
            <Select value={brandId} onValueChange={(v) => setBrandId(v ?? '')}>
              <SelectTrigger id="campaign-brand" className="w-full">
                <SelectValue placeholder="Select brand">
                  {(v) => brands?.find((b) => b.id === v)?.name ?? 'Select brand'}
                </SelectValue>
              </SelectTrigger>
              <SelectContent>
                {brands?.map((b) => <SelectItem key={b.id} value={b.id}>{b.name}</SelectItem>)}
              </SelectContent>
            </Select>
          </div>
          <div>
            <Label htmlFor="campaign-subject">Subject</Label>
            <Input id="campaign-subject" {...register('subject')} />
            {errors.subject && <p className="text-sm text-destructive">Required</p>}
          </div>
          <DialogFooter>
            <Button type="submit" disabled={!brandId || create.isPending}>Create</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

const FILTERS = ['All', 'Sent', 'Scheduled', 'Drafts'] as const
type Filter = (typeof FILTERS)[number]

function matchesFilter(status: string, f: Filter): boolean {
  if (f === 'All') return true
  if (f === 'Sent') return status === 'sent'
  if (f === 'Drafts') return status === 'draft'
  return status === 'scheduled' || status === 'queued' || status === 'sending'
}

const COLS = 'grid grid-cols-[2.4fr_1fr_1.1fr_1fr_1fr_1fr] gap-2.5 px-5'

const pct = (n: number) => `${(n * 100).toFixed(1)}%`

export function CampaignsPage() {
  const { data: campaigns, isLoading } = useCampaigns()
  const { data: brands } = useBrands()
  const { data: a } = useAnalytics(30)
  const metrics = new Map(a?.campaigns.map((c) => [c.id, c]) ?? [])
  const [filter, setFilter] = useState<Filter>('All')
  const [query, setQuery] = useState('')
  const navigate = useNavigate()

  const brandName = (id: string) => brands?.find((b) => b.id === id)?.name ?? 'Brand'
  const rows = (campaigns ?? []).filter(
    (c) => matchesFilter(c.status, filter) && c.subject.toLowerCase().includes(query.toLowerCase()),
  )

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Campaigns</h1>
          <p className="mt-1 text-muted-foreground">Broadcasts and one-off sends across all brands.</p>
        </div>
        <NewCampaignDialog trigger={<Button className="gap-1.5"><Plus className="size-4" aria-hidden="true" />New campaign</Button>} />
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <div className="flex gap-0.5 rounded-lg bg-surface-2 p-[3px]">
          {FILTERS.map((f) => (
            <button key={f} type="button" onClick={() => setFilter(f)}
              className={cn('rounded-md px-3.5 py-1.5 text-sm font-medium',
                filter === f ? 'bg-card text-primary-text shadow-sm' : 'text-muted-foreground')}>
              {f}
            </button>
          ))}
        </div>
        <div className="relative ml-auto flex items-center">
          <Search className="pointer-events-none absolute left-3 size-3.5 text-faint" aria-hidden="true" />
          <input value={query} onChange={(e) => setQuery(e.target.value)} placeholder="Search campaigns"
            aria-label="Search campaigns"
            className="rounded-lg border bg-surface py-2 pl-8 pr-3 text-sm outline-none focus:border-primary focus:ring-2 focus:ring-primary-weak" />
        </div>
      </div>

      {isLoading ? (
        <div className="space-y-3 rounded-2xl border bg-card p-5">
          {[0, 1, 2, 3].map((i) => <Skeleton key={i} className="h-9 w-full" />)}
        </div>
      ) : !campaigns?.length ? (
        <EmptyState icon={Mail} title="No campaigns yet"
          description="Create your first campaign to get started."
          action={<NewCampaignDialog trigger={<Button>New campaign</Button>} />} />
      ) : (
        <div className="overflow-hidden rounded-2xl border bg-card shadow-[var(--shadow-sm)]">
          <div className={cn(COLS, 'border-b py-3 text-xs font-semibold uppercase tracking-wide text-faint')}>
            <span>Campaign</span><span>Status</span><span>Sent</span><span>Open</span><span>Click</span><span>Revenue</span>
          </div>
          {rows.map((c) => {
            const m = metrics.get(c.id)
            return (
            <button key={c.id} type="button" onClick={() => navigate(`/campaigns/${c.id}`)}
              className={cn(COLS, 'items-center border-b py-3.5 text-left last:border-b-0 hover:bg-surface-2')}>
              <span className="flex min-w-0 items-center gap-2.5">
                <span className="flex size-[34px] shrink-0 items-center justify-center rounded-lg bg-surface-2 text-muted-foreground">
                  <Mail className="size-4" aria-hidden="true" />
                </span>
                <span className="min-w-0">
                  <span className="block truncate font-semibold">{c.subject}</span>
                  <span className="block text-xs text-muted-foreground">{brandName(c.brand_id)} · broadcast</span>
                </span>
              </span>
              <span><CampaignStatusBadge status={c.status} /></span>
              <span className="font-mono text-sm text-muted-foreground">{m && m.sent > 0 ? m.sent.toLocaleString() : '—'}</span>
              <span className="font-mono text-sm text-muted-foreground">{m && m.sent > 0 ? pct(m.openRate) : '—'}</span>
              <span className="font-mono text-sm text-muted-foreground">{m && m.sent > 0 ? pct(m.clickRate) : '—'}</span>
              <span className="font-mono text-sm text-muted-foreground">—</span>
            </button>
            )
          })}
          {rows.length === 0 && (
            <div className="px-5 py-10 text-center text-sm text-muted-foreground">No campaigns match this view.</div>
          )}
        </div>
      )}
    </div>
  )
}
