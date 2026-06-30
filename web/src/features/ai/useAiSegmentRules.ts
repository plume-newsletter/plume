import { useMutation } from '@tanstack/react-query'
import { api } from '@/lib/api'
import type { Condition } from '@/features/segments/useSegments'

export type SegmentRules = { match: 'all' | 'any'; conditions: Condition[]; count: number }

export function useAiSegmentRules() {
  return useMutation({
    mutationFn: (prompt: string) =>
      api<SegmentRules>('/api/ai/segment-rules', {
        method: 'POST',
        body: JSON.stringify({ prompt }),
      }),
  })
}
