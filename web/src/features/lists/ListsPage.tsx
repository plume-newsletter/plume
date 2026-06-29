import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { List, Plus, Upload, Search } from 'lucide-react'
import { toast } from 'sonner'
import { useLists, useCreateList } from './useLists'
import { useBrands } from '@/features/brands/useBrands'
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

const createSchema = z.object({ name: z.string().min(1) })

function NewListDialog({ trigger }: { trigger: React.ReactElement }) {
  const [open, setOpen] = useState(false)
  const [brandId, setBrandId] = useState('')
  const { data: brands } = useBrands()
  const create = useCreateList()
  const { register, handleSubmit, reset, formState: { errors } } = useForm<{ name: string }>({
    resolver: zodResolver(createSchema),
    defaultValues: { name: '' },
  })
  const onSubmit = (data: { name: string }) => {
    if (!brandId) return
    create.mutate({ brandId, name: data.name }, {
      onSuccess: () => { setOpen(false); reset(); setBrandId(''); toast.success('List created') },
      onError: () => toast.error('Something went wrong'),
    })
  }
  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={trigger} />
      <DialogContent>
        <DialogHeader><DialogTitle>New list</DialogTitle></DialogHeader>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-3">
          <div>
            <Label htmlFor="brand">Brand</Label>
            <Select value={brandId} onValueChange={(v) => setBrandId(v ?? '')}>
              <SelectTrigger id="brand" className="w-full">
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
            <Label htmlFor="name">Name</Label>
            <Input id="name" {...register('name')} />
            {errors.name && <p className="text-sm text-destructive">Required</p>}
          </div>
          <DialogFooter>
            <Button type="submit" disabled={!brandId}>Create</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

// ponytail: per-list contact counts and the subscriber table are sample until the
// audience/segments feature lands a subscribers endpoint; the list rail is real.
const SAMPLE_SUBS = [
  { initials: 'SC', name: 'Sofia Chen', email: 'sofia.chen@gmail.com', status: 'Subscribed', joined: 'Jan 14, 2025', eng: 86, tone: 'success' },
  { initials: 'MR', name: 'Marcus Reyes', email: 'm.reyes@orbit.test', status: 'Subscribed', joined: 'Feb 02, 2025', eng: 64, tone: 'success' },
  { initials: 'AK', name: 'Aria Kim', email: 'aria@studio.dev', status: 'Subscribed', joined: 'Feb 20, 2025', eng: 41, tone: 'amber' },
  { initials: 'TP', name: 'Theo Park', email: 'theo.park@mail.com', status: 'Unsubscribed', joined: 'Mar 03, 2025', eng: 8, tone: 'danger' },
  { initials: 'LN', name: 'Lena Novak', email: 'lena@novak.io', status: 'Subscribed', joined: 'Mar 18, 2025', eng: 72, tone: 'success' },
]
const engColor = (t: string) => (t === 'success' ? 'bg-success' : t === 'amber' ? 'bg-amber' : 'bg-danger')
const statusTone = (s: string) => (s === 'Subscribed' ? 'bg-success-weak text-success' : 'bg-danger-weak text-danger')

const SUB_COLS = 'grid grid-cols-[28px_2fr_1fr_1fr_0.9fr] gap-2.5 px-[18px]'

export function ListsPage() {
  const { data: lists, isLoading } = useLists()
  const { data: brands } = useBrands()
  const navigate = useNavigate()
  const brandName = (brandId: string) => brands?.find((b) => b.id === brandId)?.name ?? '—'

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Lists &amp; subscribers</h1>
          <p className="mt-1 text-muted-foreground">
            {lists?.length ? `Across ${lists.length} list${lists.length > 1 ? 's' : ''}.` : 'Organize contacts into lists under each brand.'}
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" className="gap-1.5"><Upload className="size-4" aria-hidden="true" />Import CSV</Button>
          <NewListDialog trigger={<Button className="gap-1.5"><Plus className="size-4" aria-hidden="true" />Add subscriber</Button>} />
        </div>
      </div>

      {isLoading ? (
        <Skeleton className="h-64 w-full rounded-2xl" />
      ) : !lists?.length ? (
        <EmptyState icon={List} title="No lists yet"
          description="Create a list to collect subscribers."
          action={<NewListDialog trigger={<Button>New list</Button>} />} />
      ) : (
        <div className="grid gap-4 lg:grid-cols-[230px_1fr] lg:items-start">
          {/* List rail (real) */}
          <div className="rounded-2xl border bg-card p-2 shadow-[var(--shadow-sm)]">
            <div className="flex items-center justify-between rounded-lg bg-sidebar-active px-2.5 py-1.5 text-sm font-semibold text-primary-text">
              All subscribers <span className="font-mono text-xs">—</span>
            </div>
            {lists.map((l) => (
              <button key={l.id} type="button" onClick={() => navigate(`/lists/${l.id}`)}
                className="flex w-full items-center justify-between gap-2 rounded-lg px-2.5 py-1.5 text-left text-sm hover:bg-surface-2">
                <span className="min-w-0">
                  <span className="block truncate">{l.name}</span>
                  <span className="block truncate text-xs text-muted-foreground">{brandName(l.brand_id)}</span>
                </span>
              </button>
            ))}
            <div className="mx-1 my-1.5 border-t" />
            <NewListDialog trigger={
              <button type="button" className="flex w-full items-center gap-1.5 rounded-lg px-2.5 py-1.5 text-left text-sm font-semibold text-primary-text hover:bg-surface-2">
                <Plus className="size-3.5" aria-hidden="true" />New list
              </button>
            } />
          </div>

          {/* Subscriber table (sample) */}
          <div className="overflow-hidden rounded-2xl border bg-card shadow-[var(--shadow-sm)]">
            <div className="flex items-center gap-2.5 border-b px-[18px] py-3">
              <div className="relative flex flex-1 items-center">
                <Search className="pointer-events-none absolute left-3 size-3.5 text-faint" aria-hidden="true" />
                <input placeholder="Search by email or name" aria-label="Search subscribers"
                  className="w-full rounded-lg border bg-background py-2 pl-8 pr-3 text-sm outline-none focus:border-primary focus:ring-2 focus:ring-primary-weak" />
              </div>
            </div>
            <div className={cn(SUB_COLS, 'border-b py-2.5 text-xs font-semibold uppercase tracking-wide text-faint')}>
              <span /><span>Subscriber</span><span>Status</span><span>Joined</span><span>Engaged</span>
            </div>
            {SAMPLE_SUBS.map((s) => (
              <div key={s.email} className={cn(SUB_COLS, 'items-center border-b py-3 last:border-b-0')}>
                <span className="flex size-7 items-center justify-center rounded-full bg-primary-weak text-[0.7rem] font-bold text-primary-text">{s.initials}</span>
                <span className="min-w-0">
                  <span className="block truncate text-sm font-semibold">{s.name}</span>
                  <span className="block truncate font-mono text-xs text-muted-foreground">{s.email}</span>
                </span>
                <span><span className={cn('rounded-md px-2 py-0.5 text-xs font-semibold', statusTone(s.status))}>{s.status}</span></span>
                <span className="font-mono text-xs text-muted-foreground">{s.joined}</span>
                <span><span className="block h-1.5 max-w-[48px] rounded-full bg-surface-2"><span className={cn('block h-full rounded-full', engColor(s.tone))} style={{ width: `${s.eng}%` }} /></span></span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
