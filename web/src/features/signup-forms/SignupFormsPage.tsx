import { useEffect, useState } from 'react'
import { Plus, Copy } from 'lucide-react'
import { toast } from 'sonner'
import { useSignupForms, useCreateForm, useUpdateForm, useDeleteForm, type Form } from './useSignupForms'
import { useLists } from '@/features/lists/useLists'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { cn } from '@/lib/utils'

const TABS = ['Inline', 'Popup', 'Landing page'] as const
const inputCls = 'w-full rounded-lg border bg-background px-3 py-2 text-sm outline-none focus:border-primary'
const blankDraft = { id: '', listId: '', name: '', heading: 'Join 48,000 readers', description: 'Weekly insights on building in public. No spam, unsubscribe anytime.', buttonText: 'Subscribe', createdAt: '' }

export function SignupFormsPage() {
  const forms = useSignupForms()
  const lists = useLists()
  const create = useCreateForm()
  const update = useUpdateForm()
  const del = useDeleteForm()

  const [draft, setDraft] = useState<Form>(blankDraft)
  const [tab, setTab] = useState<(typeof TABS)[number]>('Landing page')

  // default the list select once lists load and nothing chosen
  useEffect(() => {
    if (!draft.listId && lists.data && lists.data.length > 0) {
      setDraft((d) => ({ ...d, listId: lists.data![0].id }))
    }
  }, [lists.data, draft.listId])

  const set = (patch: Partial<Form>) => setDraft((d) => ({ ...d, ...patch }))
  const origin = typeof window !== 'undefined' ? window.location.origin : ''
  const landingUrl = draft.id ? `${origin}/f/${draft.id}` : ''
  const embed = draft.id ? `<script src="${origin}/f/${draft.id}.js"></script>` : ''

  const save = () => {
    const body = { listId: draft.listId, name: draft.name, heading: draft.heading, description: draft.description, buttonText: draft.buttonText }
    const opts = { onSuccess: (f: Form) => { setDraft(f); toast.success('Form saved') } }
    if (draft.id) update.mutate({ id: draft.id, ...body }, opts)
    else create.mutate(body, opts)
  }
  const copy = (text: string) => { navigator.clipboard?.writeText(text); toast.success('Copied') }

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Signup forms &amp; pages</h1>
          <p className="mt-1 text-muted-foreground">Embeddable forms and hosted landing pages to grow your list.</p>
        </div>
        <div className="flex items-center gap-2">
          {(forms.data?.length ?? 0) > 0 && (
            <select aria-label="Saved forms" className={cn(inputCls, 'w-44')} value={draft.id}
              onChange={(e) => { const f = forms.data!.find((x) => x.id === e.target.value); if (f) setDraft(f) }}>
              <option value="">Current draft</option>
              {forms.data!.map((f) => <option key={f.id} value={f.id}>{f.name || 'Untitled'}</option>)}
            </select>
          )}
          <Button className="gap-1.5" onClick={() => setDraft(blankDraft)}><Plus className="size-4" aria-hidden="true" />New form</Button>
        </div>
      </div>

      <div className="grid gap-4 lg:grid-cols-2 lg:items-start">
        {/* Live preview */}
        <div className="flex min-h-[330px] items-center justify-center rounded-2xl border bg-surface-3 p-8 shadow-[var(--shadow-sm)]">
          <div className="w-full max-w-[340px] rounded-2xl bg-white p-7 shadow-[var(--shadow)]">
            <div className="mb-4 size-10 rounded-[10px] bg-[#1E40AF]" />
            <div className="text-xl font-bold text-slate-900">{draft.heading || 'Your headline'}</div>
            <p className="mb-4 mt-1.5 text-sm text-slate-500">{draft.description}</p>
            <div className="mb-2.5 rounded-lg border border-slate-200 px-3 py-2.5 text-sm text-slate-400">you@company.com</div>
            <div className="rounded-lg bg-[#D97706] py-2.5 text-center text-sm font-semibold text-white">{draft.buttonText || 'Subscribe'}</div>
            <p className="mt-3 text-center text-xs text-slate-400">Double opt-in · GDPR ready</p>
          </div>
        </div>

        {/* Editor + embed + stats */}
        <div className="flex flex-col gap-3.5">
          <div className="space-y-3 rounded-2xl border bg-card p-[18px] shadow-[var(--shadow-sm)]">
            <div className="grid gap-3 sm:grid-cols-2">
              <div className="space-y-1"><Label htmlFor="sf-name">Name</Label><Input id="sf-name" value={draft.name} onChange={(e) => set({ name: e.target.value })} /></div>
              <div className="space-y-1">
                <Label htmlFor="sf-list">List</Label>
                <select id="sf-list" className={inputCls} value={draft.listId} onChange={(e) => set({ listId: e.target.value })}>
                  <option value="">Select a list</option>
                  {(lists.data ?? []).map((l) => <option key={l.id} value={l.id}>{l.name}</option>)}
                </select>
              </div>
            </div>
            <div className="space-y-1"><Label htmlFor="sf-heading">Heading</Label><Input id="sf-heading" value={draft.heading} onChange={(e) => set({ heading: e.target.value })} /></div>
            <div className="space-y-1">
              <Label htmlFor="sf-desc">Description</Label>
              <textarea id="sf-desc" rows={2} className={cn(inputCls, 'resize-y')} value={draft.description} onChange={(e) => set({ description: e.target.value })} />
            </div>
            <div className="space-y-1"><Label htmlFor="sf-btn">Button text</Label><Input id="sf-btn" value={draft.buttonText} onChange={(e) => set({ buttonText: e.target.value })} /></div>
            <div className="flex gap-2">
              <Button onClick={save} disabled={!draft.name || !draft.listId || create.isPending || update.isPending}>{draft.id ? 'Save changes' : 'Save'}</Button>
              {draft.id && <Button variant="outline" onClick={() => del.mutate(draft.id, { onSuccess: () => { toast.success('Deleted'); setDraft(blankDraft) } })}>Delete</Button>}
            </div>
          </div>

          <div className="rounded-2xl border bg-card p-[18px] shadow-[var(--shadow-sm)]">
            <div className="mb-3 flex w-fit gap-0.5 rounded-lg bg-surface-2 p-[3px]">
              {TABS.map((t) => (
                <button key={t} type="button" onClick={() => setTab(t)}
                  className={cn('rounded-md px-3.5 py-1.5 text-sm font-medium', tab === t ? 'bg-card text-primary-text shadow-sm' : 'text-muted-foreground')}>{t}</button>
              ))}
            </div>
            {!draft.id ? (
              <p className="text-sm text-muted-foreground">Save the form to get its link.</p>
            ) : tab === 'Landing page' ? (
              <div>
                <div className="mb-2 text-xs font-bold uppercase tracking-wide text-faint">Hosted landing URL</div>
                <div className="overflow-x-auto rounded-[10px] bg-[var(--code-bg)] p-3.5 font-mono text-[0.78rem] text-[var(--code-fg)]">{landingUrl}</div>
                <Button variant="outline" size="sm" className="mt-2.5 gap-1.5" onClick={() => copy(landingUrl)}><Copy className="size-3.5" aria-hidden="true" />Copy link</Button>
              </div>
            ) : (
              <div>
                <div className="mb-2 text-xs font-bold uppercase tracking-wide text-faint">Embed code <span className="text-muted-foreground">(preview only)</span></div>
                <div className="overflow-x-auto rounded-[10px] bg-[var(--code-bg)] p-3.5 font-mono text-[0.78rem] text-[var(--code-fg)]">{embed}</div>
                <Button variant="outline" size="sm" className="mt-2.5 gap-1.5" onClick={() => copy(embed)}><Copy className="size-3.5" aria-hidden="true" />Copy embed</Button>
              </div>
            )}
          </div>

          {/* ponytail: sample stats until view/conversion tracking lands */}
          <div className="grid grid-cols-2 gap-3.5">
            <div className="rounded-2xl border bg-card p-4 shadow-[var(--shadow-sm)]">
              <div className="text-sm text-muted-foreground">Views (30d)</div>
              <div className="mt-1 font-mono text-2xl font-bold">18,402</div>
            </div>
            <div className="rounded-2xl border bg-card p-4 shadow-[var(--shadow-sm)]">
              <div className="text-sm text-muted-foreground">Conversion</div>
              <div className="mt-1 font-mono text-2xl font-bold text-success">7.8%</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
