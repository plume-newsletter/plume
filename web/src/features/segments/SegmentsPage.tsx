import { useEffect, useRef, useState } from 'react'
import { Plus, Trash2, Sparkles, X } from 'lucide-react'
import { toast } from 'sonner'
import {
  useSegments, useSegmentFields, useSegmentPreview, useCreateSegment, useUpdateSegment, useDeleteSegment,
  type Condition, type Segment,
} from './useSegments'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Dialog, DialogTrigger, DialogContent, DialogHeader, DialogTitle, DialogFooter,
} from '@/components/ui/dialog'
import { cn } from '@/lib/utils'

const pct = (n: number) => `${(n * 100).toFixed(1)}%`
const TYPES = [
  { v: 'opened', label: 'Opened email' },
  { v: 'clicked', label: 'Clicked link' },
  { v: 'field', label: 'Custom field' },
  { v: 'status', label: 'Status' },
]
const OPS: Record<string, { v: string; label: string }[]> = {
  opened: [{ v: 'in_last', label: 'in last' }, { v: 'ever', label: 'ever' }, { v: 'never', label: 'never' }],
  clicked: [{ v: 'in_last', label: 'in last' }, { v: 'ever', label: 'ever' }, { v: 'never', label: 'never' }],
  field: [{ v: 'equals', label: 'equals' }, { v: 'not_equals', label: 'not equals' }, { v: 'contains', label: 'contains' }],
  status: [{ v: 'is', label: 'is' }, { v: 'is_not', label: 'is not' }],
}
const STATUSES = ['active', 'pending', 'unsubscribed']
const selCls = 'rounded-lg border bg-background px-2.5 py-2 text-sm outline-none focus:border-primary'

function defaultCond(type: string): Condition {
  if (type === 'field') return { type, op: 'equals', field: '', value: '' }
  if (type === 'status') return { type, op: 'is', value: 'active' }
  return { type, op: 'in_last', days: 30 }
}

export function SegmentsPage() {
  const segs = useSegments()
  const fields = useSegmentFields()
  const preview = useSegmentPreview()
  const create = useCreateSegment()
  const update = useUpdateSegment()
  const del = useDeleteSegment()

  const [match, setMatch] = useState<'all' | 'any'>('all')
  const [conds, setConds] = useState<Condition[]>([])
  const [editingId, setEditingId] = useState<string | null>(null)
  const [data, setData] = useState<{ count: number; total: number; percent: number } | null>(null)
  const [saveOpen, setSaveOpen] = useState(false)
  const [name, setName] = useState('')

  // debounced live preview
  const previewMutate = preview.mutate
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null)
  useEffect(() => {
    if (timer.current) clearTimeout(timer.current)
    timer.current = setTimeout(() => {
      previewMutate({ match, conditions: conds }, { onSuccess: (p) => setData(p) })
    }, 400)
    return () => { if (timer.current) clearTimeout(timer.current) }
  }, [match, conds, previewMutate])

  const setCond = (i: number, patch: Partial<Condition>) =>
    setConds((cur) => cur.map((c, j) => (j === i ? { ...c, ...patch } : c)))

  const loadSegment = (s: Segment) => {
    setEditingId(s.id); setName(s.name)
    setMatch((s.match === 'any' ? 'any' : 'all')); setConds(s.conditions)
  }
  const resetBuilder = () => { setEditingId(null); setName(''); setMatch('all'); setConds([]) }

  const doSave = () => {
    if (!name.trim()) return
    const onDone = () => { setSaveOpen(false); toast.success('Segment saved'); if (!editingId) resetBuilder() }
    if (editingId) update.mutate({ id: editingId, name, match, conditions: conds }, { onSuccess: onDone })
    else create.mutate({ name, match, conditions: conds }, { onSuccess: (s) => { setEditingId(s.id); onDone() } })
  }

  return (
    <div className="space-y-5">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">Segments</h1>
        <p className="mt-1 text-muted-foreground">Dynamic groups that update as subscribers behave.</p>
      </div>

      <div className="grid gap-4 lg:grid-cols-[1.1fr_1fr] lg:items-start">
        {/* Builder */}
        <div className="rounded-2xl border bg-card p-5 shadow-[var(--shadow-sm)]">
          <div className="mb-4 flex items-center justify-between">
            <h3 className="text-base font-bold">Build a segment</h3>
            <label className="text-sm text-muted-foreground">
              Match{' '}
              <select aria-label="Match" value={match} onChange={(e) => setMatch(e.target.value as 'all' | 'any')} className={selCls}>
                <option value="all">all</option>
                <option value="any">any</option>
              </select>{' '}
              conditions
            </label>
          </div>

          <div className="flex flex-col gap-2.5">
            {conds.map((c, i) => (
              <div key={i} className="flex flex-wrap items-center gap-2 rounded-xl border bg-background p-3">
                <select aria-label={`type ${i}`} value={c.type} className={selCls}
                  onChange={(e) => setConds((cur) => cur.map((x, j) => (j === i ? defaultCond(e.target.value) : x)))}>
                  {TYPES.map((t) => <option key={t.v} value={t.v}>{t.label}</option>)}
                </select>
                <select aria-label={`op ${i}`} value={c.op} className={selCls} onChange={(e) => setCond(i, { op: e.target.value })}>
                  {OPS[c.type].map((o) => <option key={o.v} value={o.v}>{o.label}</option>)}
                </select>
                {(c.type === 'opened' || c.type === 'clicked') && c.op === 'in_last' && (
                  <span className="flex items-center gap-1.5">
                    <input aria-label={`days ${i}`} type="number" value={c.days ?? 30} className={cn(selCls, 'w-20')}
                      onChange={(e) => setCond(i, { days: Number(e.target.value) })} />
                    <span className="text-sm text-muted-foreground">days</span>
                  </span>
                )}
                {c.type === 'field' && (
                  <>
                    <select aria-label={`field ${i}`} value={c.field ?? ''} className={selCls} onChange={(e) => setCond(i, { field: e.target.value })}>
                      <option value="">field…</option>
                      {(fields.data ?? []).map((f) => <option key={f} value={f}>{f}</option>)}
                    </select>
                    <input aria-label={`value ${i}`} value={c.value ?? ''} placeholder="value" className={cn(selCls, 'flex-1')} onChange={(e) => setCond(i, { value: e.target.value })} />
                  </>
                )}
                {c.type === 'status' && (
                  <select aria-label={`status ${i}`} value={c.value ?? 'active'} className={selCls} onChange={(e) => setCond(i, { value: e.target.value })}>
                    {STATUSES.map((s) => <option key={s} value={s}>{s}</option>)}
                  </select>
                )}
                <button type="button" aria-label={`remove ${i}`} className="ml-auto text-muted-foreground hover:text-danger"
                  onClick={() => setConds((cur) => cur.filter((_, j) => j !== i))}>
                  <Trash2 className="size-4" aria-hidden="true" />
                </button>
              </div>
            ))}
            <button type="button"
              onClick={() => setConds((cur) => [...cur, defaultCond('opened')])}
              className="inline-flex w-fit items-center gap-1.5 rounded-lg border-[1.5px] border-dashed border-border-strong px-3 py-2 text-sm font-semibold text-muted-foreground hover:border-primary hover:text-primary-text">
              <Plus className="size-3.5" aria-hidden="true" /> Add condition
            </button>
          </div>

          {/* ponytail: Ask-AI is a placeholder until the NL->rules endpoint ships */}
          <div className="mt-4 rounded-xl border border-primary-weak bg-[linear-gradient(140deg,var(--primary-weak),var(--purple-weak))] p-3 text-sm">
            <Sparkles className="mr-1.5 inline size-3.5 text-primary-text" aria-hidden="true" />
            <strong className="text-primary-text">Ask AI:</strong> describe your audience in plain English — coming soon.
          </div>
        </div>

        {/* Preview + saved */}
        <div className="flex flex-col gap-4">
          <div className="rounded-2xl border bg-card p-5 text-center shadow-[var(--shadow-sm)]">
            <div className="text-sm text-muted-foreground">Matching subscribers</div>
            <div className="my-1.5 font-mono text-4xl font-bold text-primary">{(data?.count ?? 0).toLocaleString()}</div>
            <div className="text-xs text-muted-foreground">{pct(data?.percent ?? 0)} of all subscribers · updates live</div>
            <Dialog open={saveOpen} onOpenChange={setSaveOpen}>
              <DialogTrigger render={<Button className="mt-3.5">{editingId ? 'Update segment' : 'Save segment'}</Button>} />
              <DialogContent>
                <DialogHeader><DialogTitle>{editingId ? 'Update segment' : 'Save segment'}</DialogTitle></DialogHeader>
                <div className="space-y-1">
                  <Label htmlFor="seg-name">Segment name</Label>
                  <Input id="seg-name" value={name} onChange={(e) => setName(e.target.value)} />
                </div>
                <DialogFooter>
                  <Button onClick={doSave} disabled={!name.trim() || create.isPending || update.isPending}>Save</Button>
                </DialogFooter>
              </DialogContent>
            </Dialog>
          </div>

          <div className="overflow-hidden rounded-2xl border bg-card shadow-[var(--shadow-sm)]">
            <div className="flex items-center justify-between border-b px-[18px] py-3.5">
              <span className="text-sm font-bold">Saved segments</span>
              {editingId && (
                <button type="button" onClick={resetBuilder} className="inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground">
                  <X className="size-3.5" aria-hidden="true" /> New
                </button>
              )}
            </div>
            {(segs.data ?? []).length === 0 && <div className="px-[18px] py-6 text-center text-sm text-muted-foreground">No saved segments yet.</div>}
            {(segs.data ?? []).map((s) => (
              <div key={s.id} className={cn('flex items-center justify-between border-b px-[18px] py-3 text-sm last:border-b-0 hover:bg-surface-2',
                editingId === s.id && 'bg-surface-2')}>
                <button type="button" className="min-w-0 flex-1 text-left font-semibold" onClick={() => loadSegment(s)}>
                  <span className="truncate">{s.name}</span>
                </button>
                <span className="font-mono text-muted-foreground">{s.count.toLocaleString()}</span>
                <button type="button" aria-label={`delete ${s.name}`} className="ml-3 text-muted-foreground hover:text-danger"
                  onClick={() => del.mutate(s.id, { onSuccess: () => { toast.success('Deleted'); if (editingId === s.id) resetBuilder() } })}>
                  <Trash2 className="size-4" aria-hidden="true" />
                </button>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}
