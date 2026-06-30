import { useMutation } from '@tanstack/react-query'
import { api } from '@/lib/api'

export function useAiSuggest() {
  return useMutation({
    mutationFn: (input: { kind: string; context: string }) =>
      api<{ options: string[] }>('/api/ai/suggest', {
        method: 'POST',
        body: JSON.stringify(input),
      }),
  })
}
