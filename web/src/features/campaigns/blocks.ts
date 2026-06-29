export type BlockType =
  | 'heading' | 'text' | 'image' | 'button'
  | 'divider' | 'spacer' | 'social' | 'columns' | 'html'

export type SocialItem = { platform: string; url: string }

export type Block = {
  id: string
  type: BlockType
  text?: string
  level?: number
  html?: string
  src?: string
  alt?: string
  href?: string
  label?: string
  align?: 'left' | 'center' | 'right'
  height?: number
  items?: SocialItem[]
  left?: Block[]
  right?: Block[]
}

function id() {
  return Math.random().toString(36).slice(2, 10)
}

export function newBlock(type: BlockType): Block {
  switch (type) {
    case 'heading': return { id: id(), type, text: 'Heading', level: 2 }
    case 'text': return { id: id(), type, html: 'Write something…' }
    case 'image': return { id: id(), type, src: '', alt: '' }
    case 'button': return { id: id(), type, label: 'Click', href: '', align: 'left' }
    case 'divider': return { id: id(), type }
    case 'spacer': return { id: id(), type, height: 16 }
    case 'social': return { id: id(), type, items: [] }
    case 'columns': return { id: id(), type, left: [], right: [] }
    case 'html': return { id: id(), type, html: '<p>Raw HTML</p>' }
  }
}

export function addBlock(list: Block[], block: Block): Block[] {
  return [...list, block]
}

export function moveBlock(list: Block[], from: number, to: number): Block[] {
  const next = [...list]
  const [item] = next.splice(from, 1)
  next.splice(to, 0, item)
  return next
}

export function removeBlock(list: Block[], blockId: string): Block[] {
  return list.filter((b) => b.id !== blockId)
}

export function updateBlock(list: Block[], blockId: string, patch: Partial<Block>): Block[] {
  return list.map((b) => (b.id === blockId ? { ...b, ...patch } : b))
}
