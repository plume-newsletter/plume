import { useState } from 'react'
import { Plus } from 'lucide-react'
import { toast } from 'sonner'
import {
  useABTests, useABTestResults, useCreateABTest, useStartABTest, useSendWinner,
  type ABTest, type VariantResult,
} from './useABTests'
import { useCampaigns } from '@/features/campaigns/useCampaigns'
import { useLists } from '@/features/lists/useLists'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Dialog, DialogTrigger, DialogContent, DialogHeader, DialogTitle, DialogFooter,
} from '@/components/ui/dialog'
import { Skeleton } from '@/components/ui/skeleton'
import { EmptyState } from '@/components/EmptyState'
import { FlaskConical } from 'lucide-react'
import { cn } from '@/lib/utils'

const pct = (n: number) => `${(n * 100).toFixed(1)}%`
const inputCls = 'w-full rounded-lg border bg-background px-3 py-2 text-sm outline-none focus:border-primary'

function StatusPill({ status }: { status: string }) {
  const tone = status === 'running' ? 'bg-primary-weak text-primary-text' : status === 'complete' ? 'bg-success-weak text-success' : 'bg-surface-2 text-muted-foreground'
  return <span className={cn('rounded-md px-2 py-0.5 text-xs font-semibold capitalize', tone)}>{status}</span>
}

function VariantCard({ v, winning }: { v: VariantResult; winning: boolean }) {
  return (
    <div className={cn('relative rounded-xl border p-4', winning ? 'border-success bg-success-weak' : 'bg-background')}>
      {winning && <span className="absolute right-3 top-3 rounded-md bg-success px-2 py-0.5 text-[0.68rem] font-bold text-white">WINNING</span>}
      <div className={cn('text-xs font-bold uppercase tracking-wide', winning ? 'text-success' : 'text-muted-foreground')}>Variant {v.variant.toUpperCase()}</div>
      <div className="my-1.5 font-semibold">{v.subject}</div>
      <div className="flex gap-5">
        <div><div className="text-xs text-muted-foreground">Open rate</div><div className={cn('font-mono text-2xl font-bold', winning && 'text-success')}>{pct(v.openRate)}</div></div>
        <div><div className="text-xs text-muted-foreground">Clicks</div><div className="font-mono text-2xl font-bold">{pct(v.clickRate)}</div></div>
      </div>
    </div>
  )
}

function TestCard({ t, campaignSubject }: { t: ABTest; campaignSubject: string }) {
  const results = useABTestResults(t.id, t.status !== 'draft')
  const start = useStartABTest()
  const sendWinner = useSendWinner()
  const variants = results.data?.variants ?? []
  const leadIdx = variants.length === 2 ? (variants[0].openRate >= variants[1].openRate ? 0 : 1) : -1

  return (
    <div className="rounded-2xl border bg-card p-[22px] shadow-[var(--shadow-sm)]">
      <div className="mb-4 flex flex-wrap items-center justify-between gap-2.5">
        <div>
          <div className="flex items-center gap-2.5"><strong className="text-[1.05rem]">Subject line test · {campaignSubject}</strong><StatusPill status={t.status} /></div>
          <div className="mt-0.5 text-sm text-muted-foreground">Testing on {t.testPercent}% · winner sends to the rest</div>
        </div>
        {t.status === 'draft' && (
          <Button onClick={() => start.mutate(t.id, { onSuccess: () => toast.success('Test started') })} disabled={start.isPending}>Start test</Button>
        )}
        {t.status === 'running' && leadIdx >= 0 && (
          <Button onClick={() => sendWinner.mutate({ id: t.id, winner: variants[leadIdx].variant }, { onSuccess: () => toast.success('Winner sent to the rest') })} disabled={sendWinner.isPending}>Send winner</Button>
        )}
      </div>
      {t.status === 'draft' ? (
        <div className="grid gap-4 sm:grid-cols-2">
          <VariantCard v={{ variant: 'a', subject: t.subjectA, sent: 0, openRate: 0, clickRate: 0 }} winning={false} />
          <VariantCard v={{ variant: 'b', subject: t.subjectB, sent: 0, openRate: 0, clickRate: 0 }} winning={false} />
        </div>
      ) : results.isLoading ? (
        <Skeleton className="h-32 w-full" />
      ) : (
        <div className="grid gap-4 sm:grid-cols-2">
          {variants.map((v, i) => <VariantCard key={v.variant} v={v} winning={i === leadIdx && v.sent > 0} />)}
        </div>
      )}
      <div className="mt-4 rounded-xl border border-primary-weak bg-[linear-gradient(140deg,var(--primary-weak),var(--purple-weak))] p-3 text-sm">
        <strong className="text-primary-text">AI call:</strong> The leading variant is ahead on open rate. Emoji + benefit framing tends to win for engaged lists.
      </div>
    </div>
  )
}

function NewTestDialog() {
  const [open, setOpen] = useState(false)
  const campaigns = useCampaigns()
  const lists = useLists()
  const create = useCreateABTest()
  const [form, setForm] = useState({ campaignId: '', listId: '', subjectA: '', subjectB: '', testPercent: 20 })
  const set = (p: Partial<typeof form>) => setForm((f) => ({ ...f, ...p }))
  const drafts = (campaigns.data ?? []).filter((c) => c.status === 'draft')

  const submit = () => {
    if (!form.campaignId || !form.listId || !form.subjectA || !form.subjectB) return
    create.mutate(form, { onSuccess: () => { setOpen(false); toast.success('Test created'); setForm({ campaignId: '', listId: '', subjectA: '', subjectB: '', testPercent: 20 }) } })
  }
  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={<Button className="gap-1.5"><Plus className="size-4" aria-hidden="true" />New test</Button>} />
      <DialogContent>
        <DialogHeader><DialogTitle>New A/B test</DialogTitle></DialogHeader>
        <div className="space-y-3">
          <div><Label htmlFor="ab-campaign">Campaign (draft)</Label>
            <select id="ab-campaign" className={inputCls} value={form.campaignId} onChange={(e) => set({ campaignId: e.target.value })}>
              <option value="">Select a draft campaign</option>
              {drafts.map((c) => <option key={c.id} value={c.id}>{c.subject || 'Untitled'}</option>)}
            </select></div>
          <div><Label htmlFor="ab-list">List</Label>
            <select id="ab-list" className={inputCls} value={form.listId} onChange={(e) => set({ listId: e.target.value })}>
              <option value="">Select a list</option>
              {(lists.data ?? []).map((l) => <option key={l.id} value={l.id}>{l.name}</option>)}
            </select></div>
          <div><Label htmlFor="ab-a">Subject A</Label><Input id="ab-a" value={form.subjectA} onChange={(e) => set({ subjectA: e.target.value })} /></div>
          <div><Label htmlFor="ab-b">Subject B</Label><Input id="ab-b" value={form.subjectB} onChange={(e) => set({ subjectB: e.target.value })} /></div>
          <div><Label htmlFor="ab-pct">Test percent</Label><Input id="ab-pct" type="number" value={form.testPercent} onChange={(e) => set({ testPercent: Number(e.target.value) })} /></div>
        </div>
        <DialogFooter><Button onClick={submit} disabled={create.isPending}>Create test</Button></DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

export function ABTestsPage() {
  const tests = useABTests()
  const campaigns = useCampaigns()
  const subjectOf = (cid: string) => campaigns.data?.find((c) => c.id === cid)?.subject || 'campaign'

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">A/B tests</h1>
          <p className="mt-1 text-muted-foreground">Test subject lines — send the winner to the rest of your list.</p>
        </div>
        <NewTestDialog />
      </div>
      {tests.isLoading ? (
        <Skeleton className="h-48 w-full rounded-2xl" />
      ) : !tests.data?.length ? (
        <EmptyState icon={FlaskConical} title="No A/B tests yet" description="Create a test to compare two subject lines." action={<NewTestDialog />} />
      ) : (
        <div className="space-y-4">
          {tests.data.map((t) => <TestCard key={t.id} t={t} campaignSubject={subjectOf(t.campaignId)} />)}
        </div>
      )}
    </div>
  )
}
