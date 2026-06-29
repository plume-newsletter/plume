import {
  DndContext, PointerSensor, useSensor, useSensors, closestCenter, type DragEndEvent,
} from '@dnd-kit/core'
import { SortableContext, useSortable, verticalListSortingStrategy } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { ChevronUp, ChevronDown, Trash2, ImageIcon } from 'lucide-react'
import type { Block } from './blocks'
import { moveBlock } from './blocks'
import { cn } from '@/lib/utils'

// The doc represents the real (always-light) email, so colors are hard-coded
// rather than themed — it should look like an email in light or dark app theme.
const HEAD_SIZE: Record<number, string> = { 1: '26px', 2: '22px', 3: '18px' }

function BlockBody({ block }: { block: Block }) {
  const align = block.align ?? 'left'
  switch (block.type) {
    case 'heading':
      return (
        <div style={{ textAlign: align, fontSize: HEAD_SIZE[block.level ?? 2] ?? '22px' }}
          className="py-1.5 font-bold leading-tight text-slate-900">
          {block.text}
        </div>
      )
    case 'text':
      return (
        <div style={{ textAlign: align }}
          className="py-1 text-[15px] leading-relaxed text-slate-700"
          dangerouslySetInnerHTML={{ __html: block.html ?? '' }} />
      )
    case 'button':
      return (
        <div className="flex py-2" style={{ justifyContent: align === 'center' ? 'center' : align === 'right' ? 'flex-end' : 'flex-start' }}>
          <span className="rounded-lg bg-[#1E40AF] px-6 py-2.5 text-sm font-semibold text-white">{block.label || 'Button'}</span>
        </div>
      )
    case 'image':
      return block.src ? (
        <img src={block.src} alt={block.alt ?? ''} className="w-full rounded-lg" />
      ) : (
        <div className="flex h-32 items-center justify-center gap-2 rounded-lg bg-[linear-gradient(135deg,#dbeafe,#ede9fe)] text-[13px] text-slate-500">
          <ImageIcon className="size-5" aria-hidden="true" />
          {block.alt || 'Image'}
        </div>
      )
    case 'divider':
      return <div className="my-2 h-px bg-slate-200" />
    case 'spacer':
      return <div style={{ height: `${block.height ?? 16}px` }} />
    case 'columns':
      return (
        <div className="grid grid-cols-2 gap-3.5 py-1.5 text-sm leading-relaxed text-slate-700">
          <div>Left column</div>
          <div>Right column</div>
        </div>
      )
    case 'social':
      return (
        <div className="flex justify-center gap-3 py-2">
          {[0, 1, 2].map((i) => <span key={i} className="size-[34px] rounded-full bg-[#1E40AF]" />)}
        </div>
      )
    case 'html':
      return <div className="py-1 text-[15px] leading-relaxed text-slate-700" dangerouslySetInnerHTML={{ __html: block.html ?? '' }} />
    default:
      return null
  }
}

function CanvasBlock({
  block, selected, first, last, onSelect, onMove, onDelete,
}: {
  block: Block
  selected: boolean
  first: boolean
  last: boolean
  onSelect: () => void
  onMove: (dir: 'up' | 'down') => void
  onDelete: () => void
}) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({ id: block.id })
  const style = { transform: CSS.Transform.toString(transform), transition }
  return (
    <div
      ref={setNodeRef}
      style={style}
      onClick={onSelect}
      {...attributes}
      {...listeners}
      className={cn(
        'relative cursor-pointer px-3.5 py-1.5 outline-offset-[-2px]',
        selected && 'outline outline-2 outline-primary',
        isDragging && 'opacity-60',
      )}
    >
      {selected && (
        <div className="absolute -top-px right-1.5 z-10 flex gap-0.5 rounded-md bg-primary p-0.5">
          <button type="button" aria-label="Move up" disabled={first}
            onClick={(e) => { e.stopPropagation(); onMove('up') }}
            className="flex size-[22px] items-center justify-center rounded text-white hover:bg-white/20 disabled:opacity-40">
            <ChevronUp className="size-3.5" aria-hidden="true" />
          </button>
          <button type="button" aria-label="Move down" disabled={last}
            onClick={(e) => { e.stopPropagation(); onMove('down') }}
            className="flex size-[22px] items-center justify-center rounded text-white hover:bg-white/20 disabled:opacity-40">
            <ChevronDown className="size-3.5" aria-hidden="true" />
          </button>
          <button type="button" aria-label="Delete block"
            onClick={(e) => { e.stopPropagation(); onDelete() }}
            className="flex size-[22px] items-center justify-center rounded text-white hover:bg-white/25">
            <Trash2 className="size-3.5" aria-hidden="true" />
          </button>
        </div>
      )}
      <BlockBody block={block} />
    </div>
  )
}

export function BlockCanvas({
  blocks, selectedId, onSelect, onDelete, onReorder,
}: {
  blocks: Block[]
  selectedId: string | null
  onSelect: (id: string) => void
  onDelete: (id: string) => void
  onReorder: (next: Block[]) => void
}) {
  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 6 } }))

  const handleDragEnd = (e: DragEndEvent) => {
    const { active, over } = e
    if (!over || active.id === over.id) return
    const from = blocks.findIndex((b) => b.id === active.id)
    const to = blocks.findIndex((b) => b.id === over.id)
    if (from >= 0 && to >= 0) onReorder(moveBlock(blocks, from, to))
  }

  const move = (id: string, dir: 'up' | 'down') => {
    const from = blocks.findIndex((b) => b.id === id)
    const to = dir === 'up' ? from - 1 : from + 1
    if (to < 0 || to >= blocks.length) return
    onReorder(moveBlock(blocks, from, to))
  }

  return (
    <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
      <SortableContext items={blocks.map((b) => b.id)} strategy={verticalListSortingStrategy}>
        {blocks.map((b, i) => (
          <CanvasBlock
            key={b.id}
            block={b}
            selected={b.id === selectedId}
            first={i === 0}
            last={i === blocks.length - 1}
            onSelect={() => onSelect(b.id)}
            onMove={(dir) => move(b.id, dir)}
            onDelete={() => onDelete(b.id)}
          />
        ))}
      </SortableContext>
      <div className="m-3.5 rounded-[10px] border-2 border-dashed border-border-strong p-5 text-center text-[13px] text-slate-400">
        Drag a block here, or click one on the left
      </div>
    </DndContext>
  )
}
