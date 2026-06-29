import { Sparkles, AlignLeft, AlignCenter, AlignRight, MousePointerClick } from 'lucide-react'
import type { Block, SocialItem } from './blocks'
import { useRewrite, type RewriteAction } from './useRewrite'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

const ALIGNABLE = new Set(['heading', 'text', 'button'])
const AI_ABLE = new Set(['heading', 'text'])

const fieldClass =
  'w-full rounded-lg border bg-background px-3 py-2 text-sm text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-primary-weak'

// label wraps its control so the field is accessible by its label text.
function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="block">
      <span className="mb-1.5 block text-xs font-semibold">{label}</span>
      {children}
    </label>
  )
}

function AlignControl({ value, onChange }: { value: Block['align']; onChange: (a: Block['align']) => void }) {
  const opts: { a: NonNullable<Block['align']>; Icon: typeof AlignLeft }[] = [
    { a: 'left', Icon: AlignLeft },
    { a: 'center', Icon: AlignCenter },
    { a: 'right', Icon: AlignRight },
  ]
  return (
    <div>
      <span className="mb-1.5 block text-xs font-semibold">Alignment</span>
      <div className="flex gap-1 rounded-lg bg-surface-2 p-[3px]">
        {opts.map(({ a, Icon }) => (
          <button
            key={a}
            type="button"
            aria-label={`Align ${a}`}
            aria-pressed={(value ?? 'left') === a}
            onClick={() => onChange(a)}
            className={cn(
              'flex flex-1 justify-center rounded-md py-1.5',
              (value ?? 'left') === a ? 'bg-card shadow-sm' : 'text-muted-foreground',
            )}
          >
            <Icon className="size-4" aria-hidden="true" />
          </button>
        ))}
      </div>
    </div>
  )
}

function AiAssist({ current, onResult }: { current: string; onResult: (text: string) => void }) {
  const rewrite = useRewrite()
  const run = (action: RewriteAction) =>
    rewrite.mutate({ action, text: current }, { onSuccess: ({ text }) => onResult(text) })
  return (
    <div className="rounded-xl border border-primary-weak bg-[linear-gradient(140deg,var(--primary-weak),var(--purple-weak))] p-3.5">
      <div className="mb-2 flex items-center gap-1.5 text-sm font-bold text-primary-text">
        <Sparkles className="size-3.5" aria-hidden="true" /> AI writing assist
      </div>
      <Button type="button" className="w-full" disabled={rewrite.isPending} onClick={() => run('rewrite')}>
        {rewrite.isPending ? 'Writing…' : 'Rewrite with AI'}
      </Button>
      <div className="mt-2 flex gap-1.5">
        <button
          type="button"
          disabled={rewrite.isPending}
          onClick={() => run('shorten')}
          className="flex-1 rounded-md border border-primary-weak bg-card py-1.5 text-xs font-semibold text-primary-text hover:bg-surface-2"
        >
          Shorten
        </button>
        <button
          type="button"
          disabled={rewrite.isPending}
          onClick={() => run('more_casual')}
          className="flex-1 rounded-md border border-primary-weak bg-card py-1.5 text-xs font-semibold text-primary-text hover:bg-surface-2"
        >
          More casual
        </button>
      </div>
    </div>
  )
}

function Fields({ block, onChange }: { block: Block; onChange: (patch: Partial<Block>) => void }) {
  switch (block.type) {
    case 'heading':
      return (
        <Field label="Text">
          <textarea id="ins-text" rows={3} className={cn(fieldClass, 'resize-y')}
            value={block.text ?? ''} onChange={(e) => onChange({ text: e.target.value })} />
        </Field>
      )
    case 'text':
      return (
        <Field label="Text">
          <textarea id="ins-html" rows={5} className={cn(fieldClass, 'resize-y')}
            value={block.html ?? ''} onChange={(e) => onChange({ html: e.target.value })} />
        </Field>
      )
    case 'html':
      return (
        <Field label="Raw HTML">
          <textarea id="ins-html" rows={6} className={cn(fieldClass, 'resize-y font-mono')}
            value={block.html ?? ''} onChange={(e) => onChange({ html: e.target.value })} />
        </Field>
      )
    case 'button':
      return (
        <div className="space-y-3">
          <Field label="Label">
            <input className={fieldClass} value={block.label ?? ''} onChange={(e) => onChange({ label: e.target.value })} />
          </Field>
          <Field label="Link URL">
            <input className={cn(fieldClass, 'font-mono')} value={block.href ?? ''} onChange={(e) => onChange({ href: e.target.value })} />
          </Field>
        </div>
      )
    case 'image':
      return (
        <div className="space-y-3">
          <Field label="Image URL">
            <input className={cn(fieldClass, 'font-mono')} value={block.src ?? ''} onChange={(e) => onChange({ src: e.target.value })} />
          </Field>
          <Field label="Alt text">
            <input className={fieldClass} value={block.alt ?? ''} onChange={(e) => onChange({ alt: e.target.value })} />
          </Field>
        </div>
      )
    case 'spacer':
      return (
        <Field label="Height (px)">
          <input type="number" className={fieldClass} value={block.height ?? 16}
            onChange={(e) => onChange({ height: Number(e.target.value) })} />
        </Field>
      )
    case 'social':
      return (
        <div className="space-y-2">
          <span className="block text-xs font-semibold">Social links</span>
          {(block.items ?? []).map((it, i) => (
            <div key={i} className="flex gap-1.5">
              <input aria-label={`platform ${i}`} placeholder="platform" className={fieldClass} value={it.platform}
                onChange={(e) => {
                  const items = [...(block.items ?? [])]
                  items[i] = { ...items[i], platform: e.target.value }
                  onChange({ items })
                }} />
              <input aria-label={`url ${i}`} placeholder="https://…" className={fieldClass} value={it.url}
                onChange={(e) => {
                  const items = [...(block.items ?? [])]
                  items[i] = { ...items[i], url: e.target.value }
                  onChange({ items })
                }} />
            </div>
          ))}
          <Button type="button" variant="outline" size="sm"
            onClick={() => onChange({ items: [...(block.items ?? []), { platform: '', url: '' } as SocialItem] })}>
            Add link
          </Button>
        </div>
      )
    case 'divider':
      return <p className="text-sm text-muted-foreground">A horizontal divider line. Use spacing to adjust padding.</p>
    case 'columns':
      // ponytail: columns store child Block[]; per-column editing lands with nested blocks.
      return <p className="text-sm text-muted-foreground">Two-column block. Per-column content editing is coming.</p>
    default:
      return null
  }
}

export function BlockInspector({
  block, onChange, onDelete,
}: {
  block: Block | null
  onChange: (patch: Partial<Block>) => void
  onDelete?: () => void
}) {
  if (!block) {
    return (
      <div className="px-4 py-10 text-center text-muted-foreground">
        <MousePointerClick className="mx-auto mb-2.5 size-7 text-faint" aria-hidden="true" />
        <p className="text-sm">Select a block to edit its content and style.</p>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <span className="text-xs font-bold uppercase tracking-wide text-faint">{block.type} block</span>
        {onDelete && (
          <button type="button" onClick={onDelete} className="text-xs font-semibold text-danger hover:underline">
            Delete
          </button>
        )}
      </div>

      <Fields block={block} onChange={onChange} />

      {ALIGNABLE.has(block.type) && (
        <AlignControl value={block.align} onChange={(align) => onChange({ align })} />
      )}

      {AI_ABLE.has(block.type) && (
        <AiAssist
          current={block.type === 'heading' ? block.text ?? '' : block.html ?? ''}
          onResult={(text) => onChange(block.type === 'heading' ? { text } : { html: text })}
        />
      )}
    </div>
  )
}
