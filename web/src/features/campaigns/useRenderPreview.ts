import { useMutation } from '@tanstack/react-query'
import { api } from '@/lib/api'
import type { Block } from './blocks'

export function useRenderPreview() {
  return useMutation({
    mutationFn: (blocks: Block[]) =>
      api<{ html: string }>('/api/blocks/render', {
        method: 'POST',
        body: JSON.stringify({ blocks }),
      }),
  })
}
