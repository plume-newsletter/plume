import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api'

export type Insight = { strong?: string; rest: string; muted?: string }

// Insights call the LLM, so cache them for the session and don't auto-retry —
// a missing AI key (400) or model hiccup just hides the panel.
export function useAiInsights() {
  return useQuery({
    queryKey: ['ai-insights'],
    queryFn: () => api<{ insights: Insight[] }>('/api/ai/insights', { method: 'POST' }),
    staleTime: 30 * 60 * 1000,
    gcTime: 60 * 60 * 1000,
    retry: false,
    refetchOnWindowFocus: false,
  })
}
