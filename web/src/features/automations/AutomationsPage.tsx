import { useState } from 'react'
import { Plus, Zap, Mail, Clock, Trash2, Workflow } from 'lucide-react'
import { toast } from 'sonner'
import {
  useAutomations, useCreateAutomation, useReplaceSteps, useSetStatus,
  type Automation, type Step,
} from './useAutomations'
import { useLists } from '@/features/lists/useLists'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Dialog, DialogTrigger, DialogContent, DialogHeader, DialogTitle, DialogFooter,
} from '@/components/ui/dialog'
import { Skeleton } from '@/components/ui/skeleton'
import { EmptyState } from '@/components/EmptyState'
import { cn } from '@/lib/utils'

const pct = (n: number) => `${(n * 100).toFixed(0)}%`
const inputCls = 'w-full rounded-lg border bg-background px-3 py-2 text-sm outline-none focus:border-primary'

function StatusPill({ status }: { status: string }) {
  const tone = status === 'live' ? 'bg-success-weak text-success' : status === 'paused' ? 'bg-surface-2 text-muted-foreground' : 'bg-surface-2 text-muted-foreground'
  return <span className={cn('rounded-md px-2 py-0.5 text-xs font-semibold capitalize', tone)}>{status}</span>
}

function Card({ a, selected, onSelect }: { a: Automation; selected: boolean; onSelect: () => void }) {
  return (
    <button type="button" onClick={onSelect}
      className={cn('rounded-2xl border bg-card p-[18px] text-left shadow-[var(--shadow-sm)] hover:border-primary', selected && 'border-primary ring-1 ring-primary')}>
      <div className="mb-2.5 flex items-center justify-between"><span className="font-bold">{a.name}</span><StatusPill status={a.status} /></div>
      <p className="mb-3 text-sm text-muted-foreground">{a.stepSends} email{a.stepSends === 1 ? '' : 's'} · triggered on subscribe</p>
      <div className="flex gap-4 text-sm">
        <span><strong className="font-mono">{a.inFlow.toLocaleString()}</strong> <span className="text-muted-foreground">in flow</span></span>
        <span><strong className="font-mono text-primary-text">{pct(a.completePct)}</strong> <span className="text-muted-foreground">complete</span></span>
      </div>
    </button>
  )
}

function Connector() { return <div className="h-6 w-0.5 bg-border-strong" /> }

function StepCard({ step, onChange, onRemove }: { step: Step; onChange: (s: Step) => void; onRemove: () => void }) {
  const isSend = step.kind === 'send'
  return (
    <div className={cn('w-full rounded-xl border-[1.5px] bg-card p-3.5 shadow-[var(--shadow-sm)]', isSend ? 'border-primary' : 'border-border-strong')}>
      <div className="flex items-center gap-2.5">
        <span className={cn('flex size-8 items-center justify-center rounded-lg', isSend ? 'bg-primary-weak text-primary-text' : 'bg-surface-2 text-muted-foreground')}>
          {isSend ? <Mail className="size-4" aria-hidden="true" /> : <Clock className="size-4" aria-hidden="true" />}
        </span>
        <div className="min-w-0 flex-1">
          <div className={cn('text-[0.7rem] font-bold uppercase tracking-wide', isSend ? 'text-primary-text' : 'text-muted-foreground')}>{isSend ? 'Send email' : 'Wait'}</div>
          {isSend ? (
            <p className="text-sm font-semibold">{step.subject || <span className="italic text-muted-foreground">Subject…</span>}</p>
          ) : (
            <div className="flex items-center gap-1.5 text-sm font-semibold">
              <input aria-label="Wait days" type="number" min={1} className="w-16 rounded border bg-background px-1.5 py-0.5" value={step.waitDays} onChange={(e) => onChange({ ...step, waitDays: Number(e.target.value) })} /> days
            </div>
          )}
        </div>
        <button type="button" aria-label="Remove step" className="text-muted-foreground hover:text-danger" onClick={onRemove}><Trash2 className="size-4" aria-hidden="true" /></button>
      </div>
      {isSend && (
        <>
          <input aria-label="Step subject" className={cn(inputCls, 'mt-2')} value={step.subject} placeholder="Subject…" onChange={(e) => onChange({ ...step, subject: e.target.value })} />
          <textarea aria-label="Step html" rows={2} className={cn(inputCls, 'mt-2 resize-y font-mono text-xs')} value={step.html} placeholder="<p>Email body…</p>" onChange={(e) => onChange({ ...step, html: e.target.value })} />
        </>
      )}
    </div>
  )
}

function Builder({ a, listName }: { a: Automation; listName: string }) {
  const replace = useReplaceSteps()
  const setStatus = useSetStatus()
  const [steps, setSteps] = useState<Step[]>(a.steps)

  const setStep = (i: number, s: Step) => setSteps((cur) => cur.map((x, j) => (j === i ? s : x)))
  const addStep = (kind: string) => setSteps((cur) => [...cur, kind === 'send' ? { kind: 'send', subject: '', html: '', waitDays: 0 } : { kind: 'wait', subject: '', html: '', waitDays: 1 }])
  const save = () => replace.mutate({ id: a.id, steps }, { onSuccess: () => toast.success('Saved'), onError: () => toast.error('Check each step (subject / days)') })

  return (
    <div className="overflow-hidden rounded-2xl border bg-card shadow-[var(--shadow-sm)]">
      <div className="flex items-center justify-between border-b px-5 py-3.5">
        <div className="flex items-center gap-2.5"><strong>{a.name}</strong><span className="text-sm text-muted-foreground">· visual editor</span></div>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" onClick={save} disabled={replace.isPending}>Save</Button>
          {a.status === 'live'
            ? <Button size="sm" onClick={() => setStatus.mutate({ id: a.id, status: 'paused' }, { onSuccess: () => toast.success('Paused') })}>Pause</Button>
            : <Button size="sm" onClick={() => setStatus.mutate({ id: a.id, status: 'live' }, { onSuccess: () => toast.success('Published') })}>Publish</Button>}
        </div>
      </div>
      <div className="flex justify-center bg-surface-3 px-5 py-8 [background-image:radial-gradient(var(--border-strong)_1.1px,transparent_1.1px)] [background-size:20px_20px]">
        <div className="flex w-full max-w-[360px] flex-col items-center">
          <div className="w-full rounded-xl border-[1.5px] border-success bg-card p-3.5 shadow-[var(--shadow)]">
            <div className="flex items-center gap-2.5">
              <span className="flex size-8 items-center justify-center rounded-lg bg-success-weak text-success"><Zap className="size-4" aria-hidden="true" /></span>
              <div><div className="text-[0.7rem] font-bold uppercase tracking-wide text-success">Trigger</div><div className="text-sm font-semibold">Subscriber joins {listName}</div></div>
            </div>
          </div>
          {steps.map((s, i) => (
            <div key={i} className="flex w-full flex-col items-center">
              <Connector />
              <StepCard step={s} onChange={(ns) => setStep(i, ns)} onRemove={() => setSteps((cur) => cur.filter((_, j) => j !== i))} />
            </div>
          ))}
          <Connector />
          <div className="flex gap-2">
            <button type="button" onClick={() => addStep('send')} className="inline-flex items-center gap-1.5 rounded-lg border-[1.5px] border-dashed border-border-strong px-3 py-2 text-sm font-semibold text-muted-foreground hover:border-primary hover:text-primary-text"><Plus className="size-3.5" aria-hidden="true" />Email</button>
            <button type="button" onClick={() => addStep('wait')} className="inline-flex items-center gap-1.5 rounded-lg border-[1.5px] border-dashed border-border-strong px-3 py-2 text-sm font-semibold text-muted-foreground hover:border-primary hover:text-primary-text"><Plus className="size-3.5" aria-hidden="true" />Wait</button>
          </div>
        </div>
      </div>
    </div>
  )
}

function NewAutomationDialog() {
  const [open, setOpen] = useState(false)
  const lists = useLists()
  const create = useCreateAutomation()
  const [name, setName] = useState('')
  const [listId, setListId] = useState('')
  const submit = () => {
    if (!name || !listId) return
    create.mutate({ name, listId }, { onSuccess: () => { setOpen(false); setName(''); toast.success('Automation created') } })
  }
  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={<Button className="gap-1.5"><Plus className="size-4" aria-hidden="true" />New automation</Button>} />
      <DialogContent>
        <DialogHeader><DialogTitle>New automation</DialogTitle></DialogHeader>
        <div className="space-y-3">
          <div><Label htmlFor="auto-name">Name</Label><Input id="auto-name" value={name} onChange={(e) => setName(e.target.value)} /></div>
          <div><Label htmlFor="auto-list">Trigger list</Label>
            <select id="auto-list" className={inputCls} value={listId} onChange={(e) => setListId(e.target.value)}>
              <option value="">Select a list</option>
              {(lists.data ?? []).map((l) => <option key={l.id} value={l.id}>{l.name}</option>)}
            </select></div>
        </div>
        <DialogFooter><Button onClick={submit} disabled={create.isPending}>Create</Button></DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

export function AutomationsPage() {
  const autos = useAutomations()
  const lists = useLists()
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const listName = (id: string) => lists.data?.find((l) => l.id === id)?.name || 'the list'
  const selected = autos.data?.find((a) => a.id === selectedId) ?? null

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Automations</h1>
          <p className="mt-1 text-muted-foreground">Trigger-based journeys that run on autopilot.</p>
        </div>
        <NewAutomationDialog />
      </div>
      {autos.isLoading ? (
        <Skeleton className="h-40 w-full rounded-2xl" />
      ) : !autos.data?.length ? (
        <EmptyState icon={Workflow} title="No automations yet" description="Create a welcome series triggered when someone joins a list." action={<NewAutomationDialog />} />
      ) : (
        <>
          <div className="grid gap-3.5 [grid-template-columns:repeat(auto-fit,minmax(260px,1fr))]">
            {autos.data.map((a) => <Card key={a.id} a={a} selected={a.id === selectedId} onSelect={() => setSelectedId(a.id)} />)}
          </div>
          {selected && <Builder key={selected.id} a={selected} listName={listName(selected.listId)} />}
        </>
      )}
    </div>
  )
}
