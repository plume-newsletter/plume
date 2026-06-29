import { useEffect, useMemo, useRef, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import {
  FileText, ChevronLeft, Monitor, Smartphone, Eye, Send, Sparkles, X,
  Heading, Type, Image as ImageIcon, RectangleHorizontal, Columns2, Minus,
  StretchVertical, Share2, Code, BookmarkPlus,
} from 'lucide-react'
import { toast } from 'sonner'
import { useCampaign, useUpdateCampaign } from './useCampaign'
import { SendDialog } from './SendDialog'
import { BlockCanvas } from './BlockCanvas'
import { BlockInspector } from './BlockInspector'
import { newBlock, addBlock, removeBlock, updateBlock, type Block, type BlockType } from './blocks'
import { useRenderPreview } from './useRenderPreview'
import { useCreateTemplate } from '@/features/templates/useTemplates'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { EmptyState } from '@/components/EmptyState'
import {
  Dialog, DialogTrigger, DialogContent, DialogHeader, DialogTitle, DialogFooter,
} from '@/components/ui/dialog'
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from '@/components/ui/select'
import { cn } from '@/lib/utils'

const TEMPLATE_CATEGORIES = ['Newsletter', 'Product', 'Promo', 'Transactional'] as const

function SaveTemplateDialog({ blocks, subject, trigger }: {
  blocks: Block[]
  subject: string
  trigger: React.ReactElement
}) {
  const [open, setOpen] = useState(false)
  const [name, setName] = useState('')
  const [category, setCategory] = useState('Newsletter')
  const createTpl = useCreateTemplate()

  const handleOpenChange = (next: boolean) => {
    if (next) setName(subject || 'My template')
    setOpen(next)
  }

  const onSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!name.trim()) return
    createTpl.mutate(
      { name: name.trim(), category, bodyJson: blocks },
      {
        onSuccess: () => { setOpen(false); toast.success('Template saved') },
        onError: () => toast.error('Something went wrong'),
      },
    )
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger render={trigger} />
      <DialogContent>
        <DialogHeader><DialogTitle>Save as template</DialogTitle></DialogHeader>
        <form onSubmit={onSubmit} className="space-y-3">
          <div>
            <Label htmlFor="tpl-name">Name</Label>
            <Input id="tpl-name" value={name} onChange={(e) => setName(e.target.value)} />
          </div>
          <div>
            <Label htmlFor="tpl-category">Category</Label>
            <Select value={category} onValueChange={(v) => setCategory(v ?? 'Newsletter')}>
              <SelectTrigger id="tpl-category" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {TEMPLATE_CATEGORIES.map((c) => (
                  <SelectItem key={c} value={c}>{c}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <DialogFooter>
            <Button type="submit" disabled={!name.trim() || createTpl.isPending}>Save template</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

const PALETTE: { type: BlockType; label: string; Icon: typeof Heading }[] = [
  { type: 'heading', label: 'Heading', Icon: Heading },
  { type: 'text', label: 'Text', Icon: Type },
  { type: 'image', label: 'Image', Icon: ImageIcon },
  { type: 'button', label: 'Button', Icon: RectangleHorizontal },
  { type: 'columns', label: 'Columns', Icon: Columns2 },
  { type: 'divider', label: 'Divider', Icon: Minus },
  { type: 'spacer', label: 'Spacer', Icon: StretchVertical },
  { type: 'social', label: 'Social', Icon: Share2 },
  { type: 'html', label: 'HTML', Icon: Code },
]

function seedBlocks(bodyJson: string, htmlBody: string): Block[] {
  try {
    const parsed = JSON.parse(bodyJson || '[]') as Block[]
    if (Array.isArray(parsed) && parsed.length > 0) return parsed
  } catch { /* fall through */ }
  if (htmlBody) return [newBlock('html')].map((b) => ({ ...b, html: htmlBody }))
  return []
}

export function ComposePage() {
  const { id } = useParams<{ id: string }>()
  const { data, isLoading, isError } = useCampaign(id!)
  const update = useUpdateCampaign(id!)
  const preview = useRenderPreview()

  const [subject, setSubject] = useState('')
  const [blocks, setBlocks] = useState<Block[]>([])
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [device, setDevice] = useState<'desktop' | 'mobile'>('desktop')
  const [previewHtml, setPreviewHtml] = useState('')
  const [showPreview, setShowPreview] = useState(false)

  const savedSnapshot = useRef<string | null>(null)

  useEffect(() => {
    if (data) {
      setSubject(data.subject)
      const seeded = seedBlocks(data.body_json, data.html_body)
      setBlocks(seeded)
      if (seeded.length > 0) setSelectedId(seeded[0].id)
      savedSnapshot.current = JSON.stringify({ subject: data.subject, blocks: seeded })
    }
  }, [data?.id]) // eslint-disable-line react-hooks/exhaustive-deps

  // Debounced auto-save: persist whenever the draft changes after hydration.
  const isDraft = data?.status === 'draft'
  const updateMutate = update.mutate
  useEffect(() => {
    if (savedSnapshot.current === null || !isDraft) return
    const snapshot = JSON.stringify({ subject, blocks })
    if (snapshot === savedSnapshot.current) return
    const t = setTimeout(() => {
      updateMutate(
        { subject, bodyJson: JSON.stringify(blocks) },
        { onSuccess: () => { savedSnapshot.current = snapshot } },
      )
    }, 700)
    return () => clearTimeout(t)
  }, [subject, blocks, isDraft, updateMutate])

  const selected = useMemo(() => blocks.find((b) => b.id === selectedId) ?? null, [blocks, selectedId])

  const openPreview = () =>
    preview.mutate(blocks, { onSuccess: ({ html }) => { setPreviewHtml(html); setShowPreview(true) } })

  if (isLoading) {
    return (
      <div className="flex h-[calc(100dvh-4rem)] gap-px">
        <Skeleton className="h-full w-[236px] rounded-none" />
        <Skeleton className="h-full flex-1 rounded-none" />
        <Skeleton className="h-full w-[296px] rounded-none" />
      </div>
    )
  }
  if (isError || !data) {
    return (
      <div className="p-6">
        <EmptyState icon={FileText} title="Campaign not found"
          description="This campaign does not exist or could not be loaded."
          action={<Link to="/campaigns"><Button variant="outline">Back to campaigns</Button></Link>} />
      </div>
    )
  }

  const status = update.isPending ? 'Saving…' : 'Auto-saved · just now'

  return (
    <div className="flex h-[calc(100dvh-4rem)] flex-col">
      {/* Builder toolbar */}
      <div className="flex flex-wrap items-center gap-3.5 border-b bg-surface px-5 py-2.5">
        <Link to="/campaigns"
          className="inline-flex items-center gap-1.5 rounded-lg border bg-background px-2.5 py-1.5 text-sm font-medium hover:bg-surface-2">
          <ChevronLeft className="size-3.5" aria-hidden="true" /> Back
        </Link>
        <div className="min-w-0">
          <input
            aria-label="Subject"
            value={subject}
            onChange={(e) => setSubject(e.target.value)}
            className="w-[230px] max-w-full rounded-md border border-transparent bg-transparent px-2 py-1 text-base font-bold outline-none hover:border-border focus:border-primary focus:bg-surface"
          />
          <div className="flex items-center gap-1.5 pl-2 text-xs text-muted-foreground">
            <span className={cn('size-1.5 rounded-full', update.isPending ? 'bg-amber' : 'bg-success')} />
            {status}
          </div>
        </div>

        <div className="ml-auto flex gap-0.5 rounded-lg bg-surface-2 p-[3px]">
          {([['desktop', Monitor, 'Desktop'], ['mobile', Smartphone, 'Mobile']] as const).map(([d, Icon, label]) => (
            <button key={d} type="button" onClick={() => setDevice(d)}
              className={cn('inline-flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-semibold',
                device === d ? 'bg-card shadow-sm' : 'text-muted-foreground')}>
              <Icon className="size-3.5" aria-hidden="true" /> {label}
            </button>
          ))}
        </div>
        <Button type="button" variant="outline" onClick={openPreview} disabled={preview.isPending} className="gap-1.5">
          <Eye className="size-3.5" aria-hidden="true" /> Preview
        </Button>
        {/* ponytail: test send is a placeholder until the send pipeline exposes it */}
        <Button type="button" variant="outline" className="gap-1.5">
          <Send className="size-3.5" aria-hidden="true" /> Send test
        </Button>
        <SaveTemplateDialog
          blocks={blocks}
          subject={subject}
          trigger={
            <Button type="button" variant="outline" className="gap-1.5">
              <BookmarkPlus className="size-3.5" aria-hidden="true" /> Save as template
            </Button>
          }
        />
        <SendDialog
          campaignId={id!}
          disabled={data.status !== 'draft'}
          label="Schedule"
          icon={<Send className="size-3.5" aria-hidden="true" />}
          triggerClassName="gap-1.5 bg-amber text-white hover:brightness-95"
        />
      </div>

      {/* 3-pane builder */}
      <div className="grid min-h-0 flex-1 grid-cols-[236px_1fr_296px]">
        {/* Palette */}
        <div className="overflow-y-auto border-r bg-surface p-4">
          <div className="mb-3 text-xs font-bold uppercase tracking-wide text-faint">Content blocks</div>
          <div className="grid grid-cols-2 gap-2.5">
            {PALETTE.map(({ type, label, Icon }) => (
              <button
                key={type}
                type="button"
                aria-label={`Add ${label}`}
                onClick={() => {
                  const b = newBlock(type)
                  setBlocks((cur) => addBlock(cur, b))
                  setSelectedId(b.id)
                }}
                className="flex flex-col items-center gap-1.5 rounded-[10px] border bg-background px-2 py-3 hover:border-primary hover:bg-primary-weak"
              >
                <Icon className="size-[19px] text-primary" aria-hidden="true" />
                <span className="text-xs font-semibold">{label}</span>
              </button>
            ))}
          </div>
          <div className="mt-4.5 rounded-xl border border-primary-weak bg-[linear-gradient(140deg,var(--primary-weak),var(--purple-weak))] p-3.5">
            <div className="mb-1.5 flex items-center gap-1.5 text-sm font-bold text-primary-text">
              <Sparkles className="size-3.5" aria-hidden="true" /> AI layouts
            </div>
            <p className="mb-2.5 text-xs text-muted-foreground">Describe your email and let AI assemble the blocks.</p>
            <Button type="button" className="w-full" size="sm">Generate</Button>
          </div>
        </div>

        {/* Canvas */}
        <div className="flex justify-center overflow-y-auto bg-surface-3 px-5 py-7">
          <div
            className="min-h-[400px] w-full overflow-hidden rounded-xl bg-white shadow-[var(--shadow)] transition-[max-width] duration-200"
            style={{ maxWidth: device === 'mobile' ? 375 : 600 }}
          >
            <BlockCanvas
              blocks={blocks}
              selectedId={selectedId}
              onSelect={setSelectedId}
              onDelete={(bid) => { setBlocks((cur) => removeBlock(cur, bid)); if (selectedId === bid) setSelectedId(null) }}
              onReorder={setBlocks}
            />
          </div>
        </div>

        {/* Inspector */}
        <div className="overflow-y-auto border-l bg-surface p-4">
          <BlockInspector
            block={selected}
            onChange={(patch) => selected && setBlocks((cur) => updateBlock(cur, selected.id, patch))}
            onDelete={selected ? () => {
              setBlocks((cur) => removeBlock(cur, selected.id))
              setSelectedId(null)
            } : undefined}
          />
        </div>
      </div>

      {showPreview && (
        <div className="fixed inset-0 z-50 flex flex-col bg-black/60 p-6" onClick={() => setShowPreview(false)}>
          <div className="mx-auto flex h-full w-full max-w-[640px] flex-col overflow-hidden rounded-xl bg-white" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between border-b px-4 py-2.5">
              <span className="text-sm font-semibold text-slate-700">Preview · {subject || 'Untitled'}</span>
              <button type="button" aria-label="Close preview" onClick={() => setShowPreview(false)} className="text-slate-500 hover:text-slate-900">
                <X className="size-4" aria-hidden="true" />
              </button>
            </div>
            <iframe title="preview" srcDoc={previewHtml} className="flex-1" sandbox="" />
          </div>
        </div>
      )}
    </div>
  )
}
