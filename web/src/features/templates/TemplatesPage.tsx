import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { toast } from 'sonner'
import { useTemplates, useUseTemplate, useDeleteTemplate, type Template } from './useTemplates'
import { useBrands } from '@/features/brands/useBrands'
import { useCreateCampaign } from '@/features/campaigns/useCampaigns'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter,
} from '@/components/ui/dialog'
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from '@/components/ui/select'

const CATEGORIES = ['All', 'Newsletter', 'Product', 'Promo', 'Transactional'] as const

const schema = z.object({ subject: z.string().min(1) })
type Form = z.infer<typeof schema>

// Single controlled dialog for both "blank canvas" and "use template" flows
function PickBrandDialog({
  open,
  onOpenChange,
  templateId,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  templateId?: string
}) {
  const [brandId, setBrandId] = useState('')
  const { data: brands } = useBrands()
  const navigate = useNavigate()
  const useTemplate = useUseTemplate()
  const createCampaign = useCreateCampaign()
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<Form>({
    resolver: zodResolver(schema),
    defaultValues: { subject: '' },
  })

  const handleClose = () => {
    onOpenChange(false)
    reset()
    setBrandId('')
  }

  const onSubmit = (data: Form) => {
    if (!brandId) return
    if (templateId) {
      useTemplate.mutate(
        { id: templateId, brandId, subject: data.subject },
        {
          onSuccess: ({ campaignId }) => {
            handleClose()
            navigate('/campaigns/' + campaignId)
          },
          onError: () => toast.error('Something went wrong'),
        },
      )
    } else {
      createCampaign.mutate(
        { brandId, subject: data.subject, htmlBody: '', plainBody: '' },
        {
          onSuccess: (created) => {
            handleClose()
            toast.success('Campaign created')
            navigate('/campaigns/' + created.id)
          },
          onError: () => toast.error('Something went wrong'),
        },
      )
    }
  }

  const isPending = useTemplate.isPending || createCampaign.isPending

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{templateId ? 'Use template' : 'New campaign'}</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-3">
          <div>
            <Label htmlFor="tpl-brand">Brand</Label>
            <Select value={brandId} onValueChange={(v) => setBrandId(v ?? '')}>
              <SelectTrigger id="tpl-brand" className="w-full">
                <SelectValue placeholder="Select brand">
                  {(v) => brands?.find((b) => b.id === v)?.name ?? 'Select brand'}
                </SelectValue>
              </SelectTrigger>
              <SelectContent>
                {brands?.map((b) => (
                  <SelectItem key={b.id} value={b.id}>
                    {b.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div>
            <Label htmlFor="tpl-subject">Subject</Label>
            <Input id="tpl-subject" {...register('subject')} />
            {errors.subject && <p className="text-sm text-destructive">Required</p>}
          </div>
          <DialogFooter>
            <Button type="submit" disabled={!brandId || isPending}>
              {templateId ? 'Use template' : 'Create'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

function TemplateCard({
  tpl,
  onUse,
  onDelete,
}: {
  tpl: Template
  onUse: (tpl: Template) => void
  onDelete: (id: string) => void
}) {
  return (
    <div className="relative bg-surface border border-border rounded-[14px] overflow-hidden shadow-sm hover:shadow hover:-translate-y-0.5 transition group">
      {/* Card body button — opens use-template dialog */}
      <button
        type="button"
        className="w-full text-left cursor-pointer"
        aria-label={tpl.name}
        onClick={() => onUse(tpl)}
      >
        {/* Thumbnail */}
        <div
          className="h-[150px] p-4 flex flex-col gap-[7px]"
          style={{ background: thumbBg(tpl) }}
        >
          <div style={{ height: 18, width: '60%', background: 'rgba(255,255,255,.85)', borderRadius: 4 }} />
          <div style={{ height: 36, background: 'rgba(255,255,255,.55)', borderRadius: 4 }} />
          <div style={{ height: 8, width: '90%', background: 'rgba(255,255,255,.55)', borderRadius: 4 }} />
          <div style={{ height: 8, width: '75%', background: 'rgba(255,255,255,.55)', borderRadius: 4 }} />
          <div
            style={{
              marginTop: 'auto',
              height: 22,
              width: 80,
              background: 'rgba(255,255,255,.9)',
              borderRadius: 4,
            }}
          />
        </div>
        {/* Footer */}
        <div className="px-[15px] py-[13px]">
          <div className="font-semibold text-[.9rem]">{tpl.name}</div>
          <div className="text-[.76rem] text-muted-foreground">{tpl.category}</div>
        </div>
      </button>

      {/* Delete button — only for user templates */}
      {!tpl.prebuilt && (
        <button
          type="button"
          aria-label="Delete template"
          onClick={(e) => {
            e.stopPropagation()
            onDelete(tpl.id)
          }}
          className="absolute top-2 right-2 hidden group-hover:flex items-center justify-center size-7 rounded-md bg-black/40 text-white hover:bg-red-500 transition"
        >
          ✕
        </button>
      )}
    </div>
  )
}

export function TemplatesPage() {
  const [category, setCategory] = useState('All')
  const { data: templates } = useTemplates(category)
  const deleteTemplate = useDeleteTemplate()

  // Controlled dialog state
  const [pickOpen, setPickOpen] = useState(false)
  const [selectedId, setSelectedId] = useState<string | undefined>(undefined)

  const handleCardUse = (tpl: Template) => {
    setSelectedId(tpl.id)
    setPickOpen(true)
  }

  const handleBlankCanvas = () => {
    setSelectedId(undefined)
    setPickOpen(true)
  }

  const handleDelete = (id: string) => {
    deleteTemplate.mutate(id, {
      onSuccess: () => toast.success('Template deleted'),
      onError: () => toast.error('Could not delete template'),
    })
  }

  return (
    <div className="space-y-5">
      {/* Header row */}
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-[1.55rem] font-bold tracking-tight">Templates</h1>
          <p className="mt-1 text-muted-foreground">
            Start fast with prebuilt layouts or your saved designs.
          </p>
        </div>
        <Button onClick={handleBlankCanvas}>Blank canvas</Button>
      </div>

      {/* Category chips */}
      <div className="flex gap-2 mb-4 flex-wrap">
        {CATEGORIES.map((cat) => (
          <button
            key={cat}
            type="button"
            onClick={() => setCategory(cat)}
            className={
              cat === category
                ? 'rounded-full border border-primary bg-primary-weak text-primary-text font-semibold px-[13px] py-[6px] text-[.8rem]'
                : 'rounded-full border border-border bg-surface text-muted-foreground font-medium px-[13px] py-[6px] text-[.8rem]'
            }
          >
            {cat}
          </button>
        ))}
      </div>

      {/* Template grid */}
      <div
        className="grid gap-4"
        style={{ gridTemplateColumns: 'repeat(auto-fill, minmax(220px, 1fr))' }}
      >
        {(templates ?? []).map((tpl) => (
          <TemplateCard key={tpl.id} tpl={tpl} onUse={handleCardUse} onDelete={handleDelete} />
        ))}
      </div>

      {/* Single controlled dialog for use-template and blank-canvas flows */}
      <PickBrandDialog
        open={pickOpen}
        onOpenChange={setPickOpen}
        templateId={selectedId}
      />
    </div>
  )
}

// ── helpers ──────────────────────────────────────────────────────────────────

const GRADIENTS = [
  'linear-gradient(135deg,#6366f1,#818cf8)',
  'linear-gradient(135deg,#10b981,#34d399)',
  'linear-gradient(135deg,#f59e0b,#fbbf24)',
]

function thumbBg(tpl: Template): string {
  if (tpl.prebuilt) {
    const cat = tpl.category.toLowerCase()
    if (cat === 'newsletter') return GRADIENTS[0]
    if (cat.startsWith('product')) return GRADIENTS[1]
    if (cat === 'promo') return GRADIENTS[2]
  }
  // Stable hash of the id → pick a gradient
  let hash = 0
  for (const ch of tpl.id) hash = (hash * 31 + ch.charCodeAt(0)) & 0xffffff
  return GRADIENTS[hash % GRADIENTS.length]
}
