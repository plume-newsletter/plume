import { useMutation } from '@tanstack/react-query'
import { api } from '@/lib/api'

export type RewriteAction = 'rewrite' | 'shorten' | 'more_casual'

export function useRewrite() {
  return useMutation({
    mutationFn: (input: { action: RewriteAction; text: string }) =>
      api<{ text: string }>('/api/ai/rewrite', {
        method: 'POST',
        body: JSON.stringify(input),
      }),
  })
}
