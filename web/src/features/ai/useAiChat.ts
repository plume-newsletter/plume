import { useMutation } from '@tanstack/react-query'
import { api } from '@/lib/api'

export type ChatMessage = { role: 'user' | 'assistant'; content: string }

export function useAiChat() {
  return useMutation({
    mutationFn: (messages: ChatMessage[]) =>
      api<{ reply: string }>('/api/ai/chat', {
        method: 'POST',
        body: JSON.stringify({ messages }),
      }),
  })
}
